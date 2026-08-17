package schedules

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type ScheduleHandlers struct {
	model *ScheduleModel
	live  *LiveSchedule
}

func NewScheduleHandlers(model *ScheduleModel, live *LiveSchedule) *ScheduleHandlers {
	return &ScheduleHandlers{model: model, live: live}
}

func (h *ScheduleHandlers) Create(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	claims, ok := runtimecontext.ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	body, _ := msg.Input().GetOr("payload", nil).(map[string]any)
	if body == nil {
		return nil, fmt.Errorf("payload is required")
	}

	if _, ok := body["message"]; !ok {
		return nil, fmt.Errorf("message is required")
	}
	if _, ok := body["cron"]; !ok {
		return nil, fmt.Errorf("cron is required")
	}

	userID, _ := body["user_id"].(string)
	if userID == "" {
		userID = claims.UserID
	}

	tenantID := runtimecontext.GetTenantID(ctx)

	doc := data.MustNewDocument(body)
	doc.Set("user_id", userID)
	if tenantID != "" {
		doc.Set("tenant_id", tenantID)
	}

	saved, err := h.model.Create(ctx, doc)
	if err != nil {
		return nil, err
	}

	h.live.Register(ctx, saved)

	return dispatch.NewDocumentResultFrom(&ScheduleCreatedView{
		ID:      saved.ID(),
		Message: "schedule created",
	})
}

func (h *ScheduleHandlers) List(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	tenantID := runtimecontext.GetTenantID(ctx)

	docs, err := h.model.ListByTenant(ctx, tenantID, 50, 0)
	if err != nil {
		return nil, err
	}
	return dispatch.NewDocumentsResult(docs), nil
}

func (h *ScheduleHandlers) Get(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	input := msg.Input()
	id, _ := input.GetString("arguments.id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	schedule, err := h.model.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found")
	}
	return &abstract.Result{Document: schedule}, nil
}

func (h *ScheduleHandlers) Update(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	doc := msg.Input()
	id, _ := doc.GetString("arguments.id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	payload, _ := doc.GetOr("payload", nil).(map[string]any)
	if payload == nil {
		return nil, fmt.Errorf("payload is required")
	}

	h.live.UnregisterByID(ctx, id)
	if err := h.model.Update(ctx, id, payload); err != nil {
		return nil, err
	}

	saved, err := h.model.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, fmt.Errorf("schedule not found after update")
	}

	h.live.Register(ctx, saved)

	return dispatch.NewDocumentResultFrom(&MessageOutput{Message: "updated"})
}

func (h *ScheduleHandlers) Delete(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	doc := msg.Input()
	id, _ := doc.GetString("arguments.id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	h.live.UnregisterByID(ctx, id)

	if err := h.model.Delete(ctx, id); err != nil {
		return nil, err
	}

	return dispatch.NewDocumentResultFrom(&MessageOutput{Message: "deleted"})
}
