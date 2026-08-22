package notifications

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

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

type NotificationsListOutput struct {
	Documents []NotificationDocumentView `anansi:"documents"`
}

type MessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}

type UnreadCountDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Count                  int64 `anansi:"count"`
}

type UnreadCountOutput struct {
	Document UnreadCountDocument `anansi:"document"`
}
