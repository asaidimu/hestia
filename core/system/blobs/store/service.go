package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	bserrors "github.com/asaidimu/blobs/errors"
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
	if logger == nil {
		logger = zap.NewNop()
	}
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

	// The v1.3.5 index upgrade is in place and one-way. Report it so an
	// operator knows the store was migrated and whether any chunks could not
	// be recovered.
	if report := s.MigrationReport(); report.Ran {
		logger.Info("blobs: upgraded index to the namespace-UID format",
			zap.Int("namespaces", report.Namespaces),
			zap.Int("blobs", report.Blobs),
			zap.Int("chunks", report.Chunks),
			zap.Int("chunks_rebuilt", report.ChunksRebuilt),
			zap.Int("unrecoverable", len(report.Unrecoverable)),
		)
		for _, u := range report.Unrecoverable {
			logger.Warn("blobs: chunk could not be recovered during index upgrade",
				zap.String("namespace", u.Namespace),
				zap.String("key", u.Key),
				zap.String("chunk_id", string(u.ChunkID)),
				zap.String("reason", u.Reason),
			)
		}
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

// @note #blob-20261029-009 issue P2 #blobs,#api : Namespace public flag lives in-band in custom, can be overridden by callers, and nothing enforces it
// @author claude
// @see #feat-ns-metadata
// @see #blob-20261029-026
//
// CreateNamespace writes custom["public"] = "true" and then copies the caller's custom map over it, so custom: {public: "false"} silently wins, and GetNamespace/ListNamespaces strip the key on read so a caller's own "public" key disappears. The flag is read back only for display: no code in core/ consults NamespaceView.Public to allow unauthenticated reads (grep found no other consumer), so "public" has no effect on access. The TS client does not expose it at all (#blob-20261029-026).
// Also missing: system:blobs:namespace:get and :update routes. GetNamespace exists but is used only by the dispatcher hydrator, so clients list every namespace to read one and cannot change display_name, custom or public after creation (#feat-ns-metadata).
// Fix: remove the flag, or give it a real field plus enforcement, and reject custom keys that collide with reserved ones.

func (svc *Service) CreateNamespace(ctx context.Context, nsID, displayName string, opts ...NamespaceOption) error {
	var cfg NamespaceOptions
	for _, o := range opts {
		o(&cfg)
	}
	custom := make(map[string]string)
	if cfg.Public {
		custom["public"] = "true"
	}
	for k, v := range cfg.Custom {
		custom[k] = v
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
	custom := make(map[string]string, len(ns.Custom))
	for k, v := range ns.Custom {
		if k == "public" {
			continue
		}
		custom[k] = v
	}
	info := &BlobNamespaceInfo{
		ID:          ns.ID,
		DisplayName: ns.DisplayName,
		Public:      ns.Custom["public"] == "true",
		Custom:      custom,
	}
	return info, nil
}

func (svc *Service) DeleteNamespace(ctx context.Context, nsID string) error {
	err := svc.s.DeleteNamespace(ctx, nsID)
	if err == nil {
		return nil
	}

	// v1.3.5 unlinks the name first and erases the data afterwards. If the
	// erase is deferred (cancelled context or backend failure) the namespace
	// is already gone and unreachable; only its files remain. The next Open
	// (or CleanupDeletedNamespaces) finishes the job, so this is not a delete
	// failure.
	var pending *bserrors.CleanupPendingError
	if errors.As(err, &pending) {
		svc.logger.Warn("blobs: namespace deleted, data erase pending",
			zap.String("namespace", nsID),
			zap.String("uid", pending.UID),
			zap.Error(pending.Err),
		)
		return nil
	}
	return err
}

func (svc *Service) ListNamespaces(ctx context.Context) ([]BlobNamespaceInfo, error) {
	objs, err := svc.s.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BlobNamespaceInfo, len(objs))
	for i, ns := range objs {
		custom := make(map[string]string, len(ns.Custom))
		for k, v := range ns.Custom {
			if k == "public" {
				continue
			}
			custom[k] = v
		}
		out[i] = BlobNamespaceInfo{
			ID:          ns.ID,
			DisplayName: ns.DisplayName,
			Public:      ns.Custom["public"] == "true",
			Custom:      custom,
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

func (h *nsHandle) Compact(ctx context.Context) (*CompactResult, error) {
	result, err := h.ns.Compact(ctx)
	if err != nil {
		return nil, err
	}
	return &CompactResult{
		BlobsRemoved:      result.BlobsRemoved,
		ChunksRemoved:     result.ChunksRemoved,
		BytesFreed:        result.BytesFreed,
		SegmentsCompacted: result.SegmentsCompacted,
	}, nil
}

func (h *nsHandle) Rename(ctx context.Context, oldKey, newKey string) error {
	return h.ns.Rename(ctx, oldKey, newKey)
}

func (h *nsHandle) Stats(ctx context.Context) (*NamespaceStats, error) {
	s, err := h.ns.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return &NamespaceStats{
		NamespaceID:   s.NamespaceID,
		BlobCount:     s.BlobCount,
		BytesStored:   s.BytesStored,
		BytesPhysical: s.BytesPhysical,
		ChunkCount:    s.ChunkCount,
		DeadBytes:     s.DeadBytes,
		DeadChunks:    s.DeadChunks,
		SegmentCount:  s.SegmentCount,
	}, nil
}

func (h *nsHandle) Verify(ctx context.Context) error {
	return h.ns.Verify(ctx)
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
