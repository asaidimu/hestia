// @note #cruft-20260821-020 observation resolved status=open priority=P2 tags=#cruft,#note : Old-style handler functions in blobs/handler.go
// @see #8uuufn
// No action needed — dynamically registered via RegisterBlobHandlers, cannot use static codegen.
//
// This file contains NewListNamespacesHandler, NewCreateNamespaceHandler,
// NewDeleteNamespaceHandler, NewListBlobsHandler, NewHeadBlobHandler,
// NewUploadBlobHandler, NewDownloadBlobHandler, NewUpdateBlobHandler,
// NewDeleteBlobHandler, and other blob handlers — all using the old pattern
// of returning abstract.MessageHandler directly with manual input extraction.
//
// Unlike other packages, these handlers are NOT superseded by generated
// registrations. They are registered dynamically via RegisterBlobHandlers
// when a new namespace is created, which is a runtime operation that cannot
// be handled by static registrations. These are legitimate old-style handlers
// that serve a different purpose.
//
// Resolution: no action needed — these handlers are dynamically registered
// and cannot be replaced by generated registrations.
package blobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"

	bserrors "github.com/asaidimu/blobs/errors"
	"github.com/asaidimu/blobs/staging"

	"github.com/asaidimu/hestia/core/abstract"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	"github.com/asaidimu/hestia/core/system/blobs/model"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func NewListNamespacesHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		namespaces, err := svc.ListNamespaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("list namespaces: %w", err)
		}

		views := make([]NamespaceView, len(namespaces))
		for i, ns := range namespaces {
			views[i] = NamespaceView{ID: ns.ID, DisplayName: ns.DisplayName, Public: ns.Public}
		}

		return dispatch.NewDocumentResultFrom(&NamespaceListDocument{Namespaces: views})
	}
}

func NewCreateNamespaceHandler(svc blobutil.BlobStore, mgr *staging.Manager, policyOp abstract.BindingPolicyStore, registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsCreateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		nsID := input.NS
		if nsID == "" {
			nsID = input.DisplayName
		}
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		var opts []blobutil.NamespaceOption
		if input.Public {
			opts = append(opts, blobutil.WithPublic(true))
		}

		if err := svc.CreateNamespace(ctx, nsID, input.DisplayName, opts...); err != nil {
			return nil, fmt.Errorf("create namespace: %w", err)
		}

		if err := SeedNamespaceBindings(ctx, policyOp, nsID); err != nil {
			return nil, fmt.Errorf("seed namespace operations: %w", err)
		}

		if err := RegisterBlobHandlers(registry, svc, mgr, nsID); err != nil {
			return nil, fmt.Errorf("register blob handlers: %w", err)
		}

		return dispatch.NewDocumentResultFrom(&NamespaceView{ID: nsID, DisplayName: input.DisplayName, Public: input.Public})
	}
}

func NewDeleteNamespaceHandler(svc blobutil.BlobStore, policyOp abstract.BindingPolicyStore, registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		nsID := input.NS
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		UnregisterBlobHandlers(registry, nsID)

		for _, op := range blobOps {
			opName := "system:blobs:" + nsID + ":" + op.Suffix
			if err := policyOp.DeleteBinding(ctx, opName); err != nil {
				return nil, fmt.Errorf("delete operation %s: %w", opName, err)
			}
		}

		if err := policyOp.ReloadPolicies(ctx); err != nil {
			return nil, fmt.Errorf("reload policies: %w", err)
		}

		if err := svc.DeleteNamespace(ctx, nsID); err != nil {
			return nil, fmt.Errorf("delete namespace: %w", err)
		}

		return &abstract.Result{}, nil
	}
}

func NewListBlobsHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobListInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if input.NS == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		blobs, err := svc.Namespace(input.NS).List(ctx, input.Prefix, input.Limit)
		if err != nil {
			return nil, fmt.Errorf("list blobs: %w", err)
		}

		items := make([]BlobMetaView, len(blobs))
		for i := range blobs {
			items[i] = blobMetaView(&blobs[i])
		}

		return dispatch.NewDocumentResultFrom(&BlobListDocument{Blobs: items})
	}
}

func NewHeadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobKeyInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		meta, err := svc.Namespace(input.NS).Head(ctx, input.Key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := blobMetaView(meta)
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewUploadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobUploadInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		payload := append([]byte(nil), input.Payload...)
		msg.Input().Release()

		if len(payload) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "request body is required")
		}
		if len(payload) > maxDirectUploadBytes {
			return nil, common.NewSystemError("VALIDATION_ERROR",
				fmt.Sprintf("direct upload exceeds %d byte limit; use the resumable upload protocol", maxDirectUploadBytes))
		}

		if err := ensureKeyWritable(ctx, svc, input.NS, input.Key, input.Overwrite == "true"); err != nil {
			return nil, err
		}

		meta, err := svc.Namespace(input.NS).Put(ctx, input.Key, input.ContentType, bytes.NewReader(payload))
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := blobMetaView(meta)
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewDownloadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobKeyInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		meta, err := svc.Namespace(input.NS).Head(ctx, input.Key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		rc, err := svc.Namespace(input.NS).Get(ctx, input.Key)
		if err != nil {
			return nil, mapBlobError(err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("read blob: %w", err)
		}

		return &abstract.Result{
			Blob: abstract.Blob{
				Data:        data,
				ContentType: meta.ContentType,
			},
		}, nil
	}
}

func NewUpdateBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobUpdateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		meta, err := svc.Namespace(input.NS).UpdateMetadata(ctx, input.Key, input.Custom)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := blobMetaView(meta)
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewDeleteBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobKeyInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if _, err := svc.Namespace(input.NS).Head(ctx, input.Key); err != nil {
			return nil, mapBlobError(err)
		}

		if err := svc.Namespace(input.NS).Delete(ctx, input.Key); err != nil {
			return nil, mapBlobError(err)
		}

		return &abstract.Result{}, nil
	}
}

func NewRenameBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobRenameInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if err := svc.Namespace(input.NS).Rename(ctx, input.Key, input.NewKey); err != nil {
			return nil, mapBlobError(err)
		}

		view := MessageOutput{Message: "renamed"}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewStatsNamespaceHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		stats, err := svc.Namespace(input.NS).Stats(ctx)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := NamespaceStatsDocument{
			NamespaceID:  stats.NamespaceID,
			BlobCount:    stats.BlobCount,
			BytesStored:  stats.BytesStored,
			BytesPhysical: stats.BytesPhysical,
			ChunkCount:   stats.ChunkCount,
			DeadBytes:    stats.DeadBytes,
			DeadChunks:   stats.DeadChunks,
			SegmentCount: stats.SegmentCount,
		}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewVerifyNamespaceHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if err := svc.Namespace(input.NS).Verify(ctx); err != nil {
			return nil, mapBlobError(err)
		}

		view := MessageOutput{Message: "ok"}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewCompactNamespaceHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		result, err := svc.Namespace(input.NS).Compact(ctx)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := model.CompactResultDocument{
			BlobsRemoved:      result.BlobsRemoved,
			ChunksRemoved:     result.ChunksRemoved,
			BytesFreed:        result.BytesFreed,
			SegmentsCompacted: result.SegmentsCompacted,
		}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

// ensureKeyWritable enforces overwrite protection: unless overwrite is true,
// an existing key rejects the write with runtime.ErrAlreadyExists.
func ensureKeyWritable(ctx context.Context, svc blobutil.BlobStore, nsID, key string, overwrite bool) error {
	if key == "" {
		return common.NewSystemError("VALIDATION_ERROR", "blob key is required")
	}
	if overwrite {
		return nil
	}
	_, err := svc.Namespace(nsID).Head(ctx, key)
	if err == nil {
		return runtime.ErrAlreadyExists.WithCause(fmt.Errorf("blob key %q already exists", key))
	}
	var notFound *bserrors.NotFoundError
	if errors.As(err, &notFound) {
		return nil
	}
	return mapBlobError(err)
}

// blobMetaView maps store metadata into its typed wire shape, rendering
// timestamps as RFC3339 strings and omitting an untouched UpdatedAt.
func blobMetaView(m *blobutil.BlobMeta) BlobMetaView {
	view := BlobMetaView{
		Key:         m.Key,
		NamespaceID: m.NamespaceID,
		ContentType: m.ContentType,
		Size:        m.Size,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339Nano),
		Custom:      m.Custom,
	}
	if !m.UpdatedAt.IsZero() {
		updated := m.UpdatedAt.Format(time.RFC3339Nano)
		view.UpdatedAt = &updated
	}
	return view
}

