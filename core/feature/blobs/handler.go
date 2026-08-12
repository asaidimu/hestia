package blobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"

	bserrors "github.com/asaidimu/blobs/errors"
	"github.com/asaidimu/blobs/staging"

	"github.com/asaidimu/hestia/core/abstract"
	blobutil "github.com/asaidimu/hestia/core/feature/blobs/store"
	"github.com/asaidimu/hestia/core/internal/util"
	"github.com/asaidimu/hestia/core/runtime"
)

func NewListNamespacesHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		namespaces, err := svc.ListNamespaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("list namespaces: %w", err)
		}

		docs := make([]map[string]any, len(namespaces))
		for i, ns := range namespaces {
			docs[i] = map[string]any{
				"id":           ns.ID,
				"display_name": ns.DisplayName,
				"public":       ns.Public,
			}
		}

		return &abstract.Result{Document: mustDoc(map[string]any{"namespaces": docs}, ctx)}, nil
	}
}

func NewCreateNamespaceHandler(svc blobutil.BlobStore, mgr *staging.Manager, policyOp abstract.BindingPolicyStore, registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		body, _ := msg.Input().GetOr("payload", nil).(map[string]any)
		displayName, _ := body["display_name"].(string)

		nsID := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			nsID = displayName
		}
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		var opts []blobutil.NamespaceOption
		if public, ok := body["public"].(bool); ok && public {
			opts = append(opts, blobutil.WithPublic(true))
		}

		if err := svc.CreateNamespace(ctx, nsID, displayName, opts...); err != nil {
			return nil, fmt.Errorf("create namespace: %w", err)
		}

		if err := SeedNamespaceBindings(ctx, policyOp, nsID); err != nil {
			return nil, fmt.Errorf("seed namespace operations: %w", err)
		}

		if err := RegisterBlobHandlers(registry, svc, mgr, nsID); err != nil {
			return nil, fmt.Errorf("register blob handlers: %w", err)
		}

		return &abstract.Result{
			Document: mustDoc(map[string]any{
				"id":           nsID,
				"display_name": displayName,
				"public":       len(opts) > 0 && body["public"] == true,
			}, ctx),
		}, nil
	}
}

func NewDeleteNamespaceHandler(svc blobutil.BlobStore, policyOp abstract.BindingPolicyStore, registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
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
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		prefix := ""
		limit := 0
		if body, ok := msg.Input().GetOr("payload", nil).(map[string]any); ok {
			prefix, _ = body["prefix"].(string)
			if l, ok := body["limit"].(float64); ok {
				limit = int(l)
			}
		}

		blobs, err := svc.Namespace(nsID).List(ctx, prefix, limit)
		if err != nil {
			return nil, fmt.Errorf("list blobs: %w", err)
		}

		items := make([]map[string]any, len(blobs))
		for i, b := range blobs {
			items[i] = map[string]any{
				"key":          b.Key,
				"namespace_id": b.NamespaceID,
				"content_type": b.ContentType,
				"size":         b.Size,
				"created_at":   b.CreatedAt,
			}
		}

		return &abstract.Result{Document: mustDoc(map[string]any{"blobs": items}, ctx)}, nil
	}
}

func NewHeadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		meta, err := svc.Namespace(nsID).Head(ctx, key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		return &abstract.Result{
			Document: mustDoc(map[string]any{
				"key":          meta.Key,
				"namespace_id": meta.NamespaceID,
				"content_type": meta.ContentType,
				"size":         meta.Size,
				"created_at":   meta.CreatedAt,
			}, ctx),
		}, nil
	}
}

func NewUploadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)
		contentType, _ := msg.Input().GetOr("headers.content_type", "").(string)

		raw, _ := msg.Input().Get("payload")
		data, _ := raw.([]byte)
		if len(data) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "request body is required")
		}
		if len(data) > maxDirectUploadBytes {
			return nil, common.NewSystemError("VALIDATION_ERROR",
				fmt.Sprintf("direct upload exceeds %d byte limit; use the resumable upload protocol", maxDirectUploadBytes))
		}

		if err := ensureKeyWritable(ctx, svc, nsID, key, msg.Input().GetOr("modifiers.overwrite", "") == "true"); err != nil {
			return nil, err
		}

		meta, err := svc.Namespace(nsID).Put(ctx, key, contentType, bytes.NewReader(data))
		if err != nil {
			return nil, mapBlobError(err)
		}

		return &abstract.Result{
			Document: mustDoc(map[string]any{
				"key":          meta.Key,
				"namespace_id": meta.NamespaceID,
				"content_type": meta.ContentType,
				"size":         meta.Size,
				"created_at":   meta.CreatedAt,
			}, ctx),
		}, nil
	}
}

func NewDownloadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		meta, err := svc.Namespace(nsID).Head(ctx, key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		rc, err := svc.Namespace(nsID).Get(ctx, key)
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
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		body, _ := msg.Input().GetOr("payload", nil).(map[string]any)

		var custom map[string]string
		if raw, ok := body["custom"].(map[string]any); ok {
			custom = make(map[string]string, len(raw))
			for k, v := range raw {
				custom[k] = fmt.Sprint(v)
			}
		}

		meta, err := svc.Namespace(nsID).UpdateMetadata(ctx, key, custom)
		if err != nil {
			return nil, mapBlobError(err)
		}

		return &abstract.Result{
			Document: mustDoc(util.StructToMap(*meta), ctx),
		}, nil
	}
}

func NewDeleteBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		if _, err := svc.Namespace(nsID).Head(ctx, key); err != nil {
			return nil, mapBlobError(err)
		}

		if err := svc.Namespace(nsID).Delete(ctx, key); err != nil {
			return nil, mapBlobError(err)
		}

		return &abstract.Result{}, nil
	}
}

func wrapErr(err error, code, msg string) *common.SystemError {
	if sysErr, ok := err.(*common.SystemError); ok {
		return common.NewSystemError(code, fmt.Sprintf("%s: %s", msg, sysErr.Error())).
			WithCause(sysErr)
	}
	return common.NewSystemError(code, fmt.Sprintf("%s: %s", msg, err.Error()))
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

func mapBlobError(err error) error {
	var notFound *bserrors.NotFoundError
	if errors.As(err, &notFound) {
		return runtime.ErrNotFound.WithCause(err)
	}
	var exists *bserrors.AlreadyExistsError
	if errors.As(err, &exists) {
		return runtime.ErrAlreadyExists.WithCause(err)
	}
	return fmt.Errorf("blob: %w", err)
}

func mustDoc(m map[string]any, ctx context.Context) *data.Document {
	return data.MustNewDocument(m, ctx)
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
