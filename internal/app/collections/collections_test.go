package collections_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/abstract"
	"github.com/asaidimu/hestia/internal/app/collections"
	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

type stubPolicyStore struct{}

func (stubPolicyStore) EnsureOperation(context.Context, string, string, string, string) error {
	return nil
}
func (stubPolicyStore) DeleteOperation(context.Context, string) error   { return nil }
func (stubPolicyStore) ForceDeleteOperation(context.Context, string) error { return nil }
func (stubPolicyStore) EnsureRule(context.Context, string, string, string) error { return nil }
func (stubPolicyStore) DeleteRule(context.Context, string) error   { return nil }
func (stubPolicyStore) ForceDeleteRule(context.Context, string) error { return nil }
func (stubPolicyStore) ReloadPolicies(context.Context) error { return nil }

type stubRegistry struct{}

func (stubRegistry) RegisterHandler(string, core.MessageHandler, core.HandlerInfo) error {
	return nil
}
func (stubRegistry) GetHandler(string) (core.MessageHandler, error) { return nil, nil }
func (stubRegistry) DeleteHandler(string) error                     { return nil }
func (stubRegistry) ListHandlers() []core.HandlerInfo               { return nil }
func (stubRegistry) SetHandlerEnabled(string, bool) error           { return nil }

type testMessage struct {
	name  string
	ctx   context.Context
	input *data.Document
}

func (m testMessage) ID() string                           { return "" }
func (m testMessage) Name() string                         { return m.name }
func (m testMessage) Context() context.Context              { return m.ctx }
func (m testMessage) Input() *data.Document                 { return m.input }
func (m testMessage) InputChannel() <-chan *data.Document   { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

var _ core.Message = testMessage{}

func TestIsSystemCollection(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"_users_", true},
		{"users", false},
		{"_users", false},
		{"users_", false},
		{"a", false},
		{"__", false},
		{"_a_", true},
		{"", false},
	}
	for _, tc := range tests {
		got := collections.IsSystemCollection(tc.name)
		if got != tc.want {
			t.Errorf("IsSystemCollection(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestNewQueryCommand(t *testing.T) {
	ctx := context.Background()
	q := query.NewQueryBuilder().Build()

	cmd := collections.NewQueryCommand(ctx, "my_collection", &q)

	if cmd.Collection != "my_collection" {
		t.Errorf("Collection = %q, want %q", cmd.Collection, "my_collection")
	}
	if cmd.Context() != ctx {
		t.Error("Context() != ctx")
	}
	if cmd.QueryName() != "collections:document:query" {
		t.Errorf("QueryName() = %q, want %q", cmd.QueryName(), "collections:document:query")
	}
	rc := cmd.ResourceContext()
	m, ok := rc.(map[string]any)
	if !ok {
		t.Fatalf("ResourceContext() type = %T, want map[string]any", rc)
	}
	if m["collection"] != "my_collection" {
		t.Errorf("ResourceContext()[\"collection\"] = %v, want %q", m["collection"], "my_collection")
	}
}

func TestCollectionCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	logger := zap.NewNop()

	createHandler := collections.NewCollectionCreateHandler(p, stubPolicyStore{}, stubRegistry{}, logger)
	deleteHandler := collections.NewCollectionDeleteHandler(p, stubPolicyStore{}, stubRegistry{}, logger)

	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"name":    "test_collection",
			"version": "1.0.0",
			"fields": map[string]any{
				"f1": map[string]any{
					"name": "name",
					"type": "string",
				},
			},
		},
	}, ctx)
	msg := testMessage{name: "create", ctx: ctx, input: input}
	result, err := createHandler(ctx, msg)
	if err != nil {
		t.Fatalf("createHandler: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("createHandler returned nil result or document")
	}
	name, _ := result.Document.Get("name")
	if name != "test_collection" {
		t.Errorf("created collection name = %v, want %q", name, "test_collection")
	}

	exists, err := p.HasCollection(ctx, "test_collection")
	if err != nil {
		t.Fatalf("HasCollection: %v", err)
	}
	if !exists {
		t.Fatal("collection was not created")
	}

	delInput := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"name": "test_collection",
		},
	}, ctx)
	delMsg := testMessage{name: "delete", ctx: ctx, input: delInput}
	delResult, err := deleteHandler(ctx, delMsg)
	if err != nil {
		t.Fatalf("deleteHandler: %v", err)
	}
	if delResult == nil {
		t.Fatal("deleteHandler returned nil result")
	}

	exists, err = p.HasCollection(ctx, "test_collection")
	if err != nil {
		t.Fatalf("HasCollection after delete: %v", err)
	}
	if exists {
		t.Fatal("collection still exists after delete")
	}
}
