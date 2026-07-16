# Blobs API

## namespace

### List blob namespaces

**`POST`** `/system/blobs/namespace/query`

List blob namespaces

- **Handler:** `system:blobs:namespace:list`

#### Response

```json
{
  "version": "1.0.0",
  "name": "blob_namespace_list",
  "description": "List of blob namespaces",
  "fields": {
    "document": {
      "name": "document",
      "description": "Namespace list document",
      "type": "object",
      "schema": {
        "id": "ns_list_document"
      }
    }
  },
  "schemas": {
    "namespace": {
      "name": "Namespace",
      "fields": {
        "display_name": {
          "name": "display_name",
          "description": "Display name",
          "type": "string"
        },
        "id": {
          "name": "id",
          "description": "Namespace ID",
          "type": "string"
        },
        "public": {
          "name": "public",
          "description": "Whether public access is enabled",
          "type": "boolean"
        }
      }
    },
    "ns_list_document": {
      "name": "NamespaceListDocument",
      "fields": {
        "namespaces": {
          "name": "namespaces",
          "description": "Array of blob namespaces",
          "type": "array",
          "schema": {
            "id": "namespace"
          }
        }
      }
    }
  }
}
```

---

### Create a blob namespace

**`POST`** `/system/blobs/namespace`

Create a blob namespace

