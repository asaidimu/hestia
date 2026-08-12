package blobs_test

import (
	"context"
	"errors"
	"io"
	"testing"

	bserrors "github.com/asaidimu/blobs/errors"
	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/blobs"
	blobutil "github.com/asaidimu/hestia/core/feature/blobs/store"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	m.Run()
}

// ── mocks ────────────────────────────────────────────────────────────────────

type mockBlobNamespace struct {
	blobutil.BlobNamespace
	headErr error
	listErr error
	listRes []blobutil.BlobMeta
}

func (m mockBlobNamespace) Head(_ context.Context, _ string) (*blobutil.BlobMeta, error) {
	return nil, m.headErr
}

func (m mockBlobNamespace) List(_ context.Context, _ string, _ int) ([]blobutil.BlobMeta, error) {
	return m.listRes, m.listErr
}

func (m mockBlobNamespace) Put(_ context.Context, key, contentType string, r io.Reader) (*blobutil.BlobMeta, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = "text/plain"
	}
	return &blobutil.BlobMeta{Key: key, NamespaceID: "test-ns", ContentType: contentType, Size: int64(len(data))}, nil
}

type mockBlobStore struct {
	blobutil.BlobStore
	ns           mockBlobNamespace
	namespaceErr error
}

func (m mockBlobStore) Namespace(_ string) blobutil.BlobNamespace {
	return m.ns
}

type testMessage struct {
	ctx   context.Context
	input data.Documenter
}

func (m testMessage) ID() string                             { return "" }
func (m testMessage) Name() string                           { return "" }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() data.Documenter                  { return m.input }
func (m testMessage) InputChannel() <-chan data.Documenter    { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m testMessage) TenantID() string                       { return "" }
func (m testMessage) TraceID() string                        { return "" }
func (m testMessage) RequestID() string                      { return "" }
func (m testMessage) SourceIP() string                       { return "" }
func (m testMessage) UserAgent() string                      { return "" }
func (m testMessage) ResourceID() string                     { return "" }
func (m testMessage) SessionID() string                      { return "" }

// ── tests ────────────────────────────────────────────────────────────────────

func TestPolicyBindings(t *testing.T) {
	bindings := blobs.PolicyBindings()
	if len(bindings) == 0 {
		t.Fatal("PolicyBindings() returned empty slice")
	}

	expected := map[string]string{
		"system:blobs:namespace:list":   "administrator",
		"system:blobs:namespace:create": "administrator",
		"system:blobs:namespace:delete": "administrator",
		"system:blobs:blob:list":        "administrator",
		"system:blobs:blob:head":        "administrator",
		"system:blobs:blob:upload":      "administrator",
		"system:blobs:blob:download":    "administrator",
		"system:blobs:blob:delete":      "administrator",
		"system:blobs:blob:update":      "administrator",
	}

	if len(bindings) != len(expected) {
		t.Errorf("got %d bindings, want %d", len(bindings), len(expected))
	}

	for _, b := range bindings {
		want, ok := expected[b.Name]
		if !ok {
			t.Errorf("unexpected binding %q", b.Name)
			continue
		}
		if b.RuleKey != want {
			t.Errorf("%s ruleKey = %q, want %q", b.Name, b.RuleKey, want)
		}
	}
}

func TestMapBlobError(t *testing.T) {
	ctx := context.Background()
	input := testutil.InputDoc(t, blobs.BlobKeyInputSchema(), `{
		"arguments": {
			"ns":  "test-ns",
			"key": "test-key"
		}
	}`)

	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "not found", err: &bserrors.NotFoundError{NamespaceID: "ns", Key: "k"}, code: "NOT_FOUND"},
		{name: "already exists", err: &bserrors.AlreadyExistsError{NamespaceID: "ns"}, code: "ALREADY_EXISTS"},
		{name: "unknown", err: io.EOF, code: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mockBlobStore{ns: mockBlobNamespace{headErr: tt.err}}
			handler := blobs.NewHeadBlobHandler(store)
			msg := testMessage{ctx: ctx, input: input}

			_, err := handler(ctx, msg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if tt.code == "" {
				if !errors.Is(err, tt.err) {
					t.Errorf("expected error to wrap %v, got %v", tt.err, err)
				}
				return
			}

			var sysErr *common.SystemError
			if !errors.As(err, &sysErr) {
				t.Fatalf("expected *common.SystemError, got %T", err)
			}
			if sysErr.Code != tt.code {
				t.Errorf("Code = %q, want %q", sysErr.Code, tt.code)
			}
		})
	}
}

