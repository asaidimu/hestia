package schedules_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/schedules/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"go.uber.org/zap"
)

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

func createTestDoc(t *testing.T) *data.Document {
	t.Helper()
	return data.MustNewDocument(map[string]any{
		"user_id": "user-1",
		"message": "system:test:handler",
		"input": map[string]any{
			"token": "abc123",
		},
		"cron": "@every 1h",
	})
}

func TestScheduleModelCreateAndList(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := model.NewScheduleModel(p)

	doc, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if doc.ID() == "" {
		t.Fatal("expected non-empty id")
	}

	docs, err := m.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	gotMessage, _ := docs[0].GetString("message")
	if gotMessage != "system:test:handler" {
		t.Errorf("message = %q, want %q", gotMessage, "system:test:handler")
	}
}

func TestScheduleModelGet(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := model.NewScheduleModel(p)

	created, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	doc, err := m.Get(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}
}

func TestScheduleModelGetNotFound(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := model.NewScheduleModel(p)

	doc, err := m.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if doc != nil {
		t.Fatal("expected nil for nonexistent id")
	}
}

func TestScheduleModelDelete(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := model.NewScheduleModel(p)

	created, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.Delete(ctx, created.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	doc, err := m.Get(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if doc != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestScheduleModelUpdate(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := model.NewScheduleModel(p)

	created, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.Update(ctx, created.ID(), map[string]any{"cron": "@every 30m"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	doc, err := m.Get(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	cron, _ := doc.GetString("cron")
	if cron != "@every 30m" {
		t.Errorf("cron = %q, want %q", cron, "@every 30m")
	}
}

func TestCreateScheduleHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)
	result, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:handler",
		Input:   map[string]any{"token": "xyz"},
		Cron:    "@every 1h",
	})
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Message != "schedule created" {
		t.Errorf("message = %q, want %q", result.Message, "schedule created")
	}
}

func TestCreateScheduleHandlerMissingCron(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)
	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:handler",
	})
	if err == nil {
		t.Fatal("expected error for missing cron")
	}
}

func TestListSchedulesHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)

	_, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)
	docs, err := svc.List(ctx, &testMessage{}, &model.ScheduleListInput{})
	if err != nil {
		t.Fatalf("list handler: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
}

func TestGetScheduleHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)

	created, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)
	result, err := svc.Get(ctx, &testMessage{}, &model.ScheduleGetInput{ID: created.ID()})
	if err != nil {
		t.Fatalf("get handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Message != "system:test:handler" {
		t.Errorf("message = %q, want %q", result.Message, "system:test:handler")
	}
}

func TestGetScheduleHandlerNotFound(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)
	_, err := svc.Get(ctx, &testMessage{}, &model.ScheduleGetInput{ID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestDeleteScheduleHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)

	created, err := m.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)
	result, err := svc.Delete(ctx, &testMessage{}, &model.ScheduleDeleteInput{ID: created.ID()})
	if err != nil {
		t.Fatalf("delete handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestUpdateScheduleHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(model.NewScheduleModel(p), live)

	// Create first
	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:handler",
		Cron:    "@every 1h",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Get the ID from the created schedule
	docs, err := m.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected at least one schedule")
	}
	id := docs[0].ID()

	// Update
	result, err := svc.Update(ctx, &testMessage{}, &model.ScheduleUpdateInput{
		ID:   id,
		Cron: "@every 30m",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify
	saved, _ := m.Get(ctx, id)
	cron, _ := saved.GetString("cron")
	if cron != "@every 30m" {
		t.Errorf("cron after update = %q, want %q", cron, "@every 30m")
	}
}

func TestLiveScheduleRegistersAndDispatches(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	m := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	sched.Start()
	defer sched.Stop()

	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	doc := data.MustNewDocument(map[string]any{
		"user_id": "user-1",
		"message": "system:test:handler",
		"input":   map[string]any{"token": "abc"},
		"cron":    "@every 1h",
	})
	created, err := m.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	saved, _ := m.Get(ctx, created.ID())
	live.Register(ctx, saved)

	// Verify the job was registered
	jobs := sched.List()
	found := false
	for _, j := range jobs {
		if j.Name == "schedule:"+created.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("job was not registered")
	}

	live.UnregisterByID(ctx, created.ID())
	jobs = sched.List()
	for _, j := range jobs {
		if j.Name == "schedule:"+created.ID() {
			t.Fatal("job should have been removed")
		}
	}
}
