# Audit API

## log

### Query audit logs

**`POST`** `/system/audit/log/query`

Query audit logs

- **Handler:** `system:audit:log:query`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "log_query_input",
  "description": "Query audit logs with filters and pagination",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Query arguments",
      "type": "object",
      "schema": {
        "id": "log_query_arguments"
      }
    },
    "modifiers": {
      "name": "modifiers",
      "description": "Query modifiers (pagination)",
      "type": "object",
      "schema": {
        "id": "log_query_modifiers"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Query filter payload",
      "type": "object",
      "schema": {
        "id": "log_query_payload"
      }
    }
  },
  "schemas": {
    "log_query_arguments": {
      "name": "LogQueryArguments"
    },
    "log_query_modifiers": {
      "name": "LogQueryModifiers",
      "fields": {
        "cursor": {
          "name": "cursor",
          "description": "Pagination cursor",
          "type": "string"
        },
        "limit": {
          "name": "limit",
          "description": "Maximum number of results",
          "type": "integer"
        }
      }
    },
    "log_query_payload": {
      "name": "LogQueryPayload",
      "fields": {
        "actor_id": {
          "name": "actor_id",
          "description": "Filter by actor ID",
          "type": "string"
        },
        "actor_type": {
          "name": "actor_type",
          "description": "Filter by actor type",
          "type": "string"
        },
        "end": {
          "name": "end",
          "description": "End time filter (RFC3339)",
          "type": "string"
        },
        "operation": {
          "name": "operation",
          "description": "Filter by operation",
          "type": "string"
        },
        "resource_type": {
          "name": "resource_type",
          "description": "Filter by resource type",
          "type": "string"
        },
        "start": {
          "name": "start",
          "description": "Start time filter (RFC3339)",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Filter by status",
          "type": "string"
        },
        "trace_id": {
          "name": "trace_id",
          "description": "Filter by trace ID",
          "type": "string"
        }
      }
    }
  }
}
```

#### Response

```json
{
  "version": "1.0.0",
  "name": "log_query_output",
  "description": "Paginated audit log entries",
  "fields": {
    "page": {
      "name": "page",
      "description": "Paginated list of log entries",
      "type": "object",
      "schema": {
        "id": "log_page"
      }
    }
  },
  "schemas": {
    "log_entry": {
      "name": "LogEntry",
      "fields": {
        "actor_id": {
          "name": "actor_id",
          "description": "Who performed the action",
          "type": "string"
        },
        "actor_type": {
          "name": "actor_type",
          "description": "Type of actor",
          "type": "string"
        },
        "auth_method": {
          "name": "auth_method",
          "description": "Authentication method used",
          "type": "string"
        },
        "error_code": {
          "name": "error_code",
          "description": "Machine-readable error code",
          "type": "string"
        },
        "error_message": {
          "name": "error_message",
          "description": "Human-readable error detail",
          "type": "string"
        },
        "event_id": {
          "name": "event_id",
          "description": "Unique event identifier",
          "type": "string"
        },
        "event_name": {
          "name": "event_name",
          "description": "Fine-grained event taxonomy",
          "type": "string"
        },
        "latency_ms": {
          "name": "latency_ms",
          "description": "Duration in milliseconds",
          "type": "integer"
        },
        "occurred_at": {
          "name": "occurred_at",
          "description": "When the event occurred (RFC3339)",
          "type": "string"
        },
        "on_behalf_of_id": {
          "name": "on_behalf_of_id",
          "description": "Delegated/impersonated identity",
          "type": "string"
        },
        "operation": {
          "name": "operation",
          "description": "Action category",
          "type": "string"
        },
        "recorded_at": {
          "name": "recorded_at",
          "description": "When the record was written (RFC3339)",
          "type": "string"
        },
        "region": {
          "name": "region",
          "description": "Deployment region",
          "type": "string"
        },
        "request_id": {
          "name": "request_id",
          "description": "Originating request ID",
          "type": "string"
        },
        "resource_id": {
          "name": "resource_id",
          "description": "Resource instance identifier",
          "type": "string"
        },
        "resource_type": {
          "name": "resource_type",
          "description": "Kind of resource acted upon",
          "type": "string"
        },
        "service_name": {
          "name": "service_name",
          "description": "Emitting service",
          "type": "string"
        },
        "session_id": {
          "name": "session_id",
          "description": "Session identifier",
          "type": "string"
        },
        "severity": {
          "name": "severity",
          "description": "Event severity level",
          "type": "string"
        },
        "source_ip": {
          "name": "source_ip",
          "description": "Originating IP address",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Outcome of the action",
          "type": "string"
        },
        "trace_id": {
          "name": "trace_id",
          "description": "Distributed trace ID",
          "type": "string"
        },
        "user_agent": {
          "name": "user_agent",
          "description": "User agent string",
          "type": "string"
        }
      }
    },
    "log_page": {
      "name": "LogPage",
      "fields": {
        "documents": {
          "name": "documents",
          "description": "Array of log entries",
          "type": "array",
          "schema": {
            "id": "log_entry"
          }
        },
        "pagination": {
          "name": "pagination",
          "description": "Pagination metadata",
          "type": "object",
          "schema": {
            "id": "pagination_meta"
          }
        }
      }
    },
    "pagination_meta": {
      "name": "PaginationMeta",
      "fields": {
        "cursor": {
          "name": "cursor",
          "description": "Cursor for next page",
          "type": "string"
        },
        "limit": {
          "name": "limit",
          "description": "Number of results requested",
          "type": "integer"
        },
        "total": {
          "name": "total",
          "description": "Total number of matching documents",
          "type": "integer"
        }
      }
    }
  }
}
```

---

### Export audit logs

**`PATCH`** `/system/audit/log`

Export audit logs

- **Handler:** `system:audit:log:export`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "log_query_input",
  "description": "Query audit logs with filters and pagination",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Query arguments",
      "type": "object",
      "schema": {
        "id": "log_query_arguments"
      }
    },
    "modifiers": {
      "name": "modifiers",
      "description": "Query modifiers (pagination)",
      "type": "object",
      "schema": {
        "id": "log_query_modifiers"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Query filter payload",
      "type": "object",
      "schema": {
        "id": "log_query_payload"
      }
    }
  },
  "schemas": {
    "log_query_arguments": {
      "name": "LogQueryArguments"
    },
    "log_query_modifiers": {
      "name": "LogQueryModifiers",
      "fields": {
        "cursor": {
          "name": "cursor",
          "description": "Pagination cursor",
          "type": "string"
        },
        "limit": {
          "name": "limit",
          "description": "Maximum number of results",
          "type": "integer"
        }
      }
    },
    "log_query_payload": {
      "name": "LogQueryPayload",
      "fields": {
        "actor_id": {
          "name": "actor_id",
          "description": "Filter by actor ID",
          "type": "string"
        },
        "actor_type": {
          "name": "actor_type",
          "description": "Filter by actor type",
          "type": "string"
        },
        "end": {
          "name": "end",
          "description": "End time filter (RFC3339)",
          "type": "string"
        },
        "operation": {
          "name": "operation",
          "description": "Filter by operation",
          "type": "string"
        },
        "resource_type": {
          "name": "resource_type",
          "description": "Filter by resource type",
          "type": "string"
        },
        "start": {
          "name": "start",
          "description": "Start time filter (RFC3339)",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Filter by status",
          "type": "string"
        },
        "trace_id": {
          "name": "trace_id",
          "description": "Filter by trace ID",
          "type": "string"
        }
      }
    }
  }
}
```

