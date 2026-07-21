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
	"github.com/asaidimu/hestia/core/internal/feature/blobs"
	"github.com/asaidimu/hestia/core/runtime"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	m.Run()
}

// ── mocks ────────────────────────────────────────────────────────────────────

type mockBlobNamespace struct {
	runtime.BlobNamespace
	headErr error
	listErr error
	listRes []runtime.BlobMeta
}

func (m mockBlobNamespace) Head(_ context.Context, _ string) (*runtime.BlobMeta, error) {
	return nil, m.headErr
}

func (m mockBlobNamespace) List(_ context.Context, _ string, _ int) ([]runtime.BlobMeta, error) {
	return m.listRes, m.listErr
}

type mockBlobStore struct {
	runtime.BlobStore
	ns           mockBlobNamespace
	namespaceErr error
}

func (m mockBlobStore) Namespace(_ string) runtime.BlobNamespace {
	return m.ns
}

type testMessage struct {
	ctx   context.Context
	input *data.Document
}

func (m testMessage) ID() string                        { return "" }
func (m testMessage) Name() string                      { return "" }
func (m testMessage) Context() context.Context           { return m.ctx }
func (m testMessage) Input() *data.Document              { return m.input }
func (m testMessage) InputChannel() <-chan *data.Document { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

// ── tests ────────────────────────────────────────────────────────────────────

func TestDefaultOperations(t *testing.T) {
	ops := blobs.DefaultOperations()
	if len(ops) == 0 {
		t.Fatal("DefaultOperations() returned empty slice")
	}

	expected := map[string]string{
		"system:blobs:namespace:list":   "administrator",
		"system:blobs:namespace:create": "administrator",
		"system:blobs:namespace:delete": "administrator",
	}

	if len(ops) != len(expected) {
		t.Errorf("got %d operations, want %d", len(ops), len(expected))
	}

	for _, op := range ops {
		want, ok := expected[op.Name]
		if !ok {
			t.Errorf("unexpected operation %q", op.Name)
			continue
		}
		if op.RuleKey != want {
			t.Errorf("%s ruleKey = %q, want %q", op.Name, op.RuleKey, want)
		}
	}
}

func TestMapBlobError(t *testing.T) {
	ctx := context.Background()
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"ns":  "test-ns",
			"key": "test-key",
		},
	}, ctx)

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
			listRes: []runtime.BlobMeta{
				{Key: "a.txt", NamespaceID: "ns1", ContentType: "text/plain", Size: 10},
			},
		},
	}
	handler := blobs.NewListBlobsHandler(store)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"ns": "ns1"},
	}, ctx)
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
