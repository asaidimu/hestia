package workflows_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/workflows"
	"github.com/asaidimu/hestia/core/system/workflows/model"

	hermescore "github.com/asaidimu/hermes/pkg/core"
	hermesruntime "github.com/asaidimu/hermes/pkg/runtime"
	hermesstore "github.com/asaidimu/hermes/pkg/store"
	_ "github.com/asaidimu/hermes/pkg/nodes" // register built-in node types
)

type nopLogger struct{}

func (l *nopLogger) Debug(msg string, keysAndValues ...any) {}
func (l *nopLogger) Info(msg string, keysAndValues ...any)  {}
func (l *nopLogger) Warn(msg string, keysAndValues ...any)  {}
func (l *nopLogger) Error(msg string, keysAndValues ...any) {}
func (l *nopLogger) With(keysAndValues ...any) hermescore.Logger { return l }

func testMsg(name string, input *data.Document) abstract.Message {
	return &testMessage{name: name, ctx: context.Background(), input: input}
}

type testMessage struct {
	name  string
	ctx   context.Context
	input *data.Document
}

func (m *testMessage) ID() string                            { return "" }
func (m *testMessage) Name() string                          { return m.name }
func (m *testMessage) Context() context.Context               { return m.ctx }
func (m *testMessage) Input() data.Documenter                  { return m.input }
func (m *testMessage) InputChannel() <-chan data.Documenter     { return nil }
func (m *testMessage) BlobInputChannel() <-chan abstract.Blob  { return nil }
func (m *testMessage) TenantID() string                       { return "" }
func (m *testMessage) TraceID() string                        { return "" }
func (m *testMessage) RequestID() string                      { return "" }
func (m *testMessage) SourceIP() string                       { return "" }
func (m *testMessage) UserAgent() string                      { return "" }
func (m *testMessage) ResourceID() string                     { return "" }
func (m *testMessage) SessionID() string                      { return "" }

func newTestModel(t *testing.T) *model.SystemWorkflowDefinitions {
	t.Helper()
	model.DangerouslyResetSystemWorkflowDefinitionsModel()
	m, err := model.InitSystemWorkflowDefinitionsModel(testutil.NewPersistence(t), nil)
	if err != nil {
		t.Fatalf("InitSystemWorkflowDefinitionsModel: %v", err)
	}
	return m
}

func newTestRuntime(t *testing.T) *hermesruntime.WorkflowRuntime {
	t.Helper()
	storeFactory := func() (hermesstore.Store, error) {
		return hermesstore.NewMemoryStore(map[string]any{}), nil
	}
	return hermesruntime.NewWorkflowRuntime(hermesruntime.Options{
		StoreFactory: storeFactory,
		Logger:       &nopLogger{},
	})
}

func newTestService(t *testing.T) (*workflows.WorkflowsService, *model.SystemWorkflowDefinitions) {
	t.Helper()
	m := newTestModel(t)
	rt := newTestRuntime(t)
	svc := workflows.NewWorkflowsServiceForTest(m, rt)
	return svc, m
}

