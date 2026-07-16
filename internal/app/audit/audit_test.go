package audit_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/audit"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

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

var _ core.Message = testMessage{}

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	m.Run()
}

func queryHandler(p base.Persistence) core.MessageHandler {
	regs := audit.Registrations(audit.Dependencies{Persist: p})
	for _, reg := range regs {
		if reg.Name == "system:audit:log:query" {
			return reg.Handler
		}
	}
	return nil
}

func TestDefaultOperations(t *testing.T) {
	ops := audit.DefaultOperations()
	if len(ops) == 0 {
		t.Fatal("DefaultOperations() returned empty slice")
	}
}

func TestAuditInsert(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := audit.NewAuditModel(p)

	entry := core.AuditEntry{
		EventName:    "test:msg",
		ActorID:      "user-1",
		ActorType:    core.ActorTypeUser,
		Operation:    core.OperationExecute,
		ResourceType: "test",
		Status:       core.AuditStatusSuccess,
		LatencyMs:    100,
		ServiceName:  "hestia",
	}

	if err := model.Insert(ctx, entry); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	handler := queryHandler(p)
	if handler == nil {
		t.Fatal("system:audit:log:query handler not found")
	}

	msg := testMessage{ctx: ctx, input: data.MustNewDocument(nil, ctx)}
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("Query handler failed: %v", err)
	}
	if result.Page == nil {
		t.Fatal("expected non-nil Page")
	}
	if len(result.Page.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(result.Page.Documents))
	}

	doc := result.Page.Documents[0]
	name, _ := doc.Get("event_name")
	if name != "test:msg" {
		t.Errorf("event_name = %v, want test:msg", name)
	}
	uid, _ := doc.Get("actor_id")
	if uid != "user-1" {
		t.Errorf("actor_id = %v, want user-1", uid)
	}
	status, _ := doc.Get("status")
	if status != "success" {
		t.Errorf("status = %v, want success", status)
	}
}

func TestAuditQueryHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := audit.NewAuditModel(p)

	entries := []core.AuditEntry{
		{EventName: "msg:1", ActorID: "user-a", ActorType: core.ActorTypeUser, Operation: core.OperationExecute, ResourceType: "test", Status: core.AuditStatusSuccess, LatencyMs: 10, ServiceName: "hestia"},
		{EventName: "msg:2", ActorID: "user-b", ActorType: core.ActorTypeUser, Operation: core.OperationExecute, ResourceType: "test", Status: core.AuditStatusDenied, LatencyMs: 20, ServiceName: "hestia"},
		{EventName: "msg:3", ActorID: "user-a", ActorType: core.ActorTypeUser, Operation: core.OperationExecute, ResourceType: "test", Status: core.AuditStatusSuccess, LatencyMs: 30, ServiceName: "hestia"},
	}
	for _, e := range entries {
		if err := model.Insert(ctx, e); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	handler := queryHandler(p)
	if handler == nil {
		t.Fatal("system:audit:log:query handler not found")
	}

	msg := testMessage{ctx: ctx, input: data.MustNewDocument(nil, ctx)}
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("Query handler failed: %v", err)
	}
	if result.Page == nil {
		t.Fatal("expected non-nil Page")
	}
	if len(result.Page.Documents) != 3 {
		t.Errorf("got %d documents, want 3", len(result.Page.Documents))
	}
}

func TestAuditQueryHandlerWithLimit(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := audit.NewAuditModel(p)

	for i := 0; i < 10; i++ {
		entry := core.AuditEntry{
			EventName:    "test:msg",
			ActorID:      "user-1",
			ActorType:    core.ActorTypeUser,
			Operation:    core.OperationExecute,
			ResourceType: "test",
			Status:       core.AuditStatusSuccess,
			LatencyMs:    int64(i),
			ServiceName:  "hestia",
		}
		if err := model.Insert(ctx, entry); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	handler := queryHandler(p)
	if handler == nil {
		t.Fatal("system:audit:log:query handler not found")
	}

	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"query": map[string]any{
				"pagination": map[string]any{
					"type":          "offset",
					"offset":        0,
					"limit":         3,
					"include_total": true,
				},
			},
		},
	}, ctx)

	msg := testMessage{ctx: ctx, input: input}
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("Query handler failed: %v", err)
	}
	if result.Page == nil {
		t.Fatal("expected non-nil Page")
	}
	if len(result.Page.Documents) > 3 {
		t.Errorf("got %d documents, want <= 3", len(result.Page.Documents))
	}
	if len(result.Page.Documents) == 0 {
		t.Error("expected at least one document")
	}
}
