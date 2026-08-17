# Collections API

## collection

### List collections

**`GET`** `/system/collections/collection/list`

List collections

- **Handler:** `system:collections:collection:list`

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7fff-8a19-26e7b7180449": {
      "name": "page",
      "schema": {
        "id": "019fba9e-d801-782b-8db0-6421a24c72db"
      },
      "type": "object"
    }
  },
  "name": "CollectionListOutput",
  "schemas": {
    "019fba9e-d801-782b-8db0-6421a24c72db": {
      "fields": {
        "019fba9e-d801-78ac-86a9-9e48721235d8": {
          "name": "collections",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "CollectionPage"
    }
  },
  "version": "1.0.0"
}
```

---

### Get collection

**`GET`** `/system/collections/collection/get/{name}`

Get collection

- **Handler:** `system:collections:collection:get`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7168-83bb-5c707b6a9aee": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7a9d-b877-2591ca1a483d"
      },
      "type": "object"
    }
  },
  "name": "CollectionGetInput",
  "schemas": {
    "019fba9e-d801-7a9d-b877-2591ca1a483d": {
      "fields": {
        "019fba9e-d801-7fd7-9944-40ed3ab99414": {
          "name": "name",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7044-b1f1-8d217fb57993": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-7521-a063-bf8107d9a3d0"
      },
      "type": "object"
    }
  },
  "name": "CollectionOutput",
  "schemas": {
    "019fba9e-d801-7521-a063-bf8107d9a3d0": {
      "fields": {
        "019fba9e-d801-7651-b4eb-50d11e319628": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7e38-920b-cf6baf03dc0c": {
          "name": "schema",
          "type": "record"
        },
        "019fba9e-d803-7fb1-843b-9c67edce29c7": {
          "name": "created",
          "type": "string"
        },
        "019fba9e-d804-7e41-98ff-6ff5c57b093a": {
          "name": "updated",
          "type": "string"
        }
      },
      "name": "CollectionMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Create collection via API

**`POST`** `/system/collections/collection/create`

Create collection via API

- **Handler:** `system:collections:collection:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7702-90ce-dd104d16c3b8": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "CollectionCreateInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7044-b1f1-8d217fb57993": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-7521-a063-bf8107d9a3d0"
      },
      "type": "object"
    }
  },
  "name": "CollectionOutput",
  "schemas": {
    "019fba9e-d801-7521-a063-bf8107d9a3d0": {
      "fields": {
        "019fba9e-d801-7651-b4eb-50d11e319628": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7e38-920b-cf6baf03dc0c": {
          "name": "schema",
          "type": "record"
        },
        "019fba9e-d803-7fb1-843b-9c67edce29c7": {
          "name": "created",
          "type": "string"
        },
        "019fba9e-d804-7e41-98ff-6ff5c57b093a": {
          "name": "updated",
          "type": "string"
        }
      },
      "name": "CollectionMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete collection via API

**`DELETE`** `/system/collections/collection/delete/{name}`

Delete collection via API

- **Handler:** `system:collections:collection:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7799-b3a5-f458bfbbf3e1": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7947-b3f6-9f186e96cafe"
      },
      "type": "object"
    }
  },
  "name": "CollectionDeleteInput",
  "schemas": {
    "019fba9e-d801-7947-b3f6-9f186e96cafe": {
      "fields": {
        "019fba9e-d801-7870-bdd0-ca4c1d95bbb6": {
          "name": "name",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

---

## document

### Query collection documents

**`POST`** `/system/collections/document/query/{name}`

Query collection documents

- **Handler:** `system:collections:document:query`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-74dd-b6bc-4abdc193221e": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-70c2-96ff-fd6b618c31d4"
      },
      "type": "object"
    },
    "019fba9e-d802-7435-8c76-c3cbb696ed3f": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "CollectionDocQueryInput",
  "schemas": {
    "019fba9e-d801-70c2-96ff-fd6b618c31d4": {
      "fields": {
        "019fba9e-d801-76b2-ba8b-02bb05b9f987": {
          "name": "name",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7f59-bdba-a5ebe6f518e7": {
      "name": "page",
      "type": "record"
    }
  },
  "name": "CollectionQueryOutput",
  "version": "1.0.0"
}
```

---

### Create document in collection

**`POST`** `/system/collections/document/create/{name}`

Create document in collection

- **Handler:** `system:collections:document:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7cb7-aec0-cd3a86b83d58": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7836-8305-9ef902620110"
      },
      "type": "object"
    },
    "019fba9e-d802-77b3-affb-83f705c1cf7b": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "CollectionDocCreateInput",
  "schemas": {
    "019fba9e-d801-7836-8305-9ef902620110": {
      "fields": {
        "019fba9e-d801-76c6-a9a4-bba87f0201c9": {
          "name": "name",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-719e-abe8-e5f896fd165d": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-73ec-a7df-a34a4f5682b7"
      },
      "type": "object"
    }
  },
  "name": "CollectionDocumentOutput",
  "schemas": {
    "019fba9e-d801-73ec-a7df-a34a4f5682b7": {
      "fields": {
        "019fba9e-d801-778d-a5cc-c7f65f0b0d1e": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-78ca-86fb-1e559cd9f301": {
          "name": "data",
          "type": "record"
        }
      },
      "name": "CollectionDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---

### Get document from collection

**`GET`** `/system/collections/document/get/{name}/{doc_id}`

Get document from collection

- **Handler:** `system:collections:document:get`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7a7f-ac2d-a4d2fd22d2ab": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7f58-a4e6-512ffd6ea7ca"
      },
      "type": "object"
    }
  },
  "name": "CollectionDocGetInput",
  "schemas": {
    "019fba9e-d801-7f58-a4e6-512ffd6ea7ca": {
      "fields": {
        "019fba9e-d801-79c3-941e-be7b679ffd0b": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7cc1-b8e3-419a5ec541b3": {
          "name": "doc_id",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-719e-abe8-e5f896fd165d": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-73ec-a7df-a34a4f5682b7"
      },
      "type": "object"
    }
  },
  "name": "CollectionDocumentOutput",
  "schemas": {
    "019fba9e-d801-73ec-a7df-a34a4f5682b7": {
      "fields": {
        "019fba9e-d801-778d-a5cc-c7f65f0b0d1e": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-78ca-86fb-1e559cd9f301": {
          "name": "data",
          "type": "record"
        }
      },
      "name": "CollectionDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---

### Update document in collection

**`PATCH`** `/system/collections/document/update/{name}/{doc_id}`

Update document in collection

- **Handler:** `system:collections:document:update`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7339-805a-b0a00df8b519": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7162-903d-50f388813d19"
      },
      "type": "object"
    },
    "019fba9e-d802-7816-a95a-cb287c905282": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "CollectionDocUpdateInput",
  "schemas": {
    "019fba9e-d801-7162-903d-50f388813d19": {
      "fields": {
        "019fba9e-d801-7a30-a5a5-ac7bc17dd02d": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7a8c-a480-44ca88d4b6ee": {
          "name": "doc_id",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-719e-abe8-e5f896fd165d": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-73ec-a7df-a34a4f5682b7"
      },
      "type": "object"
    }
  },
  "name": "CollectionDocumentOutput",
  "schemas": {
    "019fba9e-d801-73ec-a7df-a34a4f5682b7": {
      "fields": {
        "019fba9e-d801-778d-a5cc-c7f65f0b0d1e": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-78ca-86fb-1e559cd9f301": {
          "name": "data",
          "type": "record"
        }
      },
      "name": "CollectionDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete document from collection

**`DELETE`** `/system/collections/document/delete/{name}/{doc_id}`

Delete document from collection

- **Handler:** `system:collections:document:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7ed5-9939-298949280db1": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-715e-87ef-b14adbca8eb3"
      },
      "type": "object"
    }
  },
  "name": "CollectionDocDeleteInput",
  "schemas": {
    "019fba9e-d801-715e-87ef-b14adbca8eb3": {
      "fields": {
        "019fba9e-d801-73aa-ad1d-be4899190c48": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7f30-82e4-1c5df0b613c7": {
          "name": "doc_id",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

---

## _user

### Query users collection

**`GET`** `/system/collections/_user/read`

Query users collection

- **Handler:** `system:collections:_user:read`
- **Internal:** Yes

---

## _api_key

### Query API keys collection

**`GET`** `/system/collections/_api_key/read`

Query API keys collection

- **Handler:** `system:collections:_api_key:read`
- **Internal:** Yes

---

## _operation_policy

### Query policy operations

**`GET`** `/system/collections/_operation_policy/read`

Query policy operations

- **Handler:** `system:collections:_operation_policy:read`
- **Internal:** Yes

---

## _iam_rule

### Query policy rules

**`GET`** `/system/collections/_iam_rule/read`

Query policy rules

- **Handler:** `system:collections:_iam_rule:read`
- **Internal:** Yes

---

## _access_log

### Query access logs

**`GET`** `/system/collections/_access_log/read`

Query access logs

- **Handler:** `system:collections:_access_log:read`
- **Internal:** Yes

---

## user

### Query users collection

**`POST`** `/system/collections/user/query`

Query users collection

- **Handler:** `system:collections:user:query`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7991-857c-9fd90211b1be": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-70e0-91bc-0b939d8659ad"
      },
      "type": "object"
    }
  },
  "name": "UserQueryInput",
  "schemas": {
    "019fba9e-d801-70e0-91bc-0b939d8659ad": {
      "fields": {
        "019fba9e-d801-7a2a-849b-f89acee3c714": {
          "name": "username",
          "type": "string"
        },
        "019fba9e-d802-7681-8b0d-3c346ee7dd44": {
          "name": "limit",
          "type": "integer"
        },
        "019fba9e-d803-7af6-8e9c-123838dbeb50": {
          "name": "cursor",
          "type": "string"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-759d-87bf-7ed4e27ded31": {
      "name": "page",
      "schema": {
        "id": "019fba9e-d801-7459-a3a8-88ecd9738fca"
      },
      "type": "object"
    }
  },
  "name": "UserQueryOutput",
  "schemas": {
    "019fba9e-d801-7459-a3a8-88ecd9738fca": {
      "fields": {
        "019fba9e-d801-7335-9889-dca299aaaa33": {
          "name": "documents",
          "schema": {
            "id": "019fba9e-d802-71ef-924b-a1944a8e1507"
          },
          "type": "array"
        },
        "019fba9e-d802-77a0-8f51-41cd469bad44": {
          "name": "pagination",
          "schema": {
            "id": "019fba9e-d804-7f0d-bcc3-c32e7a2b616d"
          },
          "type": "object"
        }
      },
      "name": "page"
    },
    "019fba9e-d802-71ef-924b-a1944a8e1507": {
      "fields": {
        "019fba9e-d801-77b8-b367-53c9f288dabd": {
          "name": "permissions",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d802-7231-8143-384525159faa": {
          "name": "_id_",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-71a1-b6c7-f96d315e52c5": {
          "name": "email",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7522-abd3-58f27b3e8044": {
          "name": "name",
          "required": true,
          "type": "string"
        },
        "019fba9e-d805-710d-8a3f-aafdb50a7f67": {
          "name": "_metadata_",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d803-7024-9b43-be87087f8458"
          },
          "type": "object"
        },
        "019fba9e-d806-7373-98cf-e092919c0863": {
          "name": "data",
          "type": "record"
        },
        "019fba9e-d807-746b-9f3f-155c3bde2496": {
          "default": -1,
          "name": "disabled",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d808-7073-9b1c-d1fbc993306c": {
          "name": "settings",
          "type": "record"
        },
        "019fba9e-d809-781e-b662-982061f64d91": {
          "name": "tenant_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d80a-7385-bba3-3c3d209c411b": {
          "default": 0,
          "name": "token_version",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d80b-7cc2-b277-4f334593f583": {
          "name": "verified",
          "nullable": true,
          "type": "boolean"
        }
      },
      "name": "UserPublic"
    },
    "019fba9e-d803-7024-9b43-be87087f8458": {
      "fields": {
        "019fba9e-d801-7252-b37e-3289089e5e32": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-7da6-8e1a-f5c44dc92315": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7dff-bae6-46a5711effd9": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7bb8-9ea6-7dfa1b59a146": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-7858-965d-fcf50fb7ff04": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-78b6-87fa-85cdd6d3d01c": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemUserMetadata"
    },
    "019fba9e-d804-7f0d-bcc3-c32e7a2b616d": {
      "fields": {
        "019fba9e-d801-7966-b3b2-d74fad553d28": {
          "name": "total",
          "type": "integer"
        },
        "019fba9e-d802-7cc6-a67d-cc98e9ab9537": {
          "name": "cursor",
          "type": "string"
        },
        "019fba9e-d803-7aef-9b00-46265dd03d88": {
          "name": "limit",
          "type": "integer"
        }
      },
      "name": "pagination"
    }
  },
  "version": "1.0.0"
}
```

---

## audit_log

### Query audit logs

**`POST`** `/system/collections/audit_log/query`

Query audit logs

- **Handler:** `system:collections:audit_log:query`

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
