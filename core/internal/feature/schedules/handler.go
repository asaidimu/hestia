package schedules

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

func NewCreateScheduleHandler(m *ScheduleModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		claims, ok := runtimecontext.ClaimsFromContext(ctx)
		if !ok || claims.UserID == "" {
			return nil, fmt.Errorf("unauthenticated")
		}

		body, _ := msg.Input().GetOr("payload", nil).(map[string]any)
		if body == nil {
			return nil, fmt.Errorf("payload is required")
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
		if _, ok := body["send_at"]; !ok {
			return nil, fmt.Errorf("send_at is required")
		}

		id, err := m.Create(ctx, doc)
		if err != nil {
			return nil, err
		}

		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{
				"id":      id,
				"message": "scheduled message created",
			}),
		}, nil
	}
}

func NewListSchedulesHandler(m *ScheduleModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		tenantID := runtimecontext.GetTenantID(ctx)

		docs, err := m.List(ctx, tenantID, 50, 0)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Documents: docs}, nil
	}
}

func NewGetScheduleHandler(m *ScheduleModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		id, _ := doc.GetString("arguments.id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}

		schedule, err := m.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if schedule == nil {
			return nil, fmt.Errorf("schedule not found")
		}
		return &abstract.Result{Document: schedule}, nil
	}
}

func NewDeleteScheduleHandler(m *ScheduleModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		id, _ := doc.GetString("arguments.id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}

		if err := m.Delete(ctx, id); err != nil {
			return nil, err
		}
		return &abstract.Result{Document: data.MustNewDocument(map[string]any{"message": "deleted"})}, nil
	}
}