func mapBlobError(err error) error {
	var notFound *bserrors.NotFoundError
	if errors.As(err, &notFound) {
		return runtime.ErrNotFound.WithCause(err)
	}
	var exists *bserrors.AlreadyExistsError
	if errors.As(err, &exists) {
		return runtime.ErrAlreadyExists.WithCause(err)
	}
	var corruption *bserrors.CorruptionError
	if errors.As(err, &corruption) {
		return common.NewSystemError("CORRUPTION", corruption.Error()).WithCause(err)
	}
	return fmt.Errorf("blob: %w", err)
}

type BlobOp struct {
	Suffix, RuleKey, Desc string
}

var blobOps = []BlobOp{
	{"list", "administrator", "List blobs"},
	{"head", "administrator", "Get blob metadata"},
	{"upload", "administrator", "Upload a blob"},
	{"download", "administrator", "Download a blob"},
	{"delete", "administrator", "Delete a blob"},
	{"update", "administrator", "Update blob metadata"},
	{"rename", "administrator", "Rename a blob"},
	{"stats", "administrator", "Get namespace stats"},
	{"verify", "administrator", "Verify namespace integrity"},
	{"begin", "administrator", "Begin a resumable blob upload"},
	{"chunk", "administrator", "Upload a chunk of a resumable blob upload"},
	{"complete", "administrator", "Complete a resumable blob upload"},
	{"progress", "administrator", "Report progress of a resumable blob upload"},
	{"abort", "administrator", "Abort a resumable blob upload"},
	{"admin", "administrator", "Administer blob namespace"},
}

func BlobOps() []BlobOp { return blobOps }

func SeedNamespaceBindings(ctx context.Context, policyOp abstract.BindingPolicyStore, nsID string) error {
	for _, op := range blobOps {
		opName := "system:blobs:" + nsID + ":" + op.Suffix
		if err := policyOp.EnsureBinding(ctx, opName, op.RuleKey); err != nil {
			return fmt.Errorf("register operation %s: %w", opName, err)
		}
	}
	return policyOp.ReloadPolicies(ctx)
}

func RegisterBlobHandlers(registry abstract.Registry, svc blobutil.BlobStore, mgr *staging.Manager, nsID string) error {
	entries := []struct {
		suffix  string
		handler abstract.MessageHandler
	}{
		{"list", NewListBlobsHandler(svc)},
		{"head", NewHeadBlobHandler(svc)},
		{"upload", NewUploadBlobHandler(svc)},
		{"download", NewDownloadBlobHandler(svc)},
		{"delete", NewDeleteBlobHandler(svc)},
		{"update", NewUpdateBlobHandler(svc)},
		{"rename", NewRenameBlobHandler(svc)},
		{"stats", NewStatsNamespaceHandler(svc)},
		{"verify", NewVerifyNamespaceHandler(svc)},
		{"begin", NewBeginUploadHandler(svc, mgr)},
		{"chunk", NewUploadChunkHandler(mgr)},
		{"complete", NewCompleteUploadHandler(svc, mgr)},
		{"progress", NewProgressUploadHandler(mgr)},
		{"abort", NewAbortUploadHandler(mgr)},
	}
	for _, e := range entries {
		name := "system:blobs:" + nsID + ":" + e.suffix
		if err := registry.RegisterHandler(name, e.handler, abstract.HandlerInfo{
			Name:        name,
			Description: fmt.Sprintf("%s in namespace %q", blobOpDesc(e.suffix), nsID),
			Enabled:     true,
		}); err != nil {
			return fmt.Errorf("register %s: %w", name, err)
		}
	}
	return nil
}

func UnregisterBlobHandlers(registry abstract.Registry, nsID string) {
	for _, op := range blobOps {
		registry.DeleteHandler("system:blobs:" + nsID + ":" + op.Suffix)
	}
}

func blobOpDesc(suffix string) string {
	for _, op := range blobOps {
		if op.Suffix == suffix {
			return op.Desc
		}
	}
	return suffix
}
