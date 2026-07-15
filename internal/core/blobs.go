package core

import (
	"context"
	"io"
	"time"
)

type BlobMeta struct {
	Key         string    `json:"key"`
	NamespaceID string    `json:"namespace_id"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

type NamespaceOption func(*NamespaceOptions)

type NamespaceOptions struct {
	Public bool
}

func WithPublic(v bool) NamespaceOption {
	return func(o *NamespaceOptions) { o.Public = v }
}

type BlobStore interface {
	CreateNamespace(ctx context.Context, nsID, displayName string, opts ...NamespaceOption) error
	DeleteNamespace(ctx context.Context, nsID string) error
	GetNamespace(ctx context.Context, nsID string) (*BlobNamespaceInfo, error)
	ListNamespaces(ctx context.Context) ([]BlobNamespaceInfo, error)
	Namespace(nsID string) BlobNamespace
}

type BlobNamespaceInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Public      bool   `json:"public"`
}

type BlobNamespace interface {
	Put(ctx context.Context, key, contentType string, reader io.Reader) (*BlobMeta, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Head(ctx context.Context, key string) (*BlobMeta, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string, limit int) ([]BlobMeta, error)
}
