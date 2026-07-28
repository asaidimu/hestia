package notifications

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	NotificationModel *NotificationModel
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{
			Name:        "system:notifications:notification:list",
			Handler:     NewListNotificationsHandler(deps.NotificationModel),
			Description: "List notifications for the current user",
			Enabled:     true,
			Intent:      abstract.Read,
			Output:      notificationsListOutputSchema(),
		},
		{
			Name:        "system:notifications:notification:read",
			Handler:     NewMarkReadHandler(deps.NotificationModel),
			Description: "Mark a notification as read",
			Enabled:     true,
			Intent:      abstract.Update,
			Input:       runtime.Input{Arguments: []abstract.ArgDef{{Name: "notification_id", Type: definition.FieldTypeString}}, ResourceIDField: "notification_id"},
			Output:      messageOutputSchema(),
		},
		{
			Name:        "system:notifications:read:all",
			Handler:     NewMarkAllReadHandler(deps.NotificationModel),
			Description: "Mark all notifications as read",
			Enabled:     true,
			Intent:      abstract.Update,
			Output:      messageOutputSchema(),
		},
		{
			Name:        "system:notifications:unread:count",
			Handler:     NewCountUnreadHandler(deps.NotificationModel),
			Description: "Count unread notifications",
			Enabled:     true,
			Intent:      abstract.Read,
			Output:      unreadCountOutputSchema(),
		},
	}
}
