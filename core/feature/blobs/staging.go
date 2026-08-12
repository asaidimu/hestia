package blobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/asaidimu/blobs/staging"

	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/core/abstract"
	blobutil "github.com/asaidimu/hestia/core/feature/blobs/store"
	"github.com/asaidimu/hestia/core/internal/util"
	"github.com/asaidimu/hestia/core/runtime"
)

const (
	// defaultUploadBlockSize is used when the client does not request a specific
	// block size in the begin request. It balances per-chunk HTTP overhead
	// against the cost of re-uploading a chunk after a dropped connection.
	defaultUploadBlockSize = 8 << 20 // 8 MiB

	// maxUploadChunkBytes bounds a single chunk request body. Set above
	// defaultUploadBlockSize so a client-supplied custom block size still has
	// headroom.
	maxUploadChunkBytes = 256 << 20 // 256 MiB

	// maxDirectUploadBytes bounds the single-shot upload handler. Larger blobs
	// must use the resumable (staged) upload protocol.
	maxDirectUploadBytes = 16 << 20 // 16 MiB
)

// NewBeginUploadHandler starts a resumable upload session. The key is chosen
// by the caller and must not already exist unless ?overwrite=true.
func NewBeginUploadHandler(svc blobutil.BlobStore, mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		body, _ := msg.Input().GetOr("payload", nil).(map[string]any)

		key, _ := body["key"].(string)
		if key == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "payload.key is required")
		}
		size, _ := body["size"].(float64)
		if size <= 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "payload.size must be greater than 0")
		}
		contentType, _ := body["content_type"].(string)

		blockSize := int64(0)
		if bs, ok := body["block_size"].(float64); ok && bs > 0 {
			blockSize = int64(bs)
		}
		if blockSize < 0 || blockSize > int64(size) {
			return nil, common.NewSystemError("VALIDATION_ERROR", "payload.block_size must be between 0 and the file size")
		}

		overwrite := msg.Input().GetOr("modifiers.overwrite", "") == "true"
		if err := ensureKeyWritable(ctx, svc, nsID, key, overwrite); err != nil {
			return nil, err
		}

		if blockSize == 0 {
			blockSize = defaultUploadBlockSize
			if blockSize > int64(size) {
				blockSize = int64(size)
			}
		}

		sess, err := mgr.Begin(ctx, nsID, key, staging.BeginOptions{
			ContentType:  contentType,
			ExpectedSize: int64(size),
			BlockSize:    blockSize,
		})
		if err != nil {
			return nil, fmt.Errorf("begin upload: %w", err)
		}

		return &abstract.Result{
			Document: mustDoc(map[string]any{
				"session_id": sess.ID,
				"key":        sess.Key,
				"offset":     sess.Offset,
				"block_size": blockSize,
			}, ctx),
		}, nil
	}
}

// NewUploadChunkHandler writes a chunk of a staged upload. The chunk's raw
// bytes are the request payload; its logical offset and optional integrity
// check travel as headers.
func NewUploadChunkHandler(mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		sessionID, _ := msg.Input().GetOr("headers.session_id", "").(string)
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Session-ID header is required")
		}
		offsetStr, _ := msg.Input().GetOr("headers.offset", "").(string)
		offset, err := strconv.ParseInt(offsetStr, 10, 64)
		if err != nil || offset < 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Offset header must be a non-negative integer")
		}

		raw, _ := msg.Input().Get("payload")
		data, _ := raw.([]byte)
		if len(data) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "chunk request body is required")
		}
		if len(data) > maxUploadChunkBytes {
			return nil, common.NewSystemError("VALIDATION_ERROR",
				fmt.Sprintf("chunk exceeds %d byte limit", maxUploadChunkBytes))
		}

		expectedSHA, _ := msg.Input().GetOr("headers.sha256", "").(string)

		total, err := mgr.WriteChunk(ctx, sessionID, offset, bytes.NewReader(data), expectedSHA)
		if err != nil {
			return nil, mapStagingError(err)
		}

		return &abstract.Result{
			Document: mustDoc(map[string]any{"total": total}, ctx),
		}, nil
	}
}

// NewCompleteUploadHandler finalizes a staged upload and streams it straight
// into the blob store's chunking pipeline in a single pass.
func NewCompleteUploadHandler(svc blobutil.BlobStore, mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		sessionID, _ := msg.Input().GetOr("headers.session_id", "").(string)
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Session-ID header is required")
		}

		cu, err := mgr.Complete(ctx, sessionID)
		if err != nil {
			return nil, mapStagingError(err)
		}
		defer cu.Close()

		overwrite := msg.Input().GetOr("modifiers.overwrite", "") == "true"
		if !overwrite {
			if err := ensureKeyWritable(ctx, svc, cu.NamespaceID, cu.Key, false); err != nil {
				_ = mgr.Abort(sessionID)
				return nil, err
			}
		}

		meta, err := svc.Namespace(cu.NamespaceID).PutCustom(ctx, cu.Key, cu.ContentType, cu, cu.Custom)
		if err != nil {
			return nil, mapBlobError(err)
		}
		cu.Finalize()

		return &abstract.Result{
			Document: mustDoc(util.StructToMap(*meta), ctx),
		}, nil
	}
}

// NewProgressUploadHandler reports which byte ranges have already been
// received for a session, so an interrupted upload can resume.
func NewProgressUploadHandler(mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		sessionID, _ := msg.Input().GetOr("modifiers.session_id", "").(string)
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "?session_id= query parameter is required")
		}

		ranges, err := mgr.Ranges(sessionID)
		if err != nil {
			return nil, mapStagingError(err)
		}
		blockSize, _ := mgr.BlockSize(sessionID)
		expectedSize, _ := mgr.ExpectedSize(sessionID)

		var total int64
		items := make([]map[string]any, len(ranges))
		for i, r := range ranges {
			items[i] = map[string]any{"start": r.Start, "end": r.End}
			total += r.End - r.Start
		}

		return &abstract.Result{
			Document: mustDoc(map[string]any{
				"total":         total,
				"ranges":        items,
				"block_size":    blockSize,
				"expected_size": expectedSize,
			}, ctx),
		}, nil
	}
}

// NewAbortUploadHandler discards a staged upload's data and metadata.
func NewAbortUploadHandler(mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		sessionID, _ := msg.Input().GetOr("headers.session_id", "").(string)
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Session-ID header is required")
		}
		if err := mgr.Abort(sessionID); err != nil {
			return nil, mapStagingError(err)
		}
		return &abstract.Result{}, nil
	}
}

func mapStagingError(err error) error {
	switch {
	case errors.Is(err, staging.ErrSessionNotFound),
		errors.Is(err, staging.ErrInvalidSessionID):
		return runtime.ErrNotFound.WithCause(err)
	case errors.Is(err, staging.ErrSizeMismatch),
		errors.Is(err, staging.ErrIncompleteUpload),
		errors.Is(err, staging.ErrBlockAlignment),
		errors.Is(err, staging.ErrPieceCountMismatch),
		errors.Is(err, staging.ErrPieceHashMismatch),
		errors.Is(err, staging.ErrChecksumMismatch):
		return common.NewSystemError("INVALID_REQUEST", err.Error()).WithCause(err)
	default:
		return fmt.Errorf("staging: %w", err)
	}
}
