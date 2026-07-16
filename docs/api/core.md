# Core API

## health

### Health check

**`GET`** `/system/core/health`

Health check

- **Handler:** `system:core:health:check`
- **Bootstrap-safe:** Yes

#### Response

```json
{
  "version": "1.0.0",
  "name": "health",
  "description": "System health and bootstrap status",
  "fields": {
    "document": {
      "name": "document",
      "description": "Health status document",
      "type": "object",
      "schema": {
        "id": "health_document"
      }
    }
  },
  "schemas": {
    "health_document": {
      "name": "Health Status",
      "fields": {
        "bootstrapped": {
          "name": "bootstrapped",
          "description": "Whether the system has been bootstrapped",
          "type": "boolean"
        },
        "ok": {
          "name": "ok",
          "description": "Whether the system is healthy",
          "type": "boolean"
        }
      }
    }
  }
}
```

---

## capability

### List all registered handlers

**`GET`** `/system/core/capability`

List all registered handlers

- **Handler:** `system:core:capability:list`

#### Response

```json
{
  "version": "1.0.0",
  "name": "capabilities",
  "description": "List of registered command and query capabilities",
  "fields": {
    "document": {
      "name": "document",
      "description": "Capabilities document",
      "type": "object",
      "schema": {
        "id": "capabilities_document"
      }
    }
  },
  "schemas": {
    "capabilities_document": {
      "name": "CapabilitiesDocument",
      "fields": {
        "capabilities": {
          "name": "capabilities",
          "description": "Array of handler capabilities",
          "type": "array",
          "schema": {
            "id": "capability_item"
          }
        }
      }
    },
    "capability_item": {
      "name": "CapabilityItem",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "enabled": {
          "name": "enabled",
          "description": "Whether the handler is enabled",
          "type": "boolean"
        },
        "intent_type": {
          "name": "intent_type",
          "description": "Command or query",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Handler name",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Enable or disable a handler

**`PATCH`** `/system/core/capability`

Enable or disable a handler

- **Handler:** `system:core:capability:set`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "capability_name_input",
  "description": "Capability identifier from the path with enabled payload",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Capability name argument",
      "type": "object",
      "schema": {
        "id": "capability_name_input_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Enabled flag payload",
      "type": "object",
      "schema": {
        "id": "capability_name_input_payload"
      }
    }
  },
  "schemas": {
    "capability_name_input_arguments": {
      "name": "CapabilityNameInputArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Name of the capability to modify",
          "type": "string"
        }
      }
    },
    "capability_name_input_payload": {
      "name": "CapabilityNameInputPayload",
      "fields": {
        "enabled": {
          "name": "enabled",
          "description": "Whether the capability is enabled",
          "type": "boolean"
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
  "name": "message",
  "description": "A simple status message response",
  "fields": {
    "message": {
      "name": "message",
      "description": "Human-readable status message",
      "type": "string"
    }
  }
}
```

---

## audit

### Record an audit log entry

**`POST`** `/system/core/audit`

Record an audit log entry

- **Handler:** `system:core:audit:log`
- **Internal:** Yes

---

## docs

### Endpoint documentation

**`GET`** `/system/core/docs`

Endpoint documentation

- **Handler:** `system:core:docs:list`
- **Bootstrap-safe:** Yes

#### Response

```json
{
  "version": "1.0.0",
  "name": "documentation",
  "description": "List of all registered API endpoints with metadata",
  "fields": {
    "documents": {
      "name": "documents",
      "description": "Array of endpoint metadata objects",
      "type": "array",
      "schema": {
        "id": "endpoint_doc"
      }
    }
  },
  "schemas": {
    "endpoint_doc": {
      "name": "EndpointDoc",
      "fields": {
        "bootstrap_safe": {
          "name": "bootstrap_safe",
          "description": "Whether safe during bootstrap",
          "type": "boolean"
        },
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "enabled": {
          "name": "enabled",
          "description": "Whether the handler is enabled",
          "type": "boolean"
        },
        "http": {
          "name": "http",
          "description": "HTTP method and route",
          "type": "object",
          "schema": {
            "id": "http_mapping"
          }
        },
        "input": {
          "name": "input",
          "description": "Input schema definition",
          "type": "record"
        },
        "intent": {
          "name": "intent",
          "description": "Intent type",
          "type": "string"
        },
        "internal": {
          "name": "internal",
          "description": "Whether the handler is internal-only",
          "type": "boolean"
        },
        "name": {
          "name": "name",
          "description": "Handler name",
          "type": "string"
        },
        "output": {
          "name": "output",
          "description": "Output schema definition",
          "type": "record"
        }
      }
    },
    "http_mapping": {
      "name": "HTTPMapping",
      "fields": {
        "method": {
          "name": "method",
          "description": "HTTP method",
          "type": "string"
        },
        "pattern": {
          "name": "pattern",
          "description": "Combined method and route",
          "type": "string"
        },
        "route": {
          "name": "route",
          "description": "HTTP route path",
          "type": "string"
        }
      }
    }
  }
}
```

---

## bootstrap

### Mark system as bootstrapped

**`POST`** `/system/core/bootstrap`

Mark system as bootstrapped

- **Handler:** `system:core:bootstrap:mark`
- **Internal:** Yes

#### Response

```json
{
  "version": "1.0.0",
  "name": "message",
  "description": "A simple status message response",
  "fields": {
    "message": {
      "name": "message",
      "description": "Human-readable status message",
      "type": "string"
    }
  }
}
```

---

## reset

### Reset system to initial state

**`GET`** `/system/core/reset`

Reset system to initial state

- **Handler:** `system:core:reset`

#### Response

```json
{
  "version": "1.0.0",
  "name": "message",
  "description": "A simple status message response",
  "fields": {
    "message": {
      "name": "message",
      "description": "Human-readable status message",
      "type": "string"
    }
  }
}
```

---
