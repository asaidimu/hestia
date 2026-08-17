package notifications

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
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
		return dispatch.NewDocumentsResult(docs), nil
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
		return dispatch.NewDocumentResultFrom(&MessageOutput{Message: "ok"})
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
		return dispatch.NewDocumentResultFrom(&MessageOutput{Message: "ok"})
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
		return dispatch.NewDocumentResultFrom(&UnreadCountDocument{Count: int64(count)})
	}
}
