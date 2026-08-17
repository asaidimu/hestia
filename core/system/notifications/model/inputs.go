package model

type NotificationListInput struct{}

type NotificationReadInput struct {
	NotificationID string `input:"arguments.notification_id"`
}

type NotificationMarkAllReadInput struct{}

type NotificationUnreadCountInput struct{}