- **Handler:** `system:blobs:namespace:create`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_ns_create_input",
  "description": "Namespace creation request",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "blob_ns_create_payload"
      }
    }
  },
  "schemas": {
    "blob_ns_create_payload": {
      "name": "BlobNSCreatePayload",
      "fields": {
        "display_name": {
          "name": "display_name",
          "description": "Display name for the namespace",
          "type": "string"
        },
        "public": {
          "name": "public",
          "description": "Whether the namespace allows public (unauthenticated) access",
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
  "name": "blob_namespace",
  "description": "A blob namespace",
  "fields": {
    "document": {
      "name": "document",
      "description": "Namespace document",
      "type": "object",
      "schema": {
        "id": "namespace"
      }
    }
  },
  "schemas": {
    "namespace": {
      "name": "Namespace",
      "fields": {
        "display_name": {
          "name": "display_name",
          "description": "Display name",
          "type": "string"
        },
        "id": {
          "name": "id",
          "description": "Namespace ID",
          "type": "string"
        },
        "public": {
          "name": "public",
          "description": "Whether public access is enabled",
          "type": "boolean"
        }
      }
    }
  }
}
```

---

### Delete a blob namespace

**`DELETE`** `/system/blobs/namespace/{ns}`

Delete a blob namespace

- **Handler:** `system:blobs:namespace:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_ns_input",
  "description": "Namespace identifier from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_ns_input_args"
      }
    }
  },
  "schemas": {
    "blob_ns_input_args": {
      "name": "BlobNsInputArgs",
      "fields": {
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

## blob

### List blobs in a namespace

**`POST`** `/system/blobs/blob/{ns}/query`

List blobs in a namespace

- **Handler:** `system:blobs:blob:list`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_list_input",
  "description": "List blobs with optional filters",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_list_input_args"
      }
    },
    "payload": {
      "name": "payload",
      "type": "record",
      "schema": {
        "id": "blob_list_payload"
      }
    }
  },
  "schemas": {
    "blob_list_input_args": {
      "name": "BlobListInputArgs",
      "fields": {
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
          "type": "string"
        }
      }
    },
    "blob_list_payload": {
      "name": "BlobListPayload",
      "fields": {
        "limit": {
          "name": "limit",
          "description": "Max results",
          "type": "integer"
        },
        "prefix": {
          "name": "prefix",
          "description": "Key prefix filter",
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
  "name": "blob_list",
  "description": "List of blobs in a namespace",
  "fields": {
    "document": {
      "name": "document",
      "description": "Blob list document",
      "type": "object",
      "schema": {
        "id": "blob_list_document"
      }
    }
  },
  "schemas": {
    "blob_list_document": {
      "name": "BlobListDocument",
      "fields": {
        "blobs": {
          "name": "blobs",
          "description": "Array of blob metadata",
          "type": "array",
          "schema": {
            "id": "blob_meta"
          }
        }
      }
    },
    "blob_meta": {
      "name": "BlobMeta",
      "fields": {
        "content_type": {
          "name": "content_type",
          "description": "MIME content type",
          "type": "string"
        },
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "custom": {
          "name": "custom",
          "description": "Arbitrary metadata",
          "type": "object"
        },
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "namespace_id": {
          "name": "namespace_id",
          "description": "Namespace ID",
          "type": "string"
        },
        "size": {
          "name": "size",
          "description": "Size in bytes",
          "type": "integer"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last modification timestamp",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Get blob metadata

**`POST`** `/system/blobs/blob/{ns}/{key}/query`

Get blob metadata

- **Handler:** `system:blobs:blob:head`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_key_input",
  "description": "Blob key and namespace from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_key_input_args"
      }
    }
  },
  "schemas": {
    "blob_key_input_args": {
      "name": "BlobKeyInputArgs",
      "fields": {
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
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
  "name": "blob_meta",
  "description": "Blob metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "Blob metadata document",
      "type": "object",
      "schema": {
        "id": "blob_meta"
      }
    }
  },
  "schemas": {
    "blob_meta": {
      "name": "BlobMeta",
      "fields": {
        "content_type": {
          "name": "content_type",
          "description": "MIME content type",
          "type": "string"
        },
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "custom": {
          "name": "custom",
          "description": "Arbitrary metadata",
          "type": "object"
        },
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "namespace_id": {
          "name": "namespace_id",
          "description": "Namespace ID",
          "type": "string"
        },
        "size": {
          "name": "size",
          "description": "Size in bytes",
          "type": "integer"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last modification timestamp",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Upload a blob

**`POST`** `/system/blobs/blob/{ns}/{key}`

Upload a blob

- **Handler:** `system:blobs:blob:upload`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_key_input",
  "description": "Blob key and namespace from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_key_input_args"
      }
    }
  },
  "schemas": {
    "blob_key_input_args": {
      "name": "BlobKeyInputArgs",
      "fields": {
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
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
  "name": "blob_meta",
  "description": "Blob metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "Blob metadata document",
      "type": "object",
      "schema": {
        "id": "blob_meta"
      }
    }
  },
  "schemas": {
    "blob_meta": {
      "name": "BlobMeta",
      "fields": {
        "content_type": {
          "name": "content_type",
          "description": "MIME content type",
          "type": "string"
        },
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "custom": {
          "name": "custom",
          "description": "Arbitrary metadata",
          "type": "object"
        },
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "namespace_id": {
          "name": "namespace_id",
          "description": "Namespace ID",
          "type": "string"
        },
        "size": {
          "name": "size",
          "description": "Size in bytes",
          "type": "integer"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last modification timestamp",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Download a blob

**`GET`** `/system/blobs/blob/{ns}/{key}`

Download a blob

- **Handler:** `system:blobs:blob:download`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_key_input",
  "description": "Blob key and namespace from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_key_input_args"
      }
    }
  },
  "schemas": {
    "blob_key_input_args": {
      "name": "BlobKeyInputArgs",
      "fields": {
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Delete a blob

**`DELETE`** `/system/blobs/blob/{ns}/{key}`

Delete a blob

- **Handler:** `system:blobs:blob:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_key_input",
  "description": "Blob key and namespace from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_key_input_args"
      }
    }
  },
  "schemas": {
    "blob_key_input_args": {
      "name": "BlobKeyInputArgs",
      "fields": {
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Update blob metadata

**`PATCH`** `/system/blobs/blob/{ns}/{key}`

Update blob metadata

- **Handler:** `system:blobs:blob:update`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "blob_update_input",
  "description": "Update blob metadata",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "blob_update_input_args"
      }
    },
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "blob_update_payload"
      }
    }
  },
  "schemas": {
    "blob_update_input_args": {
      "name": "BlobUpdateInputArgs",
      "fields": {
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "ns": {
          "name": "ns",
          "description": "Namespace ID",
          "type": "string"
        }
      }
    },
    "blob_update_payload": {
      "name": "BlobUpdatePayload",
      "fields": {
        "content_type": {
          "name": "content_type",
          "description": "New MIME content type",
          "type": "string"
        },
        "custom": {
          "name": "custom",
          "description": "Arbitrary metadata to set",
          "type": "object"
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
  "name": "blob_meta",
  "description": "Blob metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "Blob metadata document",
      "type": "object",
      "schema": {
        "id": "blob_meta"
      }
    }
  },
  "schemas": {
    "blob_meta": {
      "name": "BlobMeta",
      "fields": {
        "content_type": {
          "name": "content_type",
          "description": "MIME content type",
          "type": "string"
        },
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "custom": {
          "name": "custom",
          "description": "Arbitrary metadata",
          "type": "object"
        },
        "key": {
          "name": "key",
          "description": "Blob key",
          "type": "string"
        },
        "namespace_id": {
          "name": "namespace_id",
          "description": "Namespace ID",
          "type": "string"
        },
        "size": {
          "name": "size",
          "description": "Size in bytes",
          "type": "integer"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last modification timestamp",
          "type": "string"
        }
      }
    }
  }
}
```

---