func TestMustDoc(t *testing.T) {
	ctx := context.Background()

	store := mockBlobStore{
		ns: mockBlobNamespace{
			listRes: []blobutil.BlobMeta{
				{Key: "a.txt", NamespaceID: "ns1", ContentType: "text/plain", Size: 10},
			},
		},
	}
	handler := blobs.NewListBlobsHandler(store)
	input := testutil.InputDoc(t, blobs.BlobListInputSchema(), `{
		"arguments": { "ns": "ns1" }
	}`)
	msg := testMessage{ctx: ctx, input: input}

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil Document")
	}

	raw, err := result.Document.Get("blobs")
	if err != nil {
		t.Fatalf("expected 'blobs' key: %v", err)
	}
	items, ok := raw.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", raw)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 blob, got %d", len(items))
	}
}

func TestUploadBlobHandler(t *testing.T) {
	ctx := context.Background()
	store := mockBlobStore{ns: mockBlobNamespace{
		headErr: &bserrors.NotFoundError{NamespaceID: "test-ns", Key: "a.txt"},
	}}
	handler := blobs.NewUploadBlobHandler(store)

	input := testutil.InputDoc(t, blobs.BlobUploadInputSchema(), `{
		"arguments": {"ns": "test-ns", "key": "a.txt"},
		"headers": {"content_type": "text/plain"},
		"payload": "aGVsbG8gd29ybGQ="
	}`)
	msg := testMessage{ctx: ctx, input: input}

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("expected non-nil result document")
	}
	ct, _ := result.Document.GetString("content_type")
	if ct != "text/plain" {
		t.Errorf("content_type = %q, want %q", ct, "text/plain")
	}
	size, _ := result.Document.GetOr("size", int64(0)).(int64)
	if size != int64(len("hello world")) {
		t.Errorf("size = %d, want %d", size, len("hello world"))
	}
}

func TestUploadBlobHandlerRejectsExistingKey(t *testing.T) {
	ctx := context.Background()
	store := mockBlobStore{ns: mockBlobNamespace{}} // Head succeeds → key exists
	handler := blobs.NewUploadBlobHandler(store)

	input := testutil.InputDoc(t, blobs.BlobUploadInputSchema(), `{
		"arguments": {"ns": "test-ns", "key": "a.txt"},
		"payload": "aGVsbG8gd29ybGQ="
	}`)

	_, err := handler(ctx, testMessage{ctx: ctx, input: input})
	if err == nil {
		t.Fatal("expected error for existing key without overwrite")
	}
	var sysErr *common.SystemError
	if !errors.As(err, &sysErr) || sysErr.Code != "ALREADY_EXISTS" {
		t.Errorf("expected ALREADY_EXISTS, got %v", err)
	}
}

func TestUploadBlobHandlerOverwriteAllowsExistingKey(t *testing.T) {
	ctx := context.Background()
	store := mockBlobStore{ns: mockBlobNamespace{}} // Head succeeds → key exists
	handler := blobs.NewUploadBlobHandler(store)

	input := testutil.InputDoc(t, blobs.BlobUploadInputSchema(), `{
		"arguments": {"ns": "test-ns", "key": "a.txt"},
		"modifiers": {"overwrite": "true"},
		"payload": "aGVsbG8gd29ybGQ="
	}`)

	result, err := handler(ctx, testMessage{ctx: ctx, input: input})
	if err != nil {
		t.Fatalf("upload with overwrite: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("expected non-nil result document")
	}
}

func TestUploadBlobHandlerEmptyPayload(t *testing.T) {
	ctx := context.Background()
	store := mockBlobStore{ns: mockBlobNamespace{}}
	handler := blobs.NewUploadBlobHandler(store)

	input := testutil.InputDoc(t, blobs.BlobUploadInputSchema(), `{
		"arguments": {"ns": "test-ns", "key": "a.txt"}
	}`)
	msg := testMessage{ctx: ctx, input: input}

	_, err := handler(ctx, msg)
	if err == nil {
		t.Fatal("expected error for empty payload")
	}
	var sysErr *common.SystemError
	if !errors.As(err, &sysErr) || sysErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}
