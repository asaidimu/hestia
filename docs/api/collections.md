# Collections API

## collection

### List collections

**`GET`** `/system/collections/collection`

List collections

- **Handler:** `system:collections:collection:list`

#### Response

```json
{
  "version": "1.0.0",
  "name": "collection_list_output",
  "description": "List of collections",
  "fields": {
    "page": {
      "name": "page",
      "description": "Paginated collection list",
      "type": "object",
      "schema": {
        "id": "collection_page"
      }
    }
  },
  "schemas": {
    "collection_page": {
      "name": "CollectionPage",
      "fields": {
        "collections": {
          "name": "collections",
          "description": "Array of collection names",
          "type": "array",
          "schema": {
            "id": "",
            "type": "string"
          }
        }
      }
    }
  }
}
```

---

### Get collection

**`GET`** `/system/collections/collection/{name}`

Get collection

- **Handler:** `system:collections:collection:get`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_get_input",
  "description": "Collection name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name argument",
      "type": "object",
      "schema": {
        "id": "collection_get_args"
      }
    }
  },
  "schemas": {
    "collection_get_args": {
      "name": "CollectionGetArgs",
      "fields": {
        "name": {
          "name": "name",
          "description": "Collection name",
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
  "name": "collection_output",
  "description": "Collection schema definition",
  "fields": {
    "document": {
      "name": "document",
      "description": "Collection document",
      "type": "object",
      "schema": {
        "id": "collection_meta"
      }
    }
  },
  "schemas": {
    "collection_meta": {
      "name": "CollectionMeta",
      "fields": {
        "name": {
          "name": "name",
          "description": "Collection name",
          "type": "string"
        },
        "schema": {
          "name": "schema",
          "description": "Collection schema JSON",
          "type": "record"
        }
      }
    }
  }
}
```

---

### Create collection via API

**`POST`** `/system/collections/collection`

Create collection via API

- **Handler:** `system:collections:collection:create`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_create_input",
  "description": "Anansi schema definition for the new collection",
  "fields": {
    "payload": {
      "name": "payload",
      "description": "Anansi schema JSON definition",
      "type": "record"
    }
  }
}
```

#### Response

```json
{
  "version": "1.0.0",
  "name": "collection_output",
  "description": "Collection schema definition",
  "fields": {
    "document": {
      "name": "document",
      "description": "Collection document",
      "type": "object",
      "schema": {
        "id": "collection_meta"
      }
    }
  },
  "schemas": {
    "collection_meta": {
      "name": "CollectionMeta",
      "fields": {
        "name": {
          "name": "name",
          "description": "Collection name",
          "type": "string"
        },
        "schema": {
          "name": "schema",
          "description": "Collection schema JSON",
          "type": "record"
        }
      }
    }
  }
}
```

---

### Delete collection via API

**`DELETE`** `/system/collections/collection/{name}`

Delete collection via API

- **Handler:** `system:collections:collection:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_delete_input",
  "description": "Collection name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name argument",
      "type": "object",
      "schema": {
        "id": "collection_delete_args"
      }
    }
  },
  "schemas": {
    "collection_delete_args": {
      "name": "CollectionDeleteArgs",
      "fields": {
        "name": {
          "name": "name",
          "description": "Collection name",
          "type": "string"
        }
      }
    }
  }
}
```

---

## document

### Query collection documents

**`POST`** `/system/collections/document/{name}/query`

Query collection documents

- **Handler:** `system:collections:document:query`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_doc_query_input",
  "description": "QDSL query with collection name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name argument",
      "type": "object",
      "schema": {
        "id": "collection_doc_query_args"
      }
    },
    "payload": {
      "name": "payload",
      "description": "QDSL query object",
      "type": "record"
    }
  },
  "schemas": {
    "collection_doc_query_args": {
      "name": "CollectionDocQueryArgs",
      "fields": {
        "name": {
          "name": "name",
          "description": "Collection name",
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
  "name": "collection_query_output",
  "description": "Paginated collection documents",
  "fields": {
    "page": {
      "name": "page",
      "description": "Paginated document results",
      "type": "record"
    }
  }
}
```

---

### Create document in collection

**`POST`** `/system/collections/document/{name}`

Create document in collection

- **Handler:** `system:collections:document:create`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_doc_create_input",
  "description": "Create a document in a collection",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name argument",
      "type": "object",
      "schema": {
        "id": "collection_doc_create_args"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Document data",
      "type": "record"
    }
  },
  "schemas": {
    "collection_doc_create_args": {
      "name": "CollectionDocCreateArgs",
      "fields": {
        "name": {
          "name": "name",
          "description": "Collection name",
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
  "name": "collection_document_output",
  "description": "A document within a collection",
  "fields": {
    "document": {
      "name": "document",
      "description": "Collection document",
      "type": "object",
      "schema": {
        "id": "collection_document"
      }
    }
  },
  "schemas": {
    "collection_document": {
      "name": "CollectionDocument",
      "fields": {
        "data": {
          "name": "data",
          "description": "Document fields",
          "type": "record"
        },
        "id": {
          "name": "id",
          "description": "Document ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Get document from collection

**`GET`** `/system/collections/document/{name}/{doc_id}`

Get document from collection

- **Handler:** `system:collections:document:get`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_doc_get_input",
  "description": "Get a document by collection and document ID",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name and document ID arguments",
      "type": "object",
      "schema": {
        "id": "collection_doc_get_args"
      }
    }
  },
  "schemas": {
    "collection_doc_get_args": {
      "name": "CollectionDocGetArgs",
      "fields": {
        "doc_id": {
          "name": "doc_id",
          "description": "Document ID",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Collection name",
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
  "name": "collection_document_output",
  "description": "A document within a collection",
  "fields": {
    "document": {
      "name": "document",
      "description": "Collection document",
      "type": "object",
      "schema": {
        "id": "collection_document"
      }
    }
  },
  "schemas": {
    "collection_document": {
      "name": "CollectionDocument",
      "fields": {
        "data": {
          "name": "data",
          "description": "Document fields",
          "type": "record"
        },
        "id": {
          "name": "id",
          "description": "Document ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Update document in collection

**`PATCH`** `/system/collections/document/{name}/{doc_id}`

Update document in collection

- **Handler:** `system:collections:document:update`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_doc_update_input",
  "description": "Update a document by collection and document ID",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name and document ID arguments",
      "type": "object",
      "schema": {
        "id": "collection_doc_update_args"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Updated document fields",
      "type": "record"
    }
  },
  "schemas": {
    "collection_doc_update_args": {
      "name": "CollectionDocUpdateArgs",
      "fields": {
        "doc_id": {
          "name": "doc_id",
          "description": "Document ID",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Collection name",
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
  "name": "collection_document_output",
  "description": "A document within a collection",
  "fields": {
    "document": {
      "name": "document",
      "description": "Collection document",
      "type": "object",
      "schema": {
        "id": "collection_document"
      }
    }
  },
  "schemas": {
    "collection_document": {
      "name": "CollectionDocument",
      "fields": {
        "data": {
          "name": "data",
          "description": "Document fields",
          "type": "record"
        },
        "id": {
          "name": "id",
          "description": "Document ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Delete document from collection

**`DELETE`** `/system/collections/document/{name}/{doc_id}`

Delete document from collection

- **Handler:** `system:collections:document:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "collection_doc_delete_input",
  "description": "Delete a document by collection and document ID",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Collection name and document ID arguments",
      "type": "object",
      "schema": {
        "id": "collection_doc_delete_args"
      }
    }
  },
  "schemas": {
    "collection_doc_delete_args": {
      "name": "CollectionDocDeleteArgs",
      "fields": {
        "doc_id": {
          "name": "doc_id",
          "description": "Document ID",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Collection name",
          "type": "string"
        }
      }
    }
  }
}
```

---

## _user

### Query users collection

**`GET`** `/system/collections/_user`

Query users collection

- **Handler:** `system:collections:_user:read`
- **Internal:** Yes

---

## _api_key

### Query API keys collection

**`GET`** `/system/collections/_api_key`

Query API keys collection

- **Handler:** `system:collections:_api_key:read`
- **Internal:** Yes

---

## _policy_operation

### Query policy operations

**`GET`** `/system/collections/_policy_operation`

Query policy operations

- **Handler:** `system:collections:_policy_operation:read`
- **Internal:** Yes

---

## _policy_rule

### Query policy rules

**`GET`** `/system/collections/_policy_rule`

Query policy rules

- **Handler:** `system:collections:_policy_rule:read`
- **Internal:** Yes

---

## _access_log

### Query access logs

**`GET`** `/system/collections/_access_log`

Query access logs

- **Handler:** `system:collections:_access_log:read`
- **Internal:** Yes

---
