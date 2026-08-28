package store

import (
	"context"
	"io"
	"time"
)

type BlobMeta struct {
	Key         string            `json:"key"`
	NamespaceID string            `json:"namespace_id"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

type NamespaceOption func(*NamespaceOptions)

type NamespaceOptions struct {
	Public bool
	Custom map[string]string
}

func WithPublic(v bool) NamespaceOption {
	return func(o *NamespaceOptions) { o.Public = v }
}

func WithCustom(m map[string]string) NamespaceOption {
	return func(o *NamespaceOptions) { o.Custom = m }
}

type BlobStore interface {
	CreateNamespace(ctx context.Context, nsID, displayName string, opts ...NamespaceOption) error
	DeleteNamespace(ctx context.Context, nsID string) error
	GetNamespace(ctx context.Context, nsID string) (*BlobNamespaceInfo, error)
	ListNamespaces(ctx context.Context) ([]BlobNamespaceInfo, error)
	Namespace(nsID string) BlobNamespace
}

// @note #feat-ns-metadata feature_request P2 : Add arbitrary metadata to blob namespaces
//
// BlobNamespaceInfo currently only carries ID, DisplayName, and Public.
// Users want to attach custom key-value metadata to namespaces (e.g. description,
// owner, environment tags, retention policy) similar to how BlobMeta.Custom works
// on individual blobs.
//
// Options: (1) add Custom map[string]string to BlobNamespaceInfo + persist in
// hestia's layer since the blobs library doesn't support it, (2) store namespace
// metadata in a dedicated collection keyed by namespace ID, (3) extend NamespaceOptions
// with a Custom field and persist alongside namespace creation.
type BlobNamespaceInfo struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"display_name"`
	Public      bool              `json:"public"`
	Custom      map[string]string `json:"custom,omitempty"`
}

type NamespaceStats struct {
	NamespaceID   string `json:"namespace_id"`
	BlobCount     int64  `json:"blob_count"`
	BytesStored   int64  `json:"bytes_stored"`
	BytesPhysical int64  `json:"bytes_physical"`
	ChunkCount    int64  `json:"chunk_count"`
	DeadBytes     int64  `json:"dead_bytes"`
	DeadChunks    int64  `json:"dead_chunks"`
	SegmentCount  int64  `json:"segment_count"`
}

type CompactResult struct {
	BlobsRemoved      int64 `json:"blobs_removed"`
	ChunksRemoved     int64 `json:"chunks_removed"`
	BytesFreed        int64 `json:"bytes_freed"`
	SegmentsCompacted int64 `json:"segments_compacted"`
}

type BlobNamespace interface {
	Put(ctx context.Context, key, contentType string, reader io.Reader) (*BlobMeta, error)
	PutCustom(ctx context.Context, key, contentType string, reader io.Reader, custom map[string]string) (*BlobMeta, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Head(ctx context.Context, key string) (*BlobMeta, error)
	UpdateMetadata(ctx context.Context, key string, custom map[string]string) (*BlobMeta, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string, limit int) ([]BlobMeta, error)
	Rename(ctx context.Context, oldKey, newKey string) error
	Stats(ctx context.Context) (*NamespaceStats, error)
	Compact(ctx context.Context) (*CompactResult, error)
	Verify(ctx context.Context) error
}
