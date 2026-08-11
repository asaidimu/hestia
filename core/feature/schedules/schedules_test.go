package schedules_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/schedules"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"go.uber.org/zap"
)

func testMsg(name string, input *data.Document) abstract.Message {
	return dispatch.NewMessage(name, context.Background(), input)
}

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
	model := schedules.NewScheduleModel(p)

	doc, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if doc.ID() == "" {
		t.Fatal("expected non-empty id")
	}

	docs, err := model.List(ctx)
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
	model := schedules.NewScheduleModel(p)

	created, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	doc, err := model.Get(ctx, created.ID())
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
	model := schedules.NewScheduleModel(p)

	doc, err := model.Get(ctx, "nonexistent")
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
	model := schedules.NewScheduleModel(p)

	created, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := model.Delete(ctx, created.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	doc, err := model.Get(ctx, created.ID())
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
	model := schedules.NewScheduleModel(p)

	created, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := model.Update(ctx, created.ID(), map[string]any{"cron": "@every 30m"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	doc, err := model.Get(ctx, created.ID())
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
	model := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	h := schedules.NewScheduleHandlers(model, live)
	input := testutil.InputDoc(t, schedules.ScheduleCreateInputSchema(), `{
		"payload": {
			"message": "system:test:handler",
			"input": {"token": "xyz"},
			"cron": "@every 1h"
		}
	}`)
	msg := dispatch.NewMessage("system:schedules:schedule:create", ctx, input)
	result, err := h.Create(ctx, msg)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil document")
	}
	msgText, _ := result.Document.GetString("message")
	if msgText != "schedule created" {
		t.Errorf("message = %q, want %q", msgText, "schedule created")
	}
}

func TestCreateScheduleHandlerMissingCron(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	h := schedules.NewScheduleHandlers(model, live)
	input := testutil.InputDoc(t, schedules.ScheduleCreateInputSchema(), `{
		"payload": {"message": "system:test:handler"}
	}`)
	msg := dispatch.NewMessage("system:schedules:schedule:create", ctx, input)
	_, err := h.Create(ctx, msg)
	if err == nil {
		t.Fatal("expected error for missing cron")
	}
}

func TestListSchedulesHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	_, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	h := schedules.NewScheduleHandlers(model, live)
	input := data.MustNewDocument(nil, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:list", ctx, input)
	result, err := h.List(ctx, msg)
	if err != nil {
		t.Fatalf("list handler: %v", err)
	}
	if len(result.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(result.Documents))
	}
}

func TestGetScheduleHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	created, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	h := schedules.NewScheduleHandlers(model, live)
	input := testutil.InputDoc(t, schedules.ScheduleGetInputSchema(), fmt.Sprintf(`{
		"arguments": {"id": %q}
	}`, created.ID()))
	msg := dispatch.NewMessage("system:schedules:schedule:get", ctx, input)
	result, err := h.Get(ctx, msg)
	if err != nil {
		t.Fatalf("get handler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil document")
	}
	gotMessage, _ := result.Document.GetString("message")
	if gotMessage != "system:test:handler" {
		t.Errorf("message = %q, want %q", gotMessage, "system:test:handler")
	}
}

func TestGetScheduleHandlerNotFound(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	h := schedules.NewScheduleHandlers(model, live)
	input := testutil.InputDoc(t, schedules.ScheduleGetInputSchema(), `{
		"arguments": {"id": "nonexistent"}
	}`)
	msg := dispatch.NewMessage("system:schedules:schedule:get", ctx, input)
	_, err := h.Get(ctx, msg)
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestDeleteScheduleHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	created, err := model.Create(ctx, createTestDoc(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	h := schedules.NewScheduleHandlers(model, live)
	input := testutil.InputDoc(t, schedules.ScheduleDeleteInputSchema(), fmt.Sprintf(`{
		"arguments": {"id": %q}
	}`, created.ID()))
	msg := dispatch.NewMessage("system:schedules:schedule:delete", ctx, input)
	result, err := h.Delete(ctx, msg)
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
	model := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	// Create first
	h := schedules.NewScheduleHandlers(model, live)
	createInput := testutil.InputDoc(t, schedules.ScheduleCreateInputSchema(), `{
		"payload": {
			"message": "system:test:handler",
			"cron": "@every 1h"
		}
	}`)
	createMsg := dispatch.NewMessage("system:schedules:schedule:create", ctx, createInput)
	createResult, err := h.Create(ctx, createMsg)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id, _ := createResult.Document.GetString("id")

	// Update
	updateInput := testutil.InputDoc(t, schedules.ScheduleUpdateInputSchema(), fmt.Sprintf(`{
		"arguments": {"id": %q},
		"payload": {"cron": "@every 30m"}
	}`, id))
	updateMsg := dispatch.NewMessage("system:schedules:schedule:update", ctx, updateInput)
	result, err := h.Update(ctx, updateMsg)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify
	saved, _ := model.Get(ctx, id)
	cron, _ := saved.GetString("cron")
	if cron != "@every 30m" {
		t.Errorf("cron after update = %q, want %q", cron, "@every 30m")
	}
}

func TestLiveScheduleRegistersAndDispatches(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)
	log := zap.NewNop()
	sched := scheduler.New(log)
	sched.Start()
	defer sched.Stop()

	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(model, sched, disp, log)

	doc := data.MustNewDocument(map[string]any{
		"user_id": "user-1",
		"message": "system:test:handler",
		"input":   map[string]any{"token": "abc"},
		"cron":    "@every 1h",
	})
	created, err := model.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	saved, _ := model.Get(ctx, created.ID())
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