#### Response

```json
{
  "version": "1.0.0",
  "name": "log_query_output",
  "description": "Paginated audit log entries",
  "fields": {
    "page": {
      "name": "page",
      "description": "Paginated list of log entries",
      "type": "object",
      "schema": {
        "id": "log_page"
      }
    }
  },
  "schemas": {
    "log_entry": {
      "name": "LogEntry",
      "fields": {
        "actor_id": {
          "name": "actor_id",
          "description": "Who performed the action",
          "type": "string"
        },
        "actor_type": {
          "name": "actor_type",
          "description": "Type of actor",
          "type": "string"
        },
        "auth_method": {
          "name": "auth_method",
          "description": "Authentication method used",
          "type": "string"
        },
        "error_code": {
          "name": "error_code",
          "description": "Machine-readable error code",
          "type": "string"
        },
        "error_message": {
          "name": "error_message",
          "description": "Human-readable error detail",
          "type": "string"
        },
        "event_id": {
          "name": "event_id",
          "description": "Unique event identifier",
          "type": "string"
        },
        "event_name": {
          "name": "event_name",
          "description": "Fine-grained event taxonomy",
          "type": "string"
        },
        "latency_ms": {
          "name": "latency_ms",
          "description": "Duration in milliseconds",
          "type": "integer"
        },
        "occurred_at": {
          "name": "occurred_at",
          "description": "When the event occurred (RFC3339)",
          "type": "string"
        },
        "on_behalf_of_id": {
          "name": "on_behalf_of_id",
          "description": "Delegated/impersonated identity",
          "type": "string"
        },
        "operation": {
          "name": "operation",
          "description": "Action category",
          "type": "string"
        },
        "recorded_at": {
          "name": "recorded_at",
          "description": "When the record was written (RFC3339)",
          "type": "string"
        },
        "region": {
          "name": "region",
          "description": "Deployment region",
          "type": "string"
        },
        "request_id": {
          "name": "request_id",
          "description": "Originating request ID",
          "type": "string"
        },
        "resource_id": {
          "name": "resource_id",
          "description": "Resource instance identifier",
          "type": "string"
        },
        "resource_type": {
          "name": "resource_type",
          "description": "Kind of resource acted upon",
          "type": "string"
        },
        "service_name": {
          "name": "service_name",
          "description": "Emitting service",
          "type": "string"
        },
        "session_id": {
          "name": "session_id",
          "description": "Session identifier",
          "type": "string"
        },
        "severity": {
          "name": "severity",
          "description": "Event severity level",
          "type": "string"
        },
        "source_ip": {
          "name": "source_ip",
          "description": "Originating IP address",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Outcome of the action",
          "type": "string"
        },
        "trace_id": {
          "name": "trace_id",
          "description": "Distributed trace ID",
          "type": "string"
        },
        "user_agent": {
          "name": "user_agent",
          "description": "User agent string",
          "type": "string"
        }
      }
    },
    "log_page": {
      "name": "LogPage",
      "fields": {
        "documents": {
          "name": "documents",
          "description": "Array of log entries",
          "type": "array",
          "schema": {
            "id": "log_entry"
          }
        },
        "pagination": {
          "name": "pagination",
          "description": "Pagination metadata",
          "type": "object",
          "schema": {
            "id": "pagination_meta"
          }
        }
      }
    },
    "pagination_meta": {
      "name": "PaginationMeta",
      "fields": {
        "cursor": {
          "name": "cursor",
          "description": "Cursor for next page",
          "type": "string"
        },
        "limit": {
          "name": "limit",
          "description": "Number of results requested",
          "type": "integer"
        },
        "total": {
          "name": "total",
          "description": "Total number of matching documents",
          "type": "integer"
        }
      }
    }
  }
}
```

