package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_schedulesListOutput = dispatch.MustFromJSON(schedulesListOutputJSON)
	_scheduleOutput      = dispatch.MustFromJSON(scheduleOutputJSON)
	_messageOutput       = dispatch.MustFromJSON(messageOutputJSON)
)

func schedulesListOutputSchema() *definition.Schema { return _schedulesListOutput }
func scheduleOutputSchema() *definition.Schema      { return _scheduleOutput }
func messageOutputSchema() *definition.Schema       { return _messageOutput }

var schedulesListOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedules_list_output",
	"description": "List of schedules",
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
				"message": { "name": "message", "type": "string" },
				"input": { "name": "input", "type": "record" },
				"cron": { "name": "cron", "type": "string" },
				"disabled": { "name": "disabled", "type": "boolean" },
				"tenant_id": { "name": "tenant_id", "type": "string" },
				"created_at": { "name": "created_at", "type": "integer" }
			}
		}
	}
}`)

var scheduleOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedule_output",
	"description": "A single schedule",
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
				"message": { "name": "message", "type": "string" },
				"input": { "name": "input", "type": "record" },
				"cron": { "name": "cron", "type": "string" },
				"disabled": { "name": "disabled", "type": "boolean" },
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
