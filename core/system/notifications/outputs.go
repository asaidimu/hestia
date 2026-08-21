// @note #cruft-20260821-024 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema functions in notifications/outputs.go
// @see #8uuufn
//
// The schema functions (notificationsListOutputSchema, messageOutputSchema,
// unreadCountOutputSchema) are dead code. The generated registrations use
// dispatch.SchemaFromType directly.
//
// Resolution: remove the schema functions. The output types themselves are
// still used by the service methods and registrations.
package notifications

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// NotificationDocumentView is the declared shape of a notification document.
// List responses carry the raw persisted documents.
type NotificationDocumentView struct {
	ID        string `anansi:"_id"`
	UserID    string `anansi:"user_id"`
	Type      string `anansi:"type"`
	Subject   string `anansi:"subject"`
	Body      string `anansi:"body"`
	Data      any    `anansi:"data"`
	Read      bool   `anansi:"read"`
	CreatedAt int64  `anansi:"created_at"`
}

// NotificationsListOutput declares the notifications list schema.
type NotificationsListOutput struct {
	Documents []NotificationDocumentView `anansi:"documents"`
}

func notificationsListOutputSchema() *definition.Schema { return dispatch.SchemaFromType[NotificationsListOutput]() }

// MessageOutput declares a simple status message response.
type MessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}

func messageOutputSchema() *definition.Schema { return dispatch.SchemaFromType[MessageOutput]() }

// UnreadCountDocument is the wire shape of the unread notification count.
type UnreadCountDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Count                  int64 `anansi:"count"`
}

// UnreadCountOutput declares the unread count schema.
type UnreadCountOutput struct {
	Document UnreadCountDocument `anansi:"document"`
}

func unreadCountOutputSchema() *definition.Schema { return dispatch.SchemaFromType[UnreadCountOutput]() }