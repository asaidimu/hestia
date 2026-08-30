package schedules_test

import (
	"context"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/schedules/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
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

func newTestModel(t *testing.T) *model.SystemScheduledMessagess {
	t.Helper()
	model.DangerouslyResetSystemScheduledMessagessModel()
	m, err := model.InitSystemScheduledMessagessModel(testutil.NewPersistence(t), nil)
	if err != nil {
		t.Fatalf("InitSystemScheduledMessagessModel: %v", err)
	}
	return m
}

func TestScheduleModelCreateAndList(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	doc, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if doc.ID() == "" {
		t.Fatal("expected non-empty id")
	}

	docs, err := m.ListSchedules(ctx)
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
	m := newTestModel(t)

	created, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	doc, err := m.GetSchedule(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil document")
	}
}

func TestScheduleModelGetNotFound(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	doc, err := m.GetSchedule(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if doc != nil {
		t.Fatal("expected nil for nonexistent id")
	}
}

func TestScheduleModelDelete(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	created, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.DeleteSchedule(ctx, created.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	doc, err := m.GetSchedule(ctx, created.ID())
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if doc != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestScheduleModelUpdate(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	created, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := m.UpdateSchedule(ctx, created.ID(), map[string]any{"cron": "@every 30m"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	doc, err := m.GetSchedule(ctx, created.ID())
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
	m := newTestModel(t)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
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
	m := newTestModel(t)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:handler",
	})
	if err == nil {
		t.Fatal("expected error for missing cron")
	}
}

func TestListSchedulesHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-1"})
	m := newTestModel(t)

	_, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	docs, err := svc.List(ctx, &testMessage{}, &model.ScheduleListInput{})
	if err != nil {
		t.Fatalf("list handler: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
}

func TestGetScheduleHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-1"})
	m := newTestModel(t)

	created, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
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
	m := newTestModel(t)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	_, err := svc.Get(ctx, &testMessage{}, &model.ScheduleGetInput{ID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestDeleteScheduleHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-1"})
	m := newTestModel(t)

	created, err := m.CreateSchedule(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	result, err := svc.Delete(ctx, &testMessage{}, &model.ScheduleDeleteInput{ID: created.ID()})
	if err != nil {
		t.Fatalf("delete handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// S-12 regression: schedules created by user-1 must be unreadable and
// undeletable by another authenticated user holding a valid ID.
func TestGetScheduleHandlerForbidden(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-2"})
	m := newTestModel(t)

	created, err := m.CreateSchedule(context.Background(), createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	_, err = svc.Get(ctx, &testMessage{}, &model.ScheduleGetInput{ID: created.ID()})
	if err == nil {
		t.Fatal("expected ownership denial for another user's schedule")
	}
}

func TestDeleteScheduleHandlerForbidden(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-2"})
	m := newTestModel(t)

	created, err := m.CreateSchedule(context.Background(), createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	_, err = svc.Delete(ctx, &testMessage{}, &model.ScheduleDeleteInput{ID: created.ID()})
	if err == nil {
		t.Fatal("expected ownership denial for another user's schedule")
	}

	doc, err := m.GetSchedule(context.Background(), created.ID())
	if err != nil {
		t.Fatalf("re-fetch after denied delete: %v", err)
	}
	if doc == nil {
		t.Fatal("schedule must survive a denied delete")
	}
}

func TestUpdateScheduleHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	m := newTestModel(t)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)

	// Create first
	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:handler",
		Cron:    "@every 1h",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Get the ID from the created schedule
	docs, err := m.ListSchedules(ctx)
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
	saved, _ := m.GetSchedule(ctx, id)
	cron, _ := saved.GetString("cron")
	if cron != "@every 30m" {
		t.Errorf("cron after update = %q, want %q", cron, "@every 30m")
	}
}

func TestLiveScheduleRegistersAndDispatches(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
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
	created, err := m.CreateSchedule(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	saved, _ := m.GetSchedule(ctx, created.ID())
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

type schedValidateDTO struct {
	Token string `anansi:"token,required=true" input:"payload.token"`
	Count int    `anansi:"count" input:"payload.count"`
}

func newValidatingService(t *testing.T) *schedules.SchedulesService {
	t.Helper()
	m := newTestModel(t)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)
	regs := []abstract.MessageRegistration{
		{
			Name: "system:test:schema",
			Input: abstract.Input{
				Schema: dispatch.SchemaFromTypeWithTag[schedValidateDTO]("input"),
			},
		},
	}
	return schedules.NewSchedulesServiceWithRegistrationsForTest(m, live, regs)
}

func TestCreateRejectsInvalidCron(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:schema",
		Cron:    "not a cron",
	})
	if err == nil {
		t.Fatal("expected error for invalid cron")
	}
	if !strings.Contains(err.Error(), "invalid cron") {
		t.Errorf("expected cron error, got: %v", err)
	}
}

func TestCreateRejectsUnknownMessage(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:does:not:exist",
		Cron:    "@every 1h",
	})
	if err == nil {
		t.Fatal("expected error for unregistered message")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("expected unknown-message error, got: %v", err)
	}
}

func TestCreateRejectsInputViolatingSchema(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	_, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:schema",
		Cron:    "@every 1h",
		Input:   map[string]any{"count": 3}, // missing required token
	})
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "does not match its schema") {
		t.Errorf("expected schema error, got: %v", err)
	}
}

func TestCreateAcceptsSchemaConformingInput(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	result, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:schema",
		Cron:    "@every 1h",
		Input:   map[string]any{"token": "abc", "count": 1},
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result == nil || result.ID == "" {
		t.Fatal("expected created schedule with ID")
	}
}

func TestUpdatePreservesUntouchedFields(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	created, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:schema",
		Cron:    "@every 1h",
		Input:   map[string]any{"token": "keepme", "count": 7},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	disabled := false
	if _, err := svc.Update(ctx, &testMessage{}, &model.ScheduleUpdateInput{
		ID:       created.ID,
		Cron:     "@every 30m",
		Disabled: &disabled,
	}); err != nil {
		t.Fatalf("update with cron+disabled only: %v", err)
	}

	docs, err := svc.List(ctx, &testMessage{}, &model.ScheduleListInput{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	saved := docs[0]
	gotMessage, _ := saved.GetString("message")
	if gotMessage != "system:test:schema" {
		t.Errorf("message was blanked: got %q", gotMessage)
	}
	raw, _ := saved.Get("input")
	inputMap, ok := raw.(map[string]any)
	if !ok || inputMap["token"] != "keepme" {
		t.Errorf("input was blanked: got %#v", raw)
	}
	gotDisabled, _ := saved.GetBool("disabled")
	if gotDisabled {
		t.Error("disabled should remain false")
	}
	gotCron, _ := saved.GetString("cron")
	if gotCron != "@every 30m" {
		t.Errorf("cron not updated: got %q", gotCron)
	}
}

func TestUpdateCanDisable(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	created, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:schema",
		Cron:    "@every 1h",
		Input:   map[string]any{"token": "t"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	truth := true
	if _, err := svc.Update(ctx, &testMessage{}, &model.ScheduleUpdateInput{
		ID:       created.ID,
		Disabled: &truth,
	}); err != nil {
		t.Fatalf("disable update: %v", err)
	}

	docs, err := svc.List(ctx, &testMessage{}, &model.ScheduleListInput{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	gotDisabled, _ := docs[0].GetBool("disabled")
	if !gotDisabled {
		t.Error("expected disabled=true after update")
	}
}

func TestUpdateValidatesMergedTarget(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	svc := newValidatingService(t)

	created, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:schema",
		Cron:    "@every 1h",
		Input:   map[string]any{"token": "ok"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Replacing input with a payload missing the required token must fail.
	_, err = svc.Update(ctx, &testMessage{}, &model.ScheduleUpdateInput{
		ID:   created.ID,
		Cron: "@every 2h",
		Input: map[string]any{"count": 1},
	})
	if err == nil || !strings.Contains(err.Error(), "does not match its schema") {
		t.Errorf("expected merged-target schema error, got: %v", err)
	}
}
