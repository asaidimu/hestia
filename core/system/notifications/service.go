package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/system/notifications/model"

	"go.uber.org/zap"
)

// NotificationsService is the service for the in-app notifications domain. It
// wraps the generated SystemNotificationss model collection.
type NotificationsService struct {
	model   *model.SystemNotificationss
	persist persistence.Persistence
	log     *zap.Logger
}

func NewNotificationsService(rt abstract.Container) (*NotificationsService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)

	m, err := model.InitSystemNotificationssModel(persist, logger)
	if err != nil {
		return nil, err
	}
	return &NotificationsService{model: m, persist: persist, log: logger}, nil
}

func userIDFrom(ctx context.Context, msg abstract.Message) (string, error) {
	claims, ok := runtimecontext.ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return "", fmt.Errorf("unauthenticated")
	}
	return claims.UserID, nil
}

// CreateNotification creates an in-app notification for a user. Content is
// taken verbatim from the payload (no template rendering); channel fan-out via
// the notifier remains an internal concern. Administrator-gated because it
// targets arbitrary users.
//
// @hestia.register(
//
//	name="system:notifications:notification:create",
//	intent="create",
//	rule="administrator",
//	description="Create an in-app notification for a user",
//
// )
func (s *NotificationsService) CreateNotification(ctx context.Context, msg abstract.Message, input *model.NotificationCreateInput) (*model.SystemNotifications, error) {
	if input.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if input.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	ntype := "manual"
	if input.Type != nil && *input.Type != "" {
		ntype = *input.Type
	}

	now := time.Now().UnixMilli()
	read := false
	n := &model.SystemNotifications{
		UserID:    input.UserID,
		Type:      ntype,
		Subject:   input.Subject,
		Data:      input.Data,
		Actions:   input.Actions,
		Read:      &read,
		CreatedAt: &now,
	}
	if input.Body != nil && *input.Body != "" {
		n.Body = input.Body
	}
	if input.ExpiresAt != nil {
		n.ExpiresAt = input.ExpiresAt
	}
	if tenantID := runtimecontext.GetTenantID(ctx); tenantID != "" {
		n.TenantID = &tenantID
	}

	created, err := s.model.Create(ctx, n)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ListNotifications lists notifications for the current user.
//
// @hestia.register(
//
//	name="system:notifications:notification:list",
//	intent="read",
//	rule="authenticated",
//	description="List notifications for the current user",
//
// )
func (s *NotificationsService) ListNotifications(ctx context.Context, msg abstract.Message, input *model.NotificationListInput) ([]*document.Document, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}

	models, err := s.model.List(ctx, userID, 50, 0)
	if err != nil {
		return nil, err
	}
	docs := make([]*document.Document, len(models))
	for i, n := range models {
		docs[i], err = n.Document()
		if err != nil {
			return nil, err
		}
	}
	return docs, nil
}

// MarkRead marks a notification as read.
//
// @hestia.register(
//
//	name="system:notifications:notification:read",
//	intent="update",
//	rule="authenticated",
//	description="Mark a notification as read",
//	resource_id="notification_id",
//
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
//
//	name="system:notifications:read:all",
//	intent="update",
//	rule="authenticated",
//	description="Mark all notifications as read",
//
// )
func (s *NotificationsService) MarkAllRead(ctx context.Context, msg abstract.Message, input *model.NotificationMarkAllReadInput) (*model.MessageOutput, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := s.model.MarkAllRead(ctx, userID); err != nil {
		return nil, err
	}
	return document.New(&model.MessageOutput{Message: "ok"}), nil
}

// CountUnread counts unread notifications for the current user.
//
// @hestia.register(
//
//	name="system:notifications:unread:count",
//	intent="read",
//	rule="authenticated",
//	description="Count unread notifications",
//
// )
func (s *NotificationsService) CountUnread(ctx context.Context, msg abstract.Message, input *model.NotificationUnreadCountInput) (*model.UnreadCountDocument, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}

	count, err := s.model.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}
	return document.New(&model.UnreadCountDocument{Count: int64(count)}), nil
}

// Stream streams new notifications for the authenticated user in real-time.
//
// @hestia.register(
//
//	name="system:notifications:notification:stream",
//	intent="stream",
//	rule="authenticated",
//	description="Stream new notifications for the current user",
//	input="model.NotificationStreamInput",
//	output="model.NotificationStreamOutput",
//
// )
func (s *NotificationsService) Stream(ctx context.Context, msg abstract.Message, input *model.NotificationStreamInput) (*abstract.Result, error) {
	userID, err := userIDFrom(ctx, msg)
	if err != nil {
		return nil, err
	}

	docCh := make(chan *document.Document, 64)

	subID := s.model.Subscribe(ctx, persistence.SubscriptionOptions{
		Event: persistence.DocumentCreateSuccess,
		Callback: func(_ context.Context, event persistence.PersistenceEvent) error {
			outMap, ok := event.Output.(map[string]any)
			if !ok {
				return nil
			}
			dataRaw, ok := outMap["data"]
			if !ok {
				return nil
			}
			dataMap, ok := dataRaw.(map[string]any)
			if !ok || dataMap == nil {
				return nil
			}
			uid, _ := dataMap["user_id"].(string)
			if uid != userID {
				return nil
			}
			doc := document.NewRecordView(dataMap, context.Background())
			select {
			case docCh <- doc:
			default:
			}
			return nil
		},
	})

	go func() {
		select {
		case <-msg.InputChannel():
		case <-ctx.Done():
			close(docCh)
			s.model.Unsubscribe(ctx, subID)
		}

		// Phase 2: stream is live. Only close when the client goes away.
		<-ctx.Done()
		close(docCh)
		s.model.Unsubscribe(ctx, subID)
	}()

	return &abstract.Result{DocumentChannel: docCh}, nil
}
