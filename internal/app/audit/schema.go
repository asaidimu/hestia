package audit

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/internal/core/schema"
)

var (
	_logQueryInput    = schema.MustFromJSON(logQueryInputJSON)
	_logQueryOutput   = schema.MustFromJSON(logQueryOutputJSON)
	_logStreamInput   = schema.MustFromJSON(logStreamInputJSON)
	_logStreamOutput  = schema.MustFromJSON(logStreamOutputJSON)
)

func logQueryInputSchema() *definition.Schema     { return _logQueryInput }
func logQueryOutputSchema() *definition.Schema    { return _logQueryOutput }
func logEntryOutputSchema() *definition.Schema    { return _logQueryOutput }
func logStreamInputSchema() *definition.Schema    { return _logStreamInput }
func logStreamOutputSchema() *definition.Schema   { return _logStreamOutput }

var logQueryInputJSON = []byte(`{
	"name": "log_query_input",
	"description": "Query audit logs with filters and pagination",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Query arguments",
			"type": "object",
			"schema": { "id": "log_query_arguments" }
		},
		"modifiers": {
			"name": "modifiers",
			"description": "Query modifiers (pagination)",
			"type": "object",
			"schema": { "id": "log_query_modifiers" }
		},
		"payload": {
			"name": "payload",
			"description": "Query filter payload",
			"type": "object",
			"schema": { "id": "log_query_payload" }
		}
	},
	"schemas": {
		"log_query_arguments": {
			"name": "LogQueryArguments",
			"fields": {}
		},
		"log_query_modifiers": {
			"name": "LogQueryModifiers",
			"fields": {
				"limit": { "name": "limit", "description": "Maximum number of results", "type": "integer" },
				"cursor": { "name": "cursor", "description": "Pagination cursor", "type": "string" }
			}
		},
		"log_query_payload": {
			"name": "LogQueryPayload",
			"fields": {
				"handler": { "name": "handler", "description": "Filter by handler name", "type": "string" },
				"user": { "name": "user", "description": "Filter by username", "type": "string" },
				"intent": { "name": "intent", "description": "Filter by intent type", "type": "string" },
				"start": { "name": "start", "description": "Start time filter (RFC3339)", "type": "string" },
				"end": { "name": "end", "description": "End time filter (RFC3339)", "type": "string" }
			}
		}
	}
}`)

var logQueryOutputJSON = []byte(`{
	"name": "log_query_output",
	"description": "Paginated audit log entries",
	"version": "1.0.0",
	"fields": {
		"page": {
			"name": "page",
			"description": "Paginated list of log entries",
			"type": "object",
			"schema": { "id": "log_page" }
		}
	},
	"schemas": {
		"log_page": {
			"name": "LogPage",
			"fields": {
				"documents": {
					"name": "documents",
					"description": "Array of log entries",
					"type": "array",
					"schema": { "id": "log_entry" }
				},
				"pagination": {
					"name": "pagination",
					"description": "Pagination metadata",
					"type": "object",
					"schema": { "id": "pagination_meta" }
				}
			}
		},
		"pagination_meta": {
			"name": "PaginationMeta",
			"fields": {
				"total": { "name": "total", "description": "Total number of matching documents", "type": "integer" },
				"cursor": { "name": "cursor", "description": "Cursor for next page", "type": "string" },
				"limit": { "name": "limit", "description": "Number of results requested", "type": "integer" }
			}
		},
		"log_entry": {
			"name": "LogEntry",
			"fields": {
				"id": { "name": "id", "description": "Log entry ID", "type": "string" },
				"handler": { "name": "handler", "description": "Handler that processed the request", "type": "string" },
				"user": { "name": "user", "description": "Username who made the request", "type": "string" },
				"intent": { "name": "intent", "description": "Intent type (command/query)", "type": "string" },
				"input": { "name": "input", "description": "Serialized request input", "type": "record" },
				"output": { "name": "output", "description": "Serialized response output", "type": "record" },
				"timestamp": { "name": "timestamp", "description": "When the request occurred (RFC3339)", "type": "string" },
				"duration": { "name": "duration", "description": "Request duration in milliseconds", "type": "integer" },
				"success": { "name": "success", "description": "Whether the request succeeded", "type": "boolean" }
			}
		}
	}
}`)

var logStreamInputJSON = []byte(`{
	"name": "log_stream_input",
	"description": "Start a real-time log stream with optional filters",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Stream arguments",
			"type": "object",
			"schema": { "id": "log_stream_arguments" }
		},
		"modifiers": {
			"name": "modifiers",
			"description": "Stream modifiers",
			"type": "object",
			"schema": { "id": "log_stream_modifiers" }
		}
	},
	"schemas": {
		"log_stream_arguments": {
			"name": "LogStreamArguments",
			"fields": {
				"handler": { "name": "handler", "description": "Filter by handler name", "type": "string" },
				"user": { "name": "user", "description": "Filter by username", "type": "string" },
				"intent": { "name": "intent", "description": "Filter by intent type", "type": "string" }
			}
		},
		"log_stream_modifiers": {
			"name": "LogStreamModifiers",
			"fields": {}
		}
	}
}`)

var logStreamOutputJSON = []byte(`{
	"name": "log_stream_output",
	"description": "A single real-time log entry",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Log entry document",
			"type": "object",
			"schema": { "id": "log_entry" }
		}
	},
	"schemas": {
		"log_entry": {
			"name": "LogEntry",
			"fields": {
				"id": { "name": "id", "description": "Log entry ID", "type": "string" },
				"handler": { "name": "handler", "description": "Handler that processed the request", "type": "string" },
				"user": { "name": "user", "description": "Username who made the request", "type": "string" },
				"intent": { "name": "intent", "description": "Intent type (command/query)", "type": "string" },
				"input": { "name": "input", "description": "Serialized request input", "type": "record" },
				"output": { "name": "output", "description": "Serialized response output", "type": "record" },
				"timestamp": { "name": "timestamp", "description": "When the request occurred (RFC3339)", "type": "string" },
				"duration": { "name": "duration", "description": "Request duration in milliseconds", "type": "integer" },
				"success": { "name": "success", "description": "Whether the request succeeded", "type": "boolean" }
			}
		}
	}
}`)
