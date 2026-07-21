package blobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	bbolt "github.com/asaidimu/blobs/index/backend"
	"github.com/asaidimu/blobs/object"
	"github.com/asaidimu/blobs/store"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/runtime"
)

type Service struct {
	s      *store.Store
	logger *zap.Logger
}

func NewService(dataDir string, logger *zap.Logger) (*Service, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("blobs: create data dir: %w", err)
	}
	idx, err := bbolt.Open(bbolt.Options{
		Path: filepath.Join(dataDir, "blobs.idx"),
	})
	if err != nil {
		return nil, fmt.Errorf("blobs: open index: %w", err)
	}

	s, err := store.Open(store.Config{
		DataDir: dataDir,
		Index:   idx,
	})
	if err != nil {
		_ = idx.Close()
		return nil, fmt.Errorf("blobs: open store: %w", err)
	}

	return &Service{s: s, logger: logger}, nil
}

func (svc *Service) Close() error {
	return svc.s.Close()
}

func (svc *Service) CreateNamespace(ctx context.Context, nsID, displayName string, opts ...runtime.NamespaceOption) error {
	var cfg runtime.NamespaceOptions
	for _, o := range opts {
		o(&cfg)
	}
	custom := make(map[string]string)
	if cfg.Public {
		custom["public"] = "true"
	}
	ns := object.Namespace{
		ID:          nsID,
		DisplayName: displayName,
		Custom:      custom,
	}
	return svc.s.CreateNamespace(ctx, ns)
}

func (svc *Service) GetNamespace(ctx context.Context, nsID string) (*runtime.BlobNamespaceInfo, error) {
	ns, err := svc.s.GetNamespace(ctx, nsID)
	if err != nil {
		return nil, err
	}
	info := &runtime.BlobNamespaceInfo{
		ID:          ns.ID,
		DisplayName: ns.DisplayName,
		Public:      ns.Custom["public"] == "true",
	}
	return info, nil
}

func (svc *Service) DeleteNamespace(ctx context.Context, nsID string) error {
	return svc.s.DeleteNamespace(ctx, nsID)
}

func (svc *Service) ListNamespaces(ctx context.Context) ([]runtime.BlobNamespaceInfo, error) {
	objs, err := svc.s.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]runtime.BlobNamespaceInfo, len(objs))
	for i, ns := range objs {
		out[i] = runtime.BlobNamespaceInfo{
			ID:          ns.ID,
			DisplayName: ns.DisplayName,
			Public:      ns.Custom["public"] == "true",
		}
	}
	return out, nil
}

func (svc *Service) Namespace(nsID string) runtime.BlobNamespace {
	return &nsHandle{ns: svc.s.Namespace(nsID)}
}

type nsHandle struct {
	ns *store.NamespaceHandle
}

func (h *nsHandle) Put(ctx context.Context, key, contentType string, reader io.Reader) (*runtime.BlobMeta, error) {
	info, err := h.ns.Put(ctx, key, reader, store.PutOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}
	return blobMetaFromInfo(info), nil
}

func (h *nsHandle) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return h.ns.Get(ctx, key)
}

func (h *nsHandle) Head(ctx context.Context, key string) (*runtime.BlobMeta, error) {
	info, err := h.ns.Head(ctx, key)
	if err != nil {
		return nil, err
	}
	return blobMetaFromInfo(info), nil
}

func (h *nsHandle) UpdateMetadata(ctx context.Context, key string, custom map[string]string) (*runtime.BlobMeta, error) {
	meta := make(map[string]any, len(custom))
	for k, v := range custom {
		meta[k] = v
	}
	if err := h.ns.Update(ctx, key, meta); err != nil {
		return nil, err
	}
	info, err := h.ns.Head(ctx, key)
	if err != nil {
		return nil, err
	}
	return blobMetaFromInfo(info), nil
}

func (h *nsHandle) Delete(ctx context.Context, key string) error {
	return h.ns.Delete(ctx, key)
}

func (h *nsHandle) List(ctx context.Context, prefix string, limit int) ([]runtime.BlobMeta, error) {
	infos, err := h.ns.List(ctx, store.ListOptions{KeyPrefix: prefix, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]runtime.BlobMeta, len(infos))
	for i, info := range infos {
		out[i] = *blobMetaFromInfo(&info)
	}
	return out, nil
}

func blobMetaFromInfo(info *object.BlobInfo) *runtime.BlobMeta {
	return &runtime.BlobMeta{
		Key:         info.Key,
		NamespaceID: info.NamespaceID,
		ContentType: info.Metadata.ContentType,
		Size:        info.Metadata.Size,
		CreatedAt:   info.Metadata.CreatedAt,
		UpdatedAt:   info.Metadata.UpdatedAt,
		Custom:      info.Metadata.Custom,
	}
}
