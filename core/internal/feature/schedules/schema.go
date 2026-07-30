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
	_scheduleUpdateInput  = dispatch.MustFromJSON(scheduleUpdateInputJSON)
)

func schedulesListOutputSchema() *definition.Schema { return _schedulesListOutput }
func scheduleOutputSchema() *definition.Schema      { return _scheduleOutput }
func messageOutputSchema() *definition.Schema       { return _messageOutput }
func scheduleCreateInputSchema() *definition.Schema { return _scheduleCreateInput }
func scheduleUpdateInputSchema() *definition.Schema { return _scheduleUpdateInput }

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

var scheduleCreateInputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedule_create_input",
	"description": "Input for creating a schedule",
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
				"message": { "name": "message", "required": true, "type": "string" },
				"input": { "name": "input", "type": "record" },
				"cron": { "name": "cron", "required": true, "type": "string" },
				"disabled": { "name": "disabled", "type": "boolean" }
			}
		}
	}
}`)

var scheduleUpdateInputJSON = []byte(`{
	"version": "1.0.0",
	"name": "schedule_update_input",
	"description": "Input for updating a schedule",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "Schedule update payload",
			"type": "object",
			"schema": { "id": "schedule_update_payload" }
		}
	},
	"schemas": {
		"schedule_update_payload": {
			"name": "ScheduleUpdatePayload",
			"fields": {
				"message": { "name": "message", "type": "string" },
				"input": { "name": "input", "type": "record" },
				"cron": { "name": "cron", "type": "string" },
				"disabled": { "name": "disabled", "type": "boolean" }
			}
		}
	},
	"arguments": [
		{ "name": "id", "type": "string", "required": true }
	]
}`)