func TestWorkflowModelCreateAndGet(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	doc := data.MustNewDocument(map[string]any{
		"name":  "test-workflow",
		"nodes": []map[string]any{{"id": "n1", "type": "executable", "kind": "trigger"}},
		"edges": []map[string]any{},
	})

	created, err := m.CreateDefinition(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID() == "" {
		t.Fatal("expected non-empty id")
	}

	got, err := m.GetDefinition(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil document")
	}
	name, _ := got.GetString("name")
	if name != "test-workflow" {
		t.Errorf("name = %q, want %q", name, "test-workflow")
	}
}

func TestWorkflowModelGetNotFound(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	doc, err := m.GetDefinition(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if doc != nil {
		t.Fatal("expected nil for nonexistent id")
	}
}

func TestWorkflowModelDelete(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	doc := data.MustNewDocument(map[string]any{
		"name":  "to-delete",
		"nodes": []map[string]any{},
		"edges": []map[string]any{},
	})
	created, err := m.CreateDefinition(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.DeleteDefinition(ctx, created.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := m.GetDefinition(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestWorkflowModelList(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	for _, name := range []string{"wf-1", "wf-2", "wf-3"} {
		doc := data.MustNewDocument(map[string]any{
			"name":  name,
			"nodes": []map[string]any{},
			"edges": []map[string]any{},
		})
		if _, err := m.CreateDefinition(ctx, doc); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	docs, err := m.ListDefinitions(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 documents, got %d", len(docs))
	}
}

func TestWorkflowModelUpdate(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	doc := data.MustNewDocument(map[string]any{
		"name":  "original",
		"nodes": []map[string]any{},
		"edges": []map[string]any{},
	})
	created, err := m.CreateDefinition(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.UpdateDefinition(ctx, created.ID(), map[string]any{
		"name": "updated",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := m.GetDefinition(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	name, _ := got.GetString("name")
	if name != "updated" {
		t.Errorf("name = %q, want %q", name, "updated")
	}
}

// ---------------------------------------------------------------------------
// Handler tests
// ---------------------------------------------------------------------------

func TestCompileHandler(t *testing.T) {
	svc, _ := newTestService(t)

	result, err := svc.Compile(context.Background(), testMsg("test", nil), &workflows.WorkflowCompileInput{
		Nodes: []map[string]any{
			{"id": "trigger-1", "type": "executable", "kind": "trigger", "config": map[string]any{"event": "__manual__"}},
			{"id": "code-1", "type": "executable", "kind": "code", "config": map[string]any{"code": "state.x = 1"}},
		},
		Edges: []map[string]any{
			{"id": "e1", "source": "trigger-1", "target": "code-1", "role": "flow"},
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if result.WorkflowID == "" {
		t.Error("expected non-empty workflow_id")
	}
	if result.Triggers != 1 {
		t.Errorf("triggers = %d, want 1", result.Triggers)
	}
	if result.Pipelines != 1 {
		t.Errorf("pipelines = %d, want 1", result.Pipelines)
	}
}

func TestCompileHandlerInvalidNodes(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Compile(context.Background(), testMsg("test", nil), &workflows.WorkflowCompileInput{
		Nodes: []map[string]any{
			{"id": ""}, // missing id
		},
		Edges: []map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for invalid nodes")
	}
}

func TestRegisterAndListDefinitions(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Register
	regResult, err := svc.Register(ctx, testMsg("test", nil), &workflows.WorkflowRegisterInput{
		Name: "my-workflow",
		Nodes: []map[string]any{
			{"id": "t1", "type": "executable", "kind": "trigger", "config": map[string]any{"event": "__manual__"}},
		},
		Edges: []map[string]any{},
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if regResult.WorkflowID == "" {
		t.Error("expected non-empty workflow_id")
	}

	// List
	docs, err := svc.ListDefinitions(ctx, testMsg("test", nil), &workflows.WorkflowDefinitionListInput{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(docs))
	}
}

func TestGetDefinition(t *testing.T) {
	svc, m := newTestService(t)
	ctx := context.Background()

	// Create directly via model
	doc := data.MustNewDocument(map[string]any{
		"name":  "get-test",
		"nodes": []map[string]any{},
		"edges": []map[string]any{},
	})
	created, err := m.CreateDefinition(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	result, err := svc.GetDefinition(ctx, testMsg("test", nil), &workflows.WorkflowDefinitionGetInput{
		ID: created.ID(),
	})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result.Name != "get-test" {
		t.Errorf("name = %q, want %q", result.Name, "get-test")
	}
}

func TestGetDefinitionNotFound(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetDefinition(context.Background(), testMsg("test", nil), &workflows.WorkflowDefinitionGetInput{
		ID: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestDeleteDefinition(t *testing.T) {
	svc, m := newTestService(t)
	ctx := context.Background()

	doc := data.MustNewDocument(map[string]any{
		"name":  "delete-test",
		"nodes": []map[string]any{},
		"edges": []map[string]any{},
	})
	created, err := m.CreateDefinition(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	result, err := svc.Deregister(ctx, testMsg("test", nil), &workflows.WorkflowDefinitionDeleteInput{
		ID: created.ID(),
	})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result.Message != "workflow deregistered" {
		t.Errorf("message = %q, want %q", result.Message, "workflow deregistered")
	}

	// Verify gone
	got, err := m.GetDefinition(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestDeleteDefinitionNotFound(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Deregister(context.Background(), testMsg("test", nil), &workflows.WorkflowDefinitionDeleteInput{
		ID: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestRunHandler(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	result, err := svc.Run(ctx, testMsg("test", nil), &workflows.WorkflowRunInput{
		Nodes: []map[string]any{
			{"id": "t1", "type": "executable", "kind": "trigger", "config": map[string]any{"event": "__manual__"}},
		},
		Edges: []map[string]any{},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.RunID == "" {
		t.Error("expected non-empty run_id")
	}
}

func TestEmitEventHandler(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.EmitEvent(ctx, testMsg("test", nil), &workflows.WorkflowEventInput{
		Type:    "test:event",
		Payload: map[string]any{"key": "value"},
	})
	if err != nil {
		t.Fatalf("EmitEvent: %v", err)
	}
}

func TestEmitEventMissingType(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.EmitEvent(ctx, testMsg("test", nil), &workflows.WorkflowEventInput{})
	if err == nil {
		t.Fatal("expected error for missing event type")
	}
}

func TestAbortHandler(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Abort non-existent run should not error (hermes silently ignores)
	_, err := svc.Abort(ctx, testMsg("test", nil), &workflows.WorkflowAbortInput{
		RunID: "nonexistent",
	})
	if err != nil {
		t.Fatalf("Abort: %v", err)
	}
}
