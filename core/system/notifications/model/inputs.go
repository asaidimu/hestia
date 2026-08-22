package model

type NotificationListInput struct{}

// NotificationCreateInput wraps the schema-declared NotificationCreate
// projection for use as a dispatcher input DTO.
type NotificationCreateInput struct {
	NotificationCreate
}

type NotificationReadInput struct {
	NotificationID string `input:"arguments.notification_id"`
}

type NotificationMarkAllReadInput struct{}

type NotificationUnreadCountInput struct{}