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
		var input BlobBeginInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		key := input.Key
		if key == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "payload.key is required")
		}
		size := input.Size
		if size <= 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "payload.size must be greater than 0")
		}
		contentType := input.ContentType

		blockSize := input.BlockSize
		if blockSize < 0 || blockSize > size {
			return nil, common.NewSystemError("VALIDATION_ERROR", "payload.block_size must be between 0 and the file size")
		}

		if err := ensureKeyWritable(ctx, svc, input.NS, key, input.Overwrite == "true"); err != nil {
			return nil, err
		}

		if blockSize == 0 {
			blockSize = defaultUploadBlockSize
			if blockSize > size {
				blockSize = size
			}
		}

		sess, err := mgr.Begin(ctx, input.NS, key, staging.BeginOptions{
			ContentType:  contentType,
			ExpectedSize: size,
			BlockSize:    blockSize,
		})
		if err != nil {
			return nil, fmt.Errorf("begin upload: %w", err)
		}

		return &abstract.Result{
			Document: newDoc(ctx, &UploadBeginDocument{
				SessionID: sess.ID,
				Key:       sess.Key,
				Offset:    sess.Offset,
				BlockSize: blockSize,
			}),
		}, nil
	}
}

// NewUploadChunkHandler writes a chunk of a staged upload. The chunk's raw
// bytes are the request payload; its logical offset and optional integrity
// check travel as headers.
func NewUploadChunkHandler(mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobChunkInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		payload := append([]byte(nil), input.Payload...)
		msg.Input().Release()

		sessionID := input.SessionID
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Session-ID header is required")
		}
		offset, err := strconv.ParseInt(input.Offset, 10, 64)
		if err != nil || offset < 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Offset header must be a non-negative integer")
		}

		if len(payload) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "chunk request body is required")
		}
		if len(payload) > maxUploadChunkBytes {
			return nil, common.NewSystemError("VALIDATION_ERROR",
				fmt.Sprintf("chunk exceeds %d byte limit", maxUploadChunkBytes))
		}

		total, err := mgr.WriteChunk(ctx, sessionID, offset, bytes.NewReader(payload), input.SHA256)
		if err != nil {
			return nil, mapStagingError(err)
		}

		return &abstract.Result{
			Document: newDoc(ctx, &UploadChunkDocument{Total: total}),
		}, nil
	}
}

// NewCompleteUploadHandler finalizes a staged upload and streams it straight
// into the blob store's chunking pipeline in a single pass.
func NewCompleteUploadHandler(svc blobutil.BlobStore, mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobCompleteInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		sessionID := input.SessionID
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "X-Session-ID header is required")
		}

		cu, err := mgr.Complete(ctx, sessionID)
		if err != nil {
			return nil, mapStagingError(err)
		}
		defer cu.Close()

		overwrite := input.Overwrite == "true"
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

		view := blobMetaView(meta)
		return &abstract.Result{Document: newDoc(ctx, &view)}, nil
	}
}

// NewProgressUploadHandler reports which byte ranges have already been
// received for a session, so an interrupted upload can resume.
func NewProgressUploadHandler(mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobProgressInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		sessionID := input.SessionID
		if sessionID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "?session_id= query parameter is required")
		}

		ranges, err := mgr.Ranges(sessionID)
		if err != nil {
			return nil, mapStagingError(err)
		}
		blockSize, _ := mgr.BlockSize(sessionID)
		expectedSize, _ := mgr.ExpectedSize(sessionID)

		items := make([]ByteRange, len(ranges))
		var total int64
		for i, r := range ranges {
			items[i] = ByteRange{Start: r.Start, End: r.End}
			total += r.End - r.Start
		}

		return &abstract.Result{
			Document: newDoc(ctx, &UploadProgressDocument{
				Total:        total,
				Ranges:       items,
				BlockSize:    blockSize,
				ExpectedSize: expectedSize,
			}),
		}, nil
	}
}

// NewAbortUploadHandler discards a staged upload's data and metadata.
func NewAbortUploadHandler(mgr *staging.Manager) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobAbortInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		sessionID := input.SessionID
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
