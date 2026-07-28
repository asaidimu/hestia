package notifications

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

func NewListNotificationsHandler(m *NotificationModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		claims, ok := runtimecontext.ClaimsFromContext(ctx)
		if !ok || claims.UserID == "" {
			return nil, fmt.Errorf("unauthenticated")
		}
		tenantID := runtimecontext.GetTenantID(ctx)

		docs, err := m.List(ctx, claims.UserID, tenantID, 50, 0)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Documents: docs}, nil
	}
}

func NewMarkReadHandler(m *NotificationModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		claims, ok := runtimecontext.ClaimsFromContext(ctx)
		if !ok || claims.UserID == "" {
			return nil, fmt.Errorf("unauthenticated")
		}

		doc := msg.Input()
		notificationID, _ := doc.GetString("arguments.notification_id")
		if notificationID == "" {
			return nil, fmt.Errorf("notification_id is required")
		}

		if err := m.MarkRead(ctx, notificationID); err != nil {
			return nil, err
		}
		return &abstract.Result{Document: data.MustNewDocument(map[string]any{"message": "ok"})}, nil
	}
}

func NewMarkAllReadHandler(m *NotificationModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		claims, ok := runtimecontext.ClaimsFromContext(ctx)
		if !ok || claims.UserID == "" {
			return nil, fmt.Errorf("unauthenticated")
		}
		tenantID := runtimecontext.GetTenantID(ctx)

		if err := m.MarkAllRead(ctx, claims.UserID, tenantID); err != nil {
			return nil, err
		}
		return &abstract.Result{Document: data.MustNewDocument(map[string]any{"message": "ok"})}, nil
	}
}

func NewCountUnreadHandler(m *NotificationModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		claims, ok := runtimecontext.ClaimsFromContext(ctx)
		if !ok || claims.UserID == "" {
			return nil, fmt.Errorf("unauthenticated")
		}
		tenantID := runtimecontext.GetTenantID(ctx)

		count, err := m.CountUnread(ctx, claims.UserID, tenantID)
		if err != nil {
			return nil, err
		}
		result := data.MustNewDocument(map[string]any{"count": count})
		return &abstract.Result{Document: result}, nil
	}
}
