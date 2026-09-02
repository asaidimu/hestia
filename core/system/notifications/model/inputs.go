package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type NotificationListInput struct{}

type NotificationReadInput struct {
	NotificationID string `input:"arguments.notification_id"`
}

type NotificationMarkAllReadInput struct{}

type NotificationUnreadCountInput struct{}

// NotificationStreamInput is the input for the notification stream endpoint.
// No fields — the authenticated user's identity is extracted from context.
type NotificationStreamInput struct{}

func NotificationStreamInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[NotificationStreamInput]("input", true)
}
