# Audit API

## log

### Stream audit log entries in real-time

**`GET`** `/system/audit/log/stream`

Stream audit log entries in real-time

- **Handler:** `system:audit:log:stream`

#### Request Body

```json
{
  "name": "LogStreamInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7d7c-8d14-6aebc7f827e7": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-7926-bea4-6396fb5dad58"
      },
      "type": "object"
    }
  },
  "name": "LogStreamOutput",
  "schemas": {
    "019fba9e-d801-7926-bea4-6396fb5dad58": {
      "fields": {
        "019fba9e-d801-7a32-8e28-0287e60f5088": {
          "name": "event_id",
          "type": "string"
        },
        "019fba9e-d802-78e4-94bc-968c0c8e2a6d": {
          "name": "occurred_at",
          "type": "string"
        },
        "019fba9e-d803-72c8-b133-0f2563996b76": {
          "name": "recorded_at",
          "type": "string"
        },
        "019fba9e-d804-77fe-b5d0-0b74248a57f8": {
          "name": "trace_id",
          "type": "string"
        },
        "019fba9e-d805-79f1-91ad-2240e3a7e200": {
          "name": "request_id",
          "type": "string"
        },
        "019fba9e-d806-71f2-974f-152587b2f454": {
          "name": "actor_id",
          "type": "string"
        },
        "019fba9e-d807-7b09-aa3c-abc5028dd7a9": {
          "name": "actor_type",
          "type": "string"
        },
        "019fba9e-d808-7b03-a955-eb348b4206fb": {
          "name": "on_behalf_of_id",
          "type": "string"
        },
        "019fba9e-d809-7101-a4cb-e996a91ecef1": {
          "name": "auth_method",
          "type": "string"
        },
        "019fba9e-d80a-7051-a61a-b54a7e492df1": {
          "name": "session_id",
          "type": "string"
        },
        "019fba9e-d80b-730f-b6ae-ad13657be9f2": {
          "name": "operation",
          "type": "string"
        },
        "019fba9e-d80c-7942-bdf0-72061a1724a0": {
          "name": "resource_type",
          "type": "string"
        },
        "019fba9e-d80d-7f22-b938-6c23557b5d37": {
          "name": "resource_id",
          "type": "string"
        },
        "019fba9e-d80e-77e2-9505-83c07a5a6918": {
          "name": "event_name",
          "type": "string"
        },
        "019fba9e-d80f-7f4d-ab99-4460528e2f07": {
          "name": "status",
          "type": "string"
        },
        "019fba9e-d810-7a08-ad03-712fde610b39": {
          "name": "severity",
          "type": "string"
        },
        "019fba9e-d811-71e4-a1ed-00632e52ed8e": {
          "name": "error_code",
          "type": "string"
        },
        "019fba9e-d812-7e11-ab8b-126b5201b152": {
          "name": "error_message",
          "type": "string"
        },
        "019fba9e-d813-73b6-827c-0bf443ea5ff0": {
          "name": "latency_ms",
          "type": "integer"
        },
        "019fba9e-d814-7562-94e5-34bdf7b1e695": {
          "name": "source_ip",
          "type": "string"
        },
        "019fba9e-d815-761a-92c0-3ddbb8e113d7": {
          "name": "user_agent",
          "type": "string"
        },
        "019fba9e-d816-7603-9d38-42b19e4f56a2": {
          "name": "service_name",
          "type": "string"
        },
        "019fba9e-d817-74a7-a68c-f642b82a321c": {
          "name": "region",
          "type": "string"
        }
      },
      "name": "AuditLogEntryView"
    }
  },
  "version": "1.0.0"
}
```

---

### Export audit logs

**`PATCH`** `/system/audit/log/export`

Export audit logs

