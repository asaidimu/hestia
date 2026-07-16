package audit

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/app/core/schema"
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
				"actor_id": { "name": "actor_id", "description": "Filter by actor ID", "type": "string" },
				"actor_type": { "name": "actor_type", "description": "Filter by actor type", "type": "string" },
				"operation": { "name": "operation", "description": "Filter by operation", "type": "string" },
				"status": { "name": "status", "description": "Filter by status", "type": "string" },
				"resource_type": { "name": "resource_type", "description": "Filter by resource type", "type": "string" },
				"trace_id": { "name": "trace_id", "description": "Filter by trace ID", "type": "string" },
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
				"event_id": { "name": "event_id", "description": "Unique event identifier", "type": "string" },
				"occurred_at": { "name": "occurred_at", "description": "When the event occurred (RFC3339)", "type": "string" },
				"recorded_at": { "name": "recorded_at", "description": "When the record was written (RFC3339)", "type": "string" },
				"trace_id": { "name": "trace_id", "description": "Distributed trace ID", "type": "string" },
				"request_id": { "name": "request_id", "description": "Originating request ID", "type": "string" },
				"actor_id": { "name": "actor_id", "description": "Who performed the action", "type": "string" },
				"actor_type": { "name": "actor_type", "description": "Type of actor", "type": "string" },
				"on_behalf_of_id": { "name": "on_behalf_of_id", "description": "Delegated/impersonated identity", "type": "string" },
				"auth_method": { "name": "auth_method", "description": "Authentication method used", "type": "string" },
				"session_id": { "name": "session_id", "description": "Session identifier", "type": "string" },
				"operation": { "name": "operation", "description": "Action category", "type": "string" },
				"resource_type": { "name": "resource_type", "description": "Kind of resource acted upon", "type": "string" },
				"resource_id": { "name": "resource_id", "description": "Resource instance identifier", "type": "string" },
				"event_name": { "name": "event_name", "description": "Fine-grained event taxonomy", "type": "string" },
				"status": { "name": "status", "description": "Outcome of the action", "type": "string" },
				"severity": { "name": "severity", "description": "Event severity level", "type": "string" },
				"error_code": { "name": "error_code", "description": "Machine-readable error code", "type": "string" },
				"error_message": { "name": "error_message", "description": "Human-readable error detail", "type": "string" },
				"latency_ms": { "name": "latency_ms", "description": "Duration in milliseconds", "type": "integer" },
				"source_ip": { "name": "source_ip", "description": "Originating IP address", "type": "string" },
				"user_agent": { "name": "user_agent", "description": "User agent string", "type": "string" },
				"service_name": { "name": "service_name", "description": "Emitting service", "type": "string" },
				"region": { "name": "region", "description": "Deployment region", "type": "string" }
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
				"actor_id": { "name": "actor_id", "description": "Filter by actor ID", "type": "string" },
				"actor_type": { "name": "actor_type", "description": "Filter by actor type", "type": "string" },
				"operation": { "name": "operation", "description": "Filter by operation", "type": "string" },
				"status": { "name": "status", "description": "Filter by status", "type": "string" }
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
				"event_id": { "name": "event_id", "description": "Unique event identifier", "type": "string" },
				"occurred_at": { "name": "occurred_at", "description": "When the event occurred (RFC3339)", "type": "string" },
				"recorded_at": { "name": "recorded_at", "description": "When the record was written (RFC3339)", "type": "string" },
				"trace_id": { "name": "trace_id", "description": "Distributed trace ID", "type": "string" },
				"request_id": { "name": "request_id", "description": "Originating request ID", "type": "string" },
				"actor_id": { "name": "actor_id", "description": "Who performed the action", "type": "string" },
				"actor_type": { "name": "actor_type", "description": "Type of actor", "type": "string" },
				"on_behalf_of_id": { "name": "on_behalf_of_id", "description": "Delegated/impersonated identity", "type": "string" },
				"auth_method": { "name": "auth_method", "description": "Authentication method used", "type": "string" },
				"session_id": { "name": "session_id", "description": "Session identifier", "type": "string" },
				"operation": { "name": "operation", "description": "Action category", "type": "string" },
				"resource_type": { "name": "resource_type", "description": "Kind of resource acted upon", "type": "string" },
				"resource_id": { "name": "resource_id", "description": "Resource instance identifier", "type": "string" },
				"event_name": { "name": "event_name", "description": "Fine-grained event taxonomy", "type": "string" },
				"status": { "name": "status", "description": "Outcome of the action", "type": "string" },
				"severity": { "name": "severity", "description": "Event severity level", "type": "string" },
				"error_code": { "name": "error_code", "description": "Machine-readable error code", "type": "string" },
				"error_message": { "name": "error_message", "description": "Human-readable error detail", "type": "string" },
				"latency_ms": { "name": "latency_ms", "description": "Duration in milliseconds", "type": "integer" },
				"source_ip": { "name": "source_ip", "description": "Originating IP address", "type": "string" },
				"user_agent": { "name": "user_agent", "description": "User agent string", "type": "string" },
				"service_name": { "name": "service_name", "description": "Emitting service", "type": "string" },
				"region": { "name": "region", "description": "Deployment region", "type": "string" }
			}
		}
	}
}`)
