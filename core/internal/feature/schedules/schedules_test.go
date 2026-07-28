package schedules_test

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/schedules"
	"github.com/asaidimu/hestia/core/internal/testutil"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func testMsg(name string, input *data.Document) abstract.Message {
	return dispatch.NewMessage(name, context.Background(), input)
}

func createTestDoc(t *testing.T, sendAt int64) *data.Document {
	t.Helper()
	return data.MustNewDocument(map[string]any{
		"user_id": "user-1",
		"type":    "password_reset",
		"channel": "in_app",
		"data": map[string]any{
			"token": "abc123",
		},
		"send_at": sendAt,
	}, context.Background())
}

func TestScheduleModelCreateAndList(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	docs, err := model.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	gotType, _ := docs[0].GetString("type")
	if gotType != "password_reset" {
		t.Errorf("type = %q, want %q", gotType, "password_reset")
	}
}

func TestScheduleModelGet(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	doc, err := model.Get(ctx, id)
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

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := model.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	doc, err := model.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if doc != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestFindDue(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	_, err := model.Create(ctx, createTestDoc(t, time.Now().Add(-1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create past: %v", err)
	}
	_, err = model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create future: %v", err)
	}

	due, err := model.FindDue(ctx)
	if err != nil {
		t.Fatalf("FindDue: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due document, got %d", len(due))
	}
}

func TestMarkSent(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(-1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := model.MarkSent(ctx, id, nil); err != nil {
		t.Fatalf("MarkSent (success): %v", err)
	}

	doc, err := model.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	sent, _ := doc.Get("sent")
	if sent != true {
		t.Errorf("sent = %v, want true", sent)
	}
	sentAt, _ := doc.Get("sent_at")
	if sentAt == nil || sentAt.(int64) == 0 {
		t.Errorf("sent_at not set, got %v", sentAt)
	}
}

func TestMarkSentError(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(-1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	sendErr := context.DeadlineExceeded
	if err := model.MarkSent(ctx, id, sendErr); err != nil {
		t.Fatalf("MarkSent (error): %v", err)
	}

	doc, err := model.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	errVal, _ := doc.GetString("error_")
	if errVal != context.DeadlineExceeded.Error() {
		t.Errorf("error_ = %q, want %q", errVal, context.DeadlineExceeded.Error())
	}
}

func TestCreateScheduleHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	handler := schedules.NewCreateScheduleHandler(model)
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"user_id": "target-user",
			"type":    "password_reset",
			"channel": "in_app",
			"data":    map[string]any{"token": "xyz"},
			"send_at": time.Now().Add(1 * time.Hour).UnixMilli(),
		},
	}, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:create", ctx, input)
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil document")
	}
	msgText, _ := result.Document.GetString("message")
	if msgText != "scheduled message created" {
		t.Errorf("message = %q, want %q", msgText, "scheduled message created")
	}
}

func TestCreateScheduleHandlerMissingSendAt(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	handler := schedules.NewCreateScheduleHandler(model)
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"type": "password_reset",
		},
	}, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:create", ctx, input)
	_, err := handler(ctx, msg)
	if err == nil {
		t.Fatal("expected error for missing send_at")
	}
}

func TestListSchedulesHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	_, err := model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	handler := schedules.NewListSchedulesHandler(model)
	input := data.MustNewDocument(nil, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:list", ctx, input)
	result, err := handler(ctx, msg)
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

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	handler := schedules.NewGetScheduleHandler(model)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"id": id},
	}, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:get", ctx, input)
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("get handler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil document")
	}
	gotType, _ := result.Document.GetString("type")
	if gotType != "password_reset" {
		t.Errorf("type = %q, want %q", gotType, "password_reset")
	}
}

func TestGetScheduleHandlerNotFound(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	handler := schedules.NewGetScheduleHandler(model)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"id": "nonexistent"},
	}, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:get", ctx, input)
	_, err := handler(ctx, msg)
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestDeleteScheduleHandler(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	model := schedules.NewScheduleModel(p)

	id, err := model.Create(ctx, createTestDoc(t, time.Now().Add(1*time.Hour).UnixMilli()))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	handler := schedules.NewDeleteScheduleHandler(model)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"id": id},
	}, ctx)
	msg := dispatch.NewMessage("system:schedules:schedule:delete", ctx, input)
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("delete handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