- **Handler:** `system:audit:log:export`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-74f4-9a89-189b6737c868": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "LogQueryInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7473-9f0f-2e0c0b777a6e": {
      "name": "page",
      "schema": {
        "id": "019fba9e-d801-7b58-a579-3e9d92310b6f"
      },
      "type": "object"
    }
  },
  "name": "LogQueryOutput",
  "schemas": {
    "019fba9e-d801-7b58-a579-3e9d92310b6f": {
      "fields": {
        "019fba9e-d801-762d-82b2-3210955fc0a4": {
          "name": "documents",
          "schema": {
            "id": "019fba9e-d802-7926-bea4-6396fb5dad58"
          },
          "type": "array"
        },
        "019fba9e-d802-767d-8ad2-18d95ea137d0": {
          "name": "pagination",
          "schema": {
            "id": "019fba9e-d803-7c34-ab22-160ebbdf538d"
          },
          "type": "object"
        }
      },
      "name": "AuditLogPageView"
    },
    "019fba9e-d802-7926-bea4-6396fb5dad58": {
      "fields": {
        "019fba9e-d801-7b2c-a889-92fdfd1b13c5": {
          "name": "event_id",
          "type": "string"
        },
        "019fba9e-d802-7183-9a81-6bae917c72b4": {
          "name": "occurred_at",
          "type": "string"
        },
        "019fba9e-d803-7908-87d8-04bd9689be3c": {
          "name": "recorded_at",
          "type": "string"
        },
        "019fba9e-d804-775c-9c33-6434cb297360": {
          "name": "trace_id",
          "type": "string"
        },
        "019fba9e-d805-7404-9b4b-f080a2895823": {
          "name": "request_id",
          "type": "string"
        },
        "019fba9e-d806-77cb-bbf6-dd8b50150235": {
          "name": "actor_id",
          "type": "string"
        },
        "019fba9e-d807-7fcb-a817-8c82927365de": {
          "name": "actor_type",
          "type": "string"
        },
        "019fba9e-d808-7583-bb18-48f56119f79d": {
          "name": "on_behalf_of_id",
          "type": "string"
        },
        "019fba9e-d809-7026-ab34-a527409ed458": {
          "name": "auth_method",
          "type": "string"
        },
        "019fba9e-d80a-7585-8153-ee20fc4feb36": {
          "name": "session_id",
          "type": "string"
        },
        "019fba9e-d80b-7fbd-b21b-5c9b6a557aac": {
          "name": "operation",
          "type": "string"
        },
        "019fba9e-d80c-7fda-a727-d0652c87e206": {
          "name": "resource_type",
          "type": "string"
        },
        "019fba9e-d80d-7797-90f3-ace7c55eb30f": {
          "name": "resource_id",
          "type": "string"
        },
        "019fba9e-d80e-7bb5-bf40-79d96746eaac": {
          "name": "event_name",
          "type": "string"
        },
        "019fba9e-d80f-7ff4-b281-582b2b71f318": {
          "name": "status",
          "type": "string"
        },
        "019fba9e-d810-7a5d-a2b4-cf5f91547377": {
          "name": "severity",
          "type": "string"
        },
        "019fba9e-d811-714e-97c4-56a8071a3c97": {
          "name": "error_code",
          "type": "string"
        },
        "019fba9e-d812-7bff-b628-223a88f82e62": {
          "name": "error_message",
          "type": "string"
        },
        "019fba9e-d813-72de-a91b-976f561fd567": {
          "name": "latency_ms",
          "type": "integer"
        },
        "019fba9e-d814-7b03-8b0e-be4b7d31320d": {
          "name": "source_ip",
          "type": "string"
        },
        "019fba9e-d815-76db-8c8b-5a14e7efaff5": {
          "name": "user_agent",
          "type": "string"
        },
        "019fba9e-d816-7f5c-ba08-75ddcf97c95f": {
          "name": "service_name",
          "type": "string"
        },
        "019fba9e-d817-7ffc-b976-e3161efe4d1a": {
          "name": "region",
          "type": "string"
        }
      },
      "name": "AuditLogEntryView"
    },
    "019fba9e-d803-7c34-ab22-160ebbdf538d": {
      "fields": {
        "019fba9e-d801-7518-a1a2-0cc7dcad8220": {
          "name": "total",
          "type": "integer"
        },
        "019fba9e-d802-7d7a-8caa-23ff242c9d66": {
          "name": "cursor",
          "type": "string"
        },
        "019fba9e-d803-7d0b-9bd7-55359f65d1fd": {
          "name": "limit",
          "type": "integer"
        }
      },
      "name": "AuditPaginationView"
    }
  },
  "version": "1.0.0"
}
```

---
