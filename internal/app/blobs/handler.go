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

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
)

type OperationPolicyStore interface {
	EnsureOperation(ctx context.Context, name, ruleKey, intentType, description string) error
	DeleteOperation(ctx context.Context, name string) error
	ForceDeleteOperation(ctx context.Context, name string) error
	EnsureRule(ctx context.Context, name, expr, description string) error
	DeleteRule(ctx context.Context, name string) error
	ForceDeleteRule(ctx context.Context, name string) error
	ReloadPolicies(ctx context.Context) error
}

func NewListNamespacesHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

		return &registration.Result{Document: mustDoc(map[string]any{"namespaces": docs}, ctx)}, nil
	}
}

func NewCreateNamespaceHandler(svc core.BlobStore, policyOp OperationPolicyStore, registry core.Registry) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		body, _ := msg.Input().GetOr("payload", nil).(map[string]any)
		displayName, _ := body["display_name"].(string)

		nsID := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			nsID = displayName
		}
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		var opts []core.NamespaceOption
		if public, ok := body["public"].(bool); ok && public {
			opts = append(opts, core.WithPublic(true))
		}

		if err := svc.CreateNamespace(ctx, nsID, displayName, opts...); err != nil {
			return nil, fmt.Errorf("create namespace: %w", err)
		}

		if err := SeedNamespaceOperations(ctx, policyOp, nsID); err != nil {
			return nil, fmt.Errorf("seed namespace operations: %w", err)
		}

		if err := RegisterBlobHandlers(registry, svc, nsID); err != nil {
			return nil, fmt.Errorf("register blob handlers: %w", err)
		}

		return &registration.Result{
			Document: mustDoc(map[string]any{
				"id":           nsID,
				"display_name": displayName,
				"public":       len(opts) > 0 && body["public"] == true,
			}, ctx),
		}, nil
	}
}

func NewDeleteNamespaceHandler(svc core.BlobStore, policyOp OperationPolicyStore, registry core.Registry) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		UnregisterBlobHandlers(registry, nsID)

		for _, op := range blobOps {
			opName := "blob." + nsID + "." + op.Suffix
			if err := policyOp.ForceDeleteOperation(ctx, opName); err != nil {
				return nil, fmt.Errorf("delete operation %s: %w", opName, err)
			}
		}

		if err := policyOp.ReloadPolicies(ctx); err != nil {
			return nil, fmt.Errorf("reload policies: %w", err)
		}

		if err := svc.DeleteNamespace(ctx, nsID); err != nil {
			return nil, fmt.Errorf("delete namespace: %w", err)
		}

		return &registration.Result{}, nil
	}
}

func NewListBlobsHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

		return &registration.Result{Document: mustDoc(map[string]any{"blobs": items}, ctx)}, nil
	}
}

func NewHeadBlobHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		meta, err := svc.Namespace(nsID).Head(ctx, key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		return &registration.Result{
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

func NewUploadBlobHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)
		contentType, _ := msg.Input().GetOr("content_type", "").(string)

		raw, _ := msg.Input().Get("payload")
		data, _ := raw.([]byte)
		if len(data) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "request body is required")
		}

		meta, err := svc.Namespace(nsID).Put(ctx, key, contentType, bytes.NewReader(data))
		if err != nil {
			return nil, mapBlobError(err)
		}

		return &registration.Result{
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

func NewDownloadBlobHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

		return &registration.Result{
			Blob: registration.Blob{
				Data:        data,
				ContentType: meta.ContentType,
			},
		}, nil
	}
}

func NewUpdateBlobHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		body, _ := msg.Input().GetOr("payload", nil).(map[string]any)
		contentType, _ := body["content_type"].(string)

		var custom map[string]string
		if raw, ok := body["custom"].(map[string]any); ok {
			custom = make(map[string]string, len(raw))
			for k, v := range raw {
				custom[k] = fmt.Sprint(v)
			}
		}

		meta, err := svc.Namespace(nsID).UpdateMetadata(ctx, key, contentType, custom)
		if err != nil {
			return nil, mapBlobError(err)
		}

		return &registration.Result{
			Document: mustDoc(blobMetaToMap(meta), ctx),
		}, nil
	}
}

func NewDeleteBlobHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		if _, err := svc.Namespace(nsID).Head(ctx, key); err != nil {
			return nil, mapBlobError(err)
		}

		if err := svc.Namespace(nsID).Delete(ctx, key); err != nil {
			return nil, mapBlobError(err)
		}

		return &registration.Result{}, nil
	}
}

func blobMetaToMap(m *core.BlobMeta) map[string]any {
	out := map[string]any{
		"key":          m.Key,
		"namespace_id": m.NamespaceID,
		"content_type": m.ContentType,
		"size":         m.Size,
		"created_at":   m.CreatedAt,
		"updated_at":   m.UpdatedAt,
	}
	if len(m.Custom) > 0 {
		out["custom"] = m.Custom
	}
	return out
}

func wrapErr(err error, code, msg string) *common.SystemError {
	if sysErr, ok := err.(*common.SystemError); ok {
		return common.NewSystemError(code, fmt.Sprintf("%s: %s", msg, sysErr.Error())).
			WithCause(sysErr)
	}
	return common.NewSystemError(code, fmt.Sprintf("%s: %s", msg, err.Error()))
}

func mapBlobError(err error) error {
	var notFound *bserrors.NotFoundError
	if errors.As(err, &notFound) {
		return core.ErrNotFound.WithCause(err)
	}
	var exists *bserrors.AlreadyExistsError
	if errors.As(err, &exists) {
		return core.ErrAlreadyExists.WithCause(err)
	}
	return fmt.Errorf("blob: %w", err)
}

func mustDoc(m map[string]any, ctx context.Context) *data.Document {
	return data.MustNewDocument(m, ctx)
}

type BlobOp struct {
	Suffix, RuleKey, Intent, Desc string
}

var blobOps = []BlobOp{
	{"list", "administrator", "QUERY", "List blobs"},
	{"head", "administrator", "QUERY", "Get blob metadata"},
	{"upload", "administrator", "COMMAND", "Upload a blob"},
	{"download", "administrator", "COMMAND", "Download a blob"},
	{"delete", "administrator", "COMMAND", "Delete a blob"},
	{"update", "administrator", "COMMAND", "Update blob metadata"},
	{"admin", "administrator", "COMMAND", "Administer blob namespace"},
}

func BlobOps() []BlobOp { return blobOps }

func SeedNamespaceOperations(ctx context.Context, policyOp OperationPolicyStore, nsID string) error {
	for _, op := range blobOps {
		opName := "blob." + nsID + "." + op.Suffix
		if err := policyOp.EnsureOperation(ctx, opName, op.RuleKey, op.Intent, op.Desc+" in "+nsID); err != nil {
			return fmt.Errorf("register operation %s: %w", opName, err)
		}
	}
	return policyOp.ReloadPolicies(ctx)
}

func RegisterBlobHandlers(registry core.Registry, svc core.BlobStore, nsID string) error {
	entries := []struct {
		suffix  string
		handler core.MessageHandler
	}{
		{"list", NewListBlobsHandler(svc)},
		{"head", NewHeadBlobHandler(svc)},
		{"upload", NewUploadBlobHandler(svc)},
		{"download", NewDownloadBlobHandler(svc)},
		{"delete", NewDeleteBlobHandler(svc)},
		{"update", NewUpdateBlobHandler(svc)},
	}
	for _, e := range entries {
		name := "blob." + nsID + "." + e.suffix
		if err := registry.RegisterHandler(name, e.handler, core.HandlerInfo{
			Name:        name,
			Description: fmt.Sprintf("%s in namespace %q", blobOpDesc(e.suffix), nsID),
			Enabled:     true,
		}); err != nil {
			return fmt.Errorf("register %s: %w", name, err)
		}
	}
	return nil
}

func UnregisterBlobHandlers(registry core.Registry, nsID string) {
	for _, op := range blobOps {
		registry.DeleteHandler("blob." + nsID + "." + op.Suffix)
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
