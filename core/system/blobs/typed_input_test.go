package blobs_test

import (
	"context"
	"testing"

	bserrors "github.com/asaidimu/blobs/errors"
	"github.com/asaidimu/blobs/staging"

	"github.com/asaidimu/hestia/core/system/blobs"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/internal/testutil"
)

func TestProbeTypedInputs(t *testing.T) {
	ctx := context.Background()

	t.Run("list prefix+limit", func(t *testing.T) {
		store := mockBlobStore{ns: mockBlobNamespace{}}
		h := blobs.NewListBlobsHandler(store)
		in := testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[blobs.BlobListInput]("input", true), `{
			"arguments": {"ns": "ns1"},
			"payload": {"prefix": "x/", "limit": 25}
		}`)
		if _, err := h(ctx, testMessage{ctx: ctx, input: in}); err != nil {
			t.Fatalf("list: %v", err)
		}
	})

	t.Run("begin key+size+block_size", func(t *testing.T) {
		mgr, err := staging.NewManager(t.TempDir())
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		store := mockBlobStore{ns: mockBlobNamespace{
			headErr: &bserrors.NotFoundError{NamespaceID: "ns1", Key: "big.bin"},
		}}
		h := blobs.NewBeginUploadHandler(store, mgr)
		in := testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[blobs.BlobBeginInput]("input", true), `{
			"arguments": {"ns": "ns1"},
			"payload": {"key": "big.bin", "size": 10485760, "content_type": "application/octet-stream", "block_size": 2097152}
		}`)
		res, err := h(ctx, testMessage{ctx: ctx, input: in})
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if res == nil || res.Document == nil {
			t.Fatal("begin: nil document")
		}
		sid, _ := res.Document.Get("session_id")
		if sid == "" {
			t.Errorf("begin: missing session_id, doc=%v", res.Document.ToMap())
		}
	})

	t.Run("update custom", func(t *testing.T) {
		store := mockBlobStore{ns: mockBlobNamespace{
			headErr: &bserrors.NotFoundError{NamespaceID: "ns1", Key: "a.txt"},
		}}
		h := blobs.NewUpdateBlobHandler(store)
		in := testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[blobs.BlobUpdateInput]("input", true), `{
			"arguments": {"ns": "ns1", "key": "a.txt"},
			"payload": {"custom": {"author": "x", "tier": "1"}}
		}`)
		if _, err := h(ctx, testMessage{ctx: ctx, input: in}); err != nil {
			t.Fatalf("update: %v", err)
		}
	})
}
