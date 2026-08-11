package audit

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_logQueryOutput  = dispatch.MustFromJSON(logQueryOutputJSON)
	_logStreamOutput = dispatch.MustFromJSON(logStreamOutputJSON)
)

func LogQueryOutputSchema() *definition.Schema  { return _logQueryOutput }
func LogStreamOutputSchema() *definition.Schema { return _logStreamOutput }

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
