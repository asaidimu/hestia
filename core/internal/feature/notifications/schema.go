package notifications

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_notificationsListOutput = dispatch.MustFromJSON(notificationsListOutputJSON)
	_notificationOutput      = dispatch.MustFromJSON(notificationOutputJSON)
	_messageOutput           = dispatch.MustFromJSON(messageOutputJSON)
	_unreadCountOutput       = dispatch.MustFromJSON(unreadCountOutputJSON)
)

func notificationsListOutputSchema() *definition.Schema { return _notificationsListOutput }
func notificationOutputSchema() *definition.Schema     { return _notificationOutput }
func messageOutputSchema() *definition.Schema          { return _messageOutput }
func unreadCountOutputSchema() *definition.Schema      { return _unreadCountOutput }

var notificationsListOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "notifications_list_output",
	"description": "List of notifications",
	"fields": {
		"documents": {
			"name": "documents",
			"type": "array",
			"schema": { "id": "notification_document" }
		}
	},
	"schemas": {
		"notification_document": {
			"name": "NotificationDocument",
			"fields": {
				"_id": { "name": "_id", "type": "string" },
				"user_id": { "name": "user_id", "type": "string" },
				"type": { "name": "type", "type": "string" },
				"subject": { "name": "subject", "type": "string" },
				"body": { "name": "body", "type": "string" },
				"data": { "name": "data", "type": "any" },
				"read": { "name": "read", "type": "boolean" },
				"created_at": { "name": "created_at", "type": "integer" }
			}
		}
	}
}`)

var notificationOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "notification_output",
	"description": "A single notification",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "notification_document" }
		}
	},
	"schemas": {
		"notification_document": {
			"name": "NotificationDocument",
			"fields": {
				"_id": { "name": "_id", "type": "string" },
				"user_id": { "name": "user_id", "type": "string" },
				"type": { "name": "type", "type": "string" },
				"subject": { "name": "subject", "type": "string" },
				"body": { "name": "body", "type": "string" },
				"data": { "name": "data", "type": "any" },
				"read": { "name": "read", "type": "boolean" },
				"created_at": { "name": "created_at", "type": "integer" }
			}
		}
	}
}`)

var messageOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "message_output",
	"description": "Generic message response",
	"fields": {
		"message": {
			"name": "message",
			"type": "string"
		}
	}
}`)

var unreadCountOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "unread_count_output",
	"description": "Unread notification count",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "unread_count_document" }
		}
	},
	"schemas": {
		"unread_count_document": {
			"name": "UnreadCountDocument",
			"fields": {
				"count": { "name": "count", "type": "integer" }
			}
		}
	}
}`)
