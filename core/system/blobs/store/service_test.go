package store

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/asaidimu/blobs/object"
	"go.uber.org/zap"
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

// TestServiceNamespaceLifecycle exercises the blobs v1.3.5 store through the
// Hestia wrapper: create, put/get, delete, and re-create the same name. The
// re-created namespace must not inherit the deleted namespace's data.
func TestServiceNamespaceLifecycle(t *testing.T) {
	svc, err := NewService(t.TempDir(), zap.NewNop())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	ctx := context.Background()
	if err := svc.CreateNamespace(ctx, "test-ns", "Test", WithPublic(true)); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
	info, err := svc.GetNamespace(ctx, "test-ns")
	if err != nil {
		t.Fatalf("GetNamespace: %v", err)
	}
	if info.ID != "test-ns" || !info.Public {
		t.Fatalf("unexpected namespace info: %+v", info)
	}

	ns := svc.Namespace("test-ns")
	if _, err := ns.Put(ctx, "hello.txt", "text/plain", bytes.NewReader([]byte("hello"))); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, err := ns.Get(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, err := io.ReadAll(rc)
	if closeErr := rc.Close(); closeErr != nil {
		t.Errorf("close reader: %v", closeErr)
	}
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("Get returned %q, want %q", got, "hello")
	}

	if err := svc.DeleteNamespace(ctx, "test-ns"); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}
	if _, err := svc.GetNamespace(ctx, "test-ns"); err == nil {
		t.Fatal("expected deleted namespace to be gone")
	}

	if err := svc.CreateNamespace(ctx, "test-ns", "Test again"); err != nil {
		t.Fatalf("re-create namespace: %v", err)
	}
	if _, err := svc.Namespace("test-ns").Head(ctx, "hello.txt"); err == nil {
		t.Fatal("re-created namespace inherited the deleted namespace's blob")
	}
}
