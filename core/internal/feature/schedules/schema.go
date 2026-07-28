package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_schedulesListOutput = dispatch.MustFromJSON(schedulesListOutputJSON)
	_scheduleOutput      = dispatch.MustFromJSON(scheduleOutputJSON)
	_messageOutput       = dispatch.MustFromJSON(messageOutputJSON)
	_scheduleCreateInput  = dispatch.MustFromJSON(scheduleCreateInputJSON)
)

func schedulesListOutputSchema() *definition.Schema { return _schedulesListOutput }
func scheduleOutputSchema() *definition.Schema      { return _scheduleOutput }
func messageOutputSchema() *definition.Schema       { return _messageOutput }
func scheduleCreateInputSchema() *definition.Schema { return _scheduleCreateInput }

var schedulesListOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedules_list_output",
	"description": "List of scheduled messages",
	"fields": {
		"documents": {
			"name": "documents",
			"type": "array",
			"schema": { "id": "schedule_document" }
		}
	},
	"schemas": {
		"schedule_document": {
			"name": "ScheduleDocument",
			"fields": {
				"_id": { "name": "_id", "type": "string" },
				"user_id": { "name": "user_id", "type": "string" },
				"type": { "name": "type", "type": "string" },
				"channel": { "name": "channel", "type": "string" },
				"data": { "name": "data", "type": "any" },
				"send_at": { "name": "send_at", "type": "integer" },
				"sent": { "name": "sent", "type": "boolean" },
				"sent_at": { "name": "sent_at", "type": "integer" },
				"error_": { "name": "error_", "type": "string" },
				"tenant_id": { "name": "tenant_id", "type": "string" },
				"created_at": { "name": "created_at", "type": "integer" }
			}
		}
	}
}`)

var scheduleOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedule_output",
	"description": "A single scheduled message",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "schedule_document" }
		}
	},
	"schemas": {
		"schedule_document": {
			"name": "ScheduleDocument",
			"fields": {
				"_id": { "name": "_id", "type": "string" },
				"user_id": { "name": "user_id", "type": "string" },
				"type": { "name": "type", "type": "string" },
				"channel": { "name": "channel", "type": "string" },
				"data": { "name": "data", "type": "any" },
				"send_at": { "name": "send_at", "type": "integer" },
				"sent": { "name": "sent", "type": "boolean" },
				"sent_at": { "name": "sent_at", "type": "integer" },
				"error_": { "name": "error_", "type": "string" },
				"tenant_id": { "name": "tenant_id", "type": "string" },
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

var scheduleCreateInputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedule_create_input",
	"description": "Input for creating a scheduled message",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "Schedule creation payload",
			"type": "object",
			"schema": { "id": "schedule_create_payload" }
		}
	},
	"schemas": {
		"schedule_create_payload": {
			"name": "ScheduleCreatePayload",
			"fields": {
				"user_id": { "name": "user_id", "type": "string" },
				"type": { "name": "type", "type": "string" },
				"channel": { "name": "channel", "type": "string" },
				"data": { "name": "data", "type": "record" },
				"send_at": { "name": "send_at", "type": "integer" }
			}
		}
	}
}`)
