# API Keys API

## key

### List API keys

**`GET`** `/system/apikeys/key`

List API keys

- **Handler:** `system:apikeys:key:list`

#### Response

```json
{
  "version": "1.0.0",
  "name": "api_key_list",
  "description": "List of API keys",
  "fields": {
    "documents": {
      "name": "documents",
      "description": "Array of API keys",
      "type": "array",
      "schema": {
        "id": "api_key_document"
      }
    }
  },
  "schemas": {
    "api_key_document": {
      "name": "APIKeyDocument",
      "fields": {
        "_id": {
          "name": "_id",
          "description": "Unique key identifier",
          "type": "string"
        },
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Display name",
          "type": "string"
        },
        "prefix": {
          "name": "prefix",
          "description": "Key prefix with hint",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Key status",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Create an API key

**`POST`** `/system/apikeys/key`

Create an API key

- **Handler:** `system:apikeys:key:create`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "api_key_create_input",
  "description": "API key creation request",
  "fields": {
    "payload": {
      "name": "payload",
      "description": "Key creation details",
      "type": "object",
      "schema": {
        "id": "api_key_create_payload"
      }
    }
  },
  "schemas": {
    "api_key_create_payload": {
      "name": "APIKeyCreatePayload",
      "fields": {
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "ip": {
          "name": "ip",
          "description": "IP restriction rules",
          "type": "string"
        },
        "limits": {
          "name": "limits",
          "description": "Rate limits configuration",
          "type": "record"
        },
        "name": {
          "name": "name",
          "description": "Display name for the key",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Key status",
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
  "name": "api_key",
  "description": "An API key with metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "API key document",
      "type": "object",
      "schema": {
        "id": "api_key_document"
      }
    }
  },
  "schemas": {
    "api_key_document": {
      "name": "APIKeyDocument",
      "fields": {
        "_id": {
          "name": "_id",
          "description": "Unique key identifier",
          "type": "string"
        },
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "key": {
          "name": "key",
          "description": "Raw API key value (shown once on create/rotate)",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Display name",
          "type": "string"
        },
        "prefix": {
          "name": "prefix",
          "description": "Key prefix with hint",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Key status",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Get API key

**`GET`** `/system/apikeys/key/{key_id}`

Get API key

- **Handler:** `system:apikeys:key:get`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "api_key_get_input",
  "description": "API key identifier from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Request arguments",
      "type": "object",
      "schema": {
        "id": "api_key_get_arguments"
      }
    }
  },
  "schemas": {
    "api_key_get_arguments": {
      "name": "APIKeyGetArguments",
      "fields": {
        "key_id": {
          "name": "key_id",
          "description": "Unique API key identifier",
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
  "name": "api_key",
  "description": "An API key with metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "API key document",
      "type": "object",
      "schema": {
        "id": "api_key_document"
      }
    }
  },
  "schemas": {
    "api_key_document": {
      "name": "APIKeyDocument",
      "fields": {
        "_id": {
          "name": "_id",
          "description": "Unique key identifier",
          "type": "string"
        },
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "key": {
          "name": "key",
          "description": "Raw API key value (shown once on create/rotate)",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Display name",
          "type": "string"
        },
        "prefix": {
          "name": "prefix",
          "description": "Key prefix with hint",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Key status",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Update API key

**`PATCH`** `/system/apikeys/key/{key_id}`

Update API key

- **Handler:** `system:apikeys:key:update`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "api_key_update_input",
  "description": "API key update request",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Request arguments",
      "type": "object",
      "schema": {
        "id": "api_key_update_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Key fields to update",
      "type": "object",
      "schema": {
        "id": "api_key_update_payload"
      }
    }
  },
  "schemas": {
    "api_key_update_arguments": {
      "name": "APIKeyUpdateArguments",
      "fields": {
        "key_id": {
          "name": "key_id",
          "description": "Unique API key identifier",
          "type": "string"
        }
      }
    },
    "api_key_update_payload": {
      "name": "APIKeyUpdatePayload",
      "fields": {
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "expiry": {
          "name": "expiry",
          "description": "Expiration timestamp (RFC3339)",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Display name",
          "type": "string"
        },
        "scopes": {
          "name": "scopes",
          "description": "Permission scopes",
          "type": "array",
          "schema": {
            "id": "",
            "type": "string"
          }
        },
        "status": {
          "name": "status",
          "description": "Key status (active/revoked)",
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
  "name": "api_key",
  "description": "An API key with metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "API key document",
      "type": "object",
      "schema": {
        "id": "api_key_document"
      }
    }
  },
  "schemas": {
    "api_key_document": {
      "name": "APIKeyDocument",
      "fields": {
        "_id": {
          "name": "_id",
          "description": "Unique key identifier",
          "type": "string"
        },
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "key": {
          "name": "key",
          "description": "Raw API key value (shown once on create/rotate)",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Display name",
          "type": "string"
        },
        "prefix": {
          "name": "prefix",
          "description": "Key prefix with hint",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Key status",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Delete API key

**`DELETE`** `/system/apikeys/key/{key_id}`

Delete API key

- **Handler:** `system:apikeys:key:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "api_key_delete_input",
  "description": "API key identifier from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Request arguments",
      "type": "object",
      "schema": {
        "id": "api_key_delete_arguments"
      }
    }
  },
  "schemas": {
    "api_key_delete_arguments": {
      "name": "APIKeyDeleteArguments",
      "fields": {
        "key_id": {
          "name": "key_id",
          "description": "Unique API key identifier",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Rotate API key

**`POST`** `/system/apikeys/key/{key_id}`

Rotate API key

- **Handler:** `system:apikeys:key:rotate`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "api_key_rotate_input",
  "description": "API key identifier for rotation",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Request arguments",
      "type": "object",
      "schema": {
        "id": "api_key_rotate_arguments"
      }
    }
  },
  "schemas": {
    "api_key_rotate_arguments": {
      "name": "APIKeyRotateArguments",
      "fields": {
        "key_id": {
          "name": "key_id",
          "description": "Unique API key identifier",
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
  "name": "api_key",
  "description": "An API key with metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "API key document",
      "type": "object",
      "schema": {
        "id": "api_key_document"
      }
    }
  },
  "schemas": {
    "api_key_document": {
      "name": "APIKeyDocument",
      "fields": {
        "_id": {
          "name": "_id",
          "description": "Unique key identifier",
          "type": "string"
        },
        "environment": {
          "name": "environment",
          "description": "Environment restriction",
          "type": "string"
        },
        "key": {
          "name": "key",
          "description": "Raw API key value (shown once on create/rotate)",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Display name",
          "type": "string"
        },
        "prefix": {
          "name": "prefix",
          "description": "Key prefix with hint",
          "type": "string"
        },
        "status": {
          "name": "status",
          "description": "Key status",
          "type": "string"
        }
      }
    }
  }
}
```

---
