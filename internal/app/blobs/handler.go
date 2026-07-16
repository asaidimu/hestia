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

func NewCreateNamespaceHandler(svc core.BlobStore) core.MessageHandler {
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

		return &registration.Result{
			Document: mustDoc(map[string]any{
				"id":           nsID,
				"display_name": displayName,
				"public":       len(opts) > 0 && body["public"] == true,
			}, ctx),
		}, nil
	}
}

func NewDeleteNamespaceHandler(svc core.BlobStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
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