---

### Stream audit log entries in real-time

**`GET`** `/system/audit/log/stream`

Stream audit log entries in real-time

- **Handler:** `system:audit:log:stream`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "log_stream_input",
  "description": "Start a real-time log stream with optional filters",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Stream arguments",
      "type": "object",
      "schema": {
        "id": "log_stream_arguments"
      }
    },
    "modifiers": {
      "name": "modifiers",
      "description": "Stream modifiers",
      "type": "object",
      "schema": {
        "id": "log_stream_modifiers"
      }
    }
  },
  "schemas": {
    "log_stream_arguments": {
      "name": "LogStreamArguments",
      "fields": {
        "actor_id": {
          "name": "actor_id",
          "description": "Filter by actor ID",
          "type": "string"
        },
        "actor_type": {
          "name": "actor_type",
          "description": "Filter by actor type",
          "type": "string"
        },
        "operation": {
          "name": "operation",
          "description": "Filter by operation",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Filter by status",
          "type": "string"
        }
      }
    },
    "log_stream_modifiers": {
      "name": "LogStreamModifiers"
    }
  }
}
```

#### Response

```json
{
  "version": "1.0.0",
  "name": "log_stream_output",
  "description": "A single real-time log entry",
  "fields": {
    "document": {
      "name": "document",
      "description": "Log entry document",
      "type": "object",
      "schema": {
        "id": "log_entry"
      }
    }
  },
  "schemas": {
    "log_entry": {
      "name": "LogEntry",
      "fields": {
        "actor_id": {
          "name": "actor_id",
          "description": "Who performed the action",
          "type": "string"
        },
        "actor_type": {
          "name": "actor_type",
          "description": "Type of actor",
          "type": "string"
        },
        "auth_method": {
          "name": "auth_method",
          "description": "Authentication method used",
          "type": "string"
        },
        "error_code": {
          "name": "error_code",
          "description": "Machine-readable error code",
          "type": "string"
        },
        "error_message": {
          "name": "error_message",
          "description": "Human-readable error detail",
          "type": "string"
        },
        "event_id": {
          "name": "event_id",
          "description": "Unique event identifier",
          "type": "string"
        },
        "event_name": {
          "name": "event_name",
          "description": "Fine-grained event taxonomy",
          "type": "string"
        },
        "latency_ms": {
          "name": "latency_ms",
          "description": "Duration in milliseconds",
          "type": "integer"
        },
        "occurred_at": {
          "name": "occurred_at",
          "description": "When the event occurred (RFC3339)",
          "type": "string"
        },
        "on_behalf_of_id": {
          "name": "on_behalf_of_id",
          "description": "Delegated/impersonated identity",
          "type": "string"
        },
        "operation": {
          "name": "operation",
          "description": "Action category",
          "type": "string"
        },
        "recorded_at": {
          "name": "recorded_at",
          "description": "When the record was written (RFC3339)",
          "type": "string"
        },
        "region": {
          "name": "region",
          "description": "Deployment region",
          "type": "string"
        },
        "request_id": {
          "name": "request_id",
          "description": "Originating request ID",
          "type": "string"
        },
        "resource_id": {
          "name": "resource_id",
          "description": "Resource instance identifier",
          "type": "string"
        },
        "resource_type": {
          "name": "resource_type",
          "description": "Kind of resource acted upon",
          "type": "string"
        },
        "service_name": {
          "name": "service_name",
          "description": "Emitting service",
          "type": "string"
        },
        "session_id": {
          "name": "session_id",
          "description": "Session identifier",
          "type": "string"
        },
        "severity": {
          "name": "severity",
          "description": "Event severity level",
          "type": "string"
        },
        "source_ip": {
          "name": "source_ip",
          "description": "Originating IP address",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Outcome of the action",
          "type": "string"
        },
        "trace_id": {
          "name": "trace_id",
          "description": "Distributed trace ID",
          "type": "string"
        },
        "user_agent": {
          "name": "user_agent",
          "description": "User agent string",
          "type": "string"
        }
      }
    }
  }
}
```

---
