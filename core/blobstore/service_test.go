package blobs

import (
	"testing"
	"time"

	"github.com/asaidimu/blobs/object"
)

func TestBlobMetaFromInfo(t *testing.T) {
	now := time.Now().UTC()
	info := &object.BlobInfo{
		Key:         "my-key",
		NamespaceID: "my-ns",
		Metadata: object.Metadata{
			ContentType: "image/png",
			Size:        4096,
			CreatedAt:   now,
			BlobID:      "sha256:abc123",
			ChunkCount:  1,
		},
	}

	meta := blobMetaFromInfo(info)
	if meta == nil {
		t.Fatal("blobMetaFromInfo returned nil")
	}
	if meta.Key != "my-key" {
		t.Fatalf("expected Key %q, got %q", "my-key", meta.Key)
	}
	if meta.NamespaceID != "my-ns" {
		t.Fatalf("expected NamespaceID %q, got %q", "my-ns", meta.NamespaceID)
	}
	if meta.ContentType != "image/png" {
		t.Fatalf("expected ContentType %q, got %q", "image/png", meta.ContentType)
	}
	if meta.Size != 4096 {
		t.Fatalf("expected Size %d, got %d", 4096, meta.Size)
	}
	if !meta.CreatedAt.Equal(now) {
		t.Fatalf("expected CreatedAt %v, got %v", now, meta.CreatedAt)
	}
}

func TestBlobMetaFromInfoZeroValues(t *testing.T) {
	info := &object.BlobInfo{
		Key:         "",
		NamespaceID: "",
		Metadata:    object.Metadata{},
	}

	meta := blobMetaFromInfo(info)
	if meta == nil {
		t.Fatal("blobMetaFromInfo returned nil")
	}
	if meta.Key != "" {
		t.Fatalf("expected empty Key, got %q", meta.Key)
	}
	if meta.NamespaceID != "" {
		t.Fatalf("expected empty NamespaceID, got %q", meta.NamespaceID)
	}
	if meta.ContentType != "" {
		t.Fatalf("expected empty ContentType, got %q", meta.ContentType)
	}
	if meta.Size != 0 {
		t.Fatalf("expected Size 0, got %d", meta.Size)
	}
	if !meta.CreatedAt.IsZero() {
		t.Fatal("expected zero CreatedAt")
	}
}
