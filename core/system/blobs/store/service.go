package store

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	bbolt "github.com/asaidimu/blobs/index/backend"
	"github.com/asaidimu/blobs/object"
	"github.com/asaidimu/blobs/staging"
	"github.com/asaidimu/blobs/store"
	"go.uber.org/zap"
)

type Service struct {
	s          *store.Store
	staging    *staging.Manager
	stopReaper func()
	logger     *zap.Logger
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

	stagingDir := filepath.Join(dataDir, "staging")
	mgr, err := staging.NewManager(stagingDir)
	if err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("blobs: init staging manager: %w", err)
	}
	// Reap abandoned upload sessions: every 5 minutes, discard anything idle
	// for more than 6 hours.
	stopReaper := mgr.StartReaper(5*time.Minute, 6*time.Hour)

	return &Service{s: s, staging: mgr, stopReaper: stopReaper, logger: logger}, nil
}

func (svc *Service) Staging() *staging.Manager {
	return svc.staging
}

func (svc *Service) Close() error {
	if svc.stopReaper != nil {
		svc.stopReaper()
		svc.stopReaper = nil
	}
	return svc.s.Close()
}

func (svc *Service) CreateNamespace(ctx context.Context, nsID, displayName string, opts ...NamespaceOption) error {
	var cfg NamespaceOptions
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

func (svc *Service) GetNamespace(ctx context.Context, nsID string) (*BlobNamespaceInfo, error) {
	ns, err := svc.s.GetNamespace(ctx, nsID)
	if err != nil {
		return nil, err
	}
	info := &BlobNamespaceInfo{
		ID:          ns.ID,
		DisplayName: ns.DisplayName,
		Public:      ns.Custom["public"] == "true",
	}
	return info, nil
}

func (svc *Service) DeleteNamespace(ctx context.Context, nsID string) error {
	return svc.s.DeleteNamespace(ctx, nsID)
}

func (svc *Service) ListNamespaces(ctx context.Context) ([]BlobNamespaceInfo, error) {
	objs, err := svc.s.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BlobNamespaceInfo, len(objs))
	for i, ns := range objs {
		out[i] = BlobNamespaceInfo{
			ID:          ns.ID,
			DisplayName: ns.DisplayName,
			Public:      ns.Custom["public"] == "true",
		}
	}
	return out, nil
}

func (svc *Service) Namespace(nsID string) BlobNamespace {
	return &nsHandle{ns: svc.s.Namespace(nsID)}
}

type nsHandle struct {
	ns *store.NamespaceHandle
}

func (h *nsHandle) Put(ctx context.Context, key, contentType string, reader io.Reader) (*BlobMeta, error) {
	return h.PutCustom(ctx, key, contentType, reader, nil)
}

func (h *nsHandle) PutCustom(ctx context.Context, key, contentType string, reader io.Reader, custom map[string]string) (*BlobMeta, error) {
	info, err := h.ns.Put(ctx, key, reader, store.PutOptions{ContentType: contentType, Custom: custom})
	if err != nil {
		return nil, err
	}
	return blobMetaFromInfo(info), nil
}

func (h *nsHandle) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return h.ns.Get(ctx, key)
}

func (h *nsHandle) Head(ctx context.Context, key string) (*BlobMeta, error) {
	info, err := h.ns.Head(ctx, key)
	if err != nil {
		return nil, err
	}
	return blobMetaFromInfo(info), nil
}

func (h *nsHandle) UpdateMetadata(ctx context.Context, key string, custom map[string]string) (*BlobMeta, error) {
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

func (h *nsHandle) List(ctx context.Context, prefix string, limit int) ([]BlobMeta, error) {
	infos, err := h.ns.List(ctx, store.ListOptions{KeyPrefix: prefix, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]BlobMeta, len(infos))
	for i, info := range infos {
		out[i] = *blobMetaFromInfo(&info)
	}
	return out, nil
}

func (h *nsHandle) Compact(ctx context.Context) (int64, error) {
	result, err := h.ns.Compact(ctx)
	if err != nil {
		return 0, err
	}
	return result.BytesFreed, nil
}

func blobMetaFromInfo(info *object.BlobInfo) *BlobMeta {
	return &BlobMeta{
		Key:         info.Key,
		NamespaceID: info.NamespaceID,
		ContentType: info.Metadata.ContentType,
		Size:        info.Metadata.Size,
		CreatedAt:   info.Metadata.CreatedAt,
		UpdatedAt:   info.Metadata.UpdatedAt,
		Custom:      info.Metadata.Custom,
	}
}
