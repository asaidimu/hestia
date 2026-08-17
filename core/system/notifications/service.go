package notifications

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/system/notifications/model"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
)

// NotificationsService is the service for the in-app notifications domain. It
// wraps the hand-rolled NotificationModel (raw persistence over the
// _notifications_ collection) rather than the generated collection.
type NotificationsService struct {
	model *model.NotificationModel
}

func NewNotificationsService(rt abstract.Container) (*NotificationsService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)

	return &NotificationsService{model: model.NewNotificationModel(persist)}, nil
}

func userIDFrom(ctx context.Context, msg abstract.Message) (string, error) {
	claims, ok := runtimecontext.ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return "", fmt.Errorf("unauthenticated")
	}
	return claims.UserID, nil
}

// ListNotifications lists notifications for the current user.
//
// @hestia.register(
//   name="system:notifications:notification:list",
//   intent="read",
//   rule="authenticated",
//   description="List notifications for the current user",
// )
func (s *NotificationsService) ListNotifications(ctx context.Context, msg abstract.Message, input *model.NotificationListInput) ([]*document.Document, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}
	tenantID := runtimecontext.GetTenantID(ctx)

	docs, err := s.model.List(ctx, userID, tenantID, 50, 0)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// MarkRead marks a notification as read.
//
// @hestia.register(
//   name="system:notifications:notification:read",
//   intent="update",
//   rule="authenticated",
//   description="Mark a notification as read",
//   resource_id="notification_id",
// )
func (s *NotificationsService) MarkRead(ctx context.Context, msg abstract.Message, input *model.NotificationReadInput) (*model.MessageOutput, error) {
	if _, err := userIDFrom(ctx, msg); err != nil {
		return nil, err
	}
	if input.NotificationID == "" {
		return nil, fmt.Errorf("notification_id is required")
	}
	if err := s.model.MarkRead(ctx, input.NotificationID); err != nil {
		return nil, err
	}
	return document.New(&model.MessageOutput{Message: "ok"}), nil
}

// MarkAllRead marks all notifications as read for the current user.
//
// @hestia.register(
//   name="system:notifications:read:all",
//   intent="update",
//   rule="authenticated",
//   description="Mark all notifications as read",
// )
func (s *NotificationsService) MarkAllRead(ctx context.Context, msg abstract.Message, input *model.NotificationMarkAllReadInput) (*model.MessageOutput, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}
	tenantID := runtimecontext.GetTenantID(ctx)

	if err := s.model.MarkAllRead(ctx, userID, tenantID); err != nil {
		return nil, err
	}
	return document.New(&model.MessageOutput{Message: "ok"}), nil
}

// CountUnread counts unread notifications for the current user.
//
// @hestia.register(
//   name="system:notifications:unread:count",
//   intent="read",
//   rule="authenticated",
//   description="Count unread notifications",
// )
func (s *NotificationsService) CountUnread(ctx context.Context, msg abstract.Message, input *model.NotificationUnreadCountInput) (*model.UnreadCountDocument, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}
	tenantID := runtimecontext.GetTenantID(ctx)

	count, err := s.model.CountUnread(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	return document.New(&model.UnreadCountDocument{Count: int64(count)}), nil
}