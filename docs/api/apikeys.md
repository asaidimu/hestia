# API Keys API

## key

### List own API keys

**`GET`** `/system/apikeys/key/list`

List own API keys

- **Handler:** `system:apikeys:key:list`

#### Request Body

```json
{
  "name": "APIKeyListInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7051-83ab-d9398d4b9fb9": {
      "name": "cs",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7c79-8568-c9e703ea8bcd"
      },
      "type": "object"
    },
    "019fba9e-d802-77e4-ba34-be268aeff31d": {
      "name": "c",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d806-78df-abf3-68a3241931f9"
      },
      "type": "object"
    },
    "019fba9e-d803-7884-844b-696dcf021431": {
      "name": "ctx",
      "type": "unknown"
    },
    "019fba9e-d804-7380-8af5-a853585a96dc": {
      "name": "pool",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d812-7cbb-8125-a59a1ba1fada"
      },
      "type": "object"
    },
    "019fba9e-d805-74aa-8478-c3ce2a87b88a": {
      "name": "prefix",
      "schema": {
        "type": "integer"
      },
      "type": "array"
    },
    "019fba9e-d806-7641-bc2c-99405dde66cf": {
      "name": "slot_idx",
      "type": "integer"
    },
    "019fba9e-d807-79a7-ac07-a5cda5988e99": {
      "name": "record",
      "type": "record"
    }
  },
  "name": "Document",
  "schemas": {
    "019fba9e-d801-7c79-8568-c9e703ea8bcd": {
      "fields": {
        "019fba9e-d801-70eb-ac69-e0dfd40aba7d": {
          "name": "descriptors",
          "schema": {
            "type": "integer"
          },
          "type": "array"
        },
        "019fba9e-d802-782e-81f3-ee30f9c05dab": {
          "name": "fields_meta",
          "schema": {
            "id": "019fba9e-d802-792f-b95a-bb508d0e3e2e"
          },
          "type": "array"
        },
        "019fba9e-d803-75bb-8a6c-3512c225afca": {
          "name": "schemas",
          "schema": {
            "id": "019fba9e-d804-7f56-901c-055b8fc5d6d0"
          },
          "type": "array"
        },
        "019fba9e-d804-7434-a274-0a6ee0514f42": {
          "name": "schemas_meta",
          "schema": {
            "id": "019fba9e-d805-732a-a2ff-262abe38c020"
          },
          "type": "array"
        },
        "019fba9e-d805-7c78-b16b-eef31d79b826": {
          "name": "defaults",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d806-78df-abf3-68a3241931f9"
          },
          "type": "object"
        },
        "019fba9e-d806-73bf-964a-6fc05d39a04a": {
          "name": "enums",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d806-78df-abf3-68a3241931f9"
          },
          "type": "object"
        },
        "019fba9e-d807-78be-a583-27ee14ee3b62": {
          "name": "variants",
          "schema": {
            "type": "unknown"
          },
          "type": "record"
        },
        "019fba9e-d808-787b-bcce-a2db2cc65222": {
          "name": "constraints",
          "schema": {
            "id": "019fba9e-d807-7fdf-81c5-f619ae66e040"
          },
          "type": "array"
        },
        "019fba9e-d809-7fd6-aad0-d7efd26a7af5": {
          "name": "indexes",
          "schema": {
            "id": "019fba9e-d80a-79a2-b16f-2d33b1f3bba6"
          },
          "type": "record"
        },
        "019fba9e-d80a-7177-ac4e-4bd42b5aacc4": {
          "name": "field_types",
          "type": "bytes"
        },
        "019fba9e-d80b-7d6c-a44b-e31fa3652451": {
          "name": "local_offsets",
          "schema": {
            "type": "integer"
          },
          "type": "array"
        },
        "019fba9e-d80c-7e65-aee8-45ec1a5146fc": {
          "name": "schema_constraints",
          "schema": {
            "type": "unknown"
          },
          "type": "array"
        },
        "019fba9e-d80d-771c-8787-946138ed9f97": {
          "name": "field_ref_constraints",
          "schema": {
            "type": "unknown"
          },
          "type": "record"
        },
        "019fba9e-d80e-748e-8e30-34eb05eb842d": {
          "name": "addr_mu",
          "schema": {
            "id": "019fba9e-d80c-700f-bc3a-cc42f9742213"
          },
          "type": "object"
        },
        "019fba9e-d80f-72cb-ae84-5677d50acc89": {
          "name": "addr_cache",
          "schema": {
            "type": "unknown"
          },
          "type": "record"
        },
        "019fba9e-d810-7897-8a58-d1f1b8210d90": {
          "name": "path_by_addr",
          "schema": {
            "type": "unknown"
          },
          "type": "record"
        },
        "019fba9e-d811-79d3-8a6d-1be350db0f5e": {
          "name": "name_by_addr",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "CompiledSchema"
    },
    "019fba9e-d802-792f-b95a-bb508d0e3e2e": {
      "fields": {
        "019fba9e-d801-7ffa-ad3b-22126c4bf68e": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-7d0e-b08a-04d779832dfc": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d803-755a-8db3-3e5eb32316ed": {
          "name": "path",
          "type": "string"
        },
        "019fba9e-d804-77f3-9ded-a2c9dfdbc181": {
          "name": "parts",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d805-780e-8195-9bc487eb84e3": {
          "name": "description",
          "type": "string"
        },
        "019fba9e-d806-7093-b4e7-b68442c0d758": {
          "name": "default",
          "schema": {
            "id": "019fba9e-d803-72a0-ae3d-63cfa39caaed"
          },
          "type": "object"
        }
      },
      "name": "FieldMeta"
    },
    "019fba9e-d803-72a0-ae3d-63cfa39caaed": {
      "fields": {
        "019fba9e-d801-7350-9c81-ec143560a0f6": {
          "name": "kind",
          "type": "integer"
        },
        "019fba9e-d802-7202-a7c6-325596cb478a": {
          "name": "value",
          "type": "unknown"
        }
      },
      "name": "LiteralValue"
    },
    "019fba9e-d804-7f56-901c-055b8fc5d6d0": {
      "fields": {
        "019fba9e-d801-7f47-9363-464e3baf4e7b": {
          "name": "field_start",
          "type": "integer"
        },
        "019fba9e-d802-79f1-99a8-2363be654d5b": {
          "name": "field_count",
          "type": "integer"
        },
        "019fba9e-d803-77d3-85a7-222f0da392f5": {
          "name": "footprint",
          "type": "integer"
        }
      },
      "name": "SchemaSlot"
    },
    "019fba9e-d805-732a-a2ff-262abe38c020": {
      "fields": {
        "019fba9e-d801-78c9-8fce-945ce40dce8f": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7c67-bf8f-3c19eaf30096": {
          "name": "description",
          "type": "string"
        }
      },
      "name": "SchemaMeta"
    },
    "019fba9e-d806-78df-abf3-68a3241931f9": {
      "fields": {
        "019fba9e-d801-7cf8-a398-a2c84f53360b": {
          "name": "data",
          "type": "unknown"
        },
        "019fba9e-d802-74e6-9cf1-8aad00f58d36": {
          "name": "positions",
          "schema": {
            "type": "integer"
          },
          "type": "record"
        },
        "019fba9e-d803-76fc-877d-fa2c2b0a1b15": {
          "name": "holes",
          "schema": {
            "type": "integer"
          },
          "type": "array"
        }
      },
      "name": "DataContainer"
    },
    "019fba9e-d807-7fdf-81c5-f619ae66e040": {
      "fields": {
        "019fba9e-d801-781d-94a6-e1d9e9ffa3b4": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-74b4-8c84-1cad29ee44d1": {
          "name": "scope",
          "type": "integer"
        },
        "019fba9e-d803-78d2-be31-edcbbe4497ce": {
          "name": "rule",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d808-7e0f-bb62-d05dd24069b3"
          },
          "type": "object"
        },
        "019fba9e-d804-7506-a72d-6a7a2e099f09": {
          "name": "group",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d809-791e-ba77-40ef8c94911b"
          },
          "type": "object"
        },
        "019fba9e-d805-7da5-8c91-9338dd33e240": {
          "name": "abs_field_paths",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d806-7b98-b876-0b627446579e": {
          "name": "abs_field_parts",
          "schema": {
            "type": "unknown"
          },
          "type": "array"
        },
        "019fba9e-d807-7b82-9ded-9759758edfa8": {
          "name": "rel_field_paths",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d808-7631-8d2f-a1c9585700e7": {
          "name": "rel_field_parts",
          "schema": {
            "type": "unknown"
          },
          "type": "array"
        }
      },
      "name": "ResolvedConstraint"
    },
    "019fba9e-d808-7e0f-bb62-d05dd24069b3": {
      "fields": {
        "019fba9e-d801-7f22-89c5-2cbc3954792b": {
          "name": "predicate",
          "type": "string"
        },
        "019fba9e-d802-7264-b660-10b98eae83f8": {
          "name": "fields",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d803-7624-bf1c-801c2f0ff99e": {
          "name": "parameters",
          "schema": {
            "id": "019fba9e-d803-72a0-ae3d-63cfa39caaed"
          },
          "type": "object"
        }
      },
      "name": "ResolvedConstraintRule"
    },
    "019fba9e-d809-791e-ba77-40ef8c94911b": {
      "fields": {
        "019fba9e-d801-711b-8d84-6ffe9f65e615": {
          "name": "operator",
          "type": "integer"
        },
        "019fba9e-d802-7f8c-b1f3-432422c448e5": {
          "name": "members",
          "schema": {
            "id": "019fba9e-d807-7fdf-81c5-f619ae66e040"
          },
          "type": "array"
        },
        "019fba9e-d803-74c6-b279-c476b92e1318": {
          "name": "required_field_paths",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d804-704e-b47d-a12f2b761984": {
          "name": "required_field_parts",
          "schema": {
            "type": "unknown"
          },
          "type": "array"
        }
      },
      "name": "ResolvedConstraintGroup"
    },
    "019fba9e-d80a-79a2-b16f-2d33b1f3bba6": {
      "fields": {
        "019fba9e-d801-7a3f-94bb-779734528276": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-793d-9ce3-f721b15758b5": {
          "name": "description",
          "type": "string"
        },
        "019fba9e-d803-7a8e-a788-b872810556e7": {
          "name": "order",
          "type": "string"
        },
        "019fba9e-d804-7e85-b482-d4767f6ea599": {
          "name": "condition",
          "schema": {
            "id": "019fba9e-d80b-7b50-b476-19de6062ab65"
          },
          "type": "object"
        },
        "019fba9e-d805-7db9-9691-c9dd6286b8ab": {
          "name": "fields",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d806-7b9e-adcf-679d08ca6223": {
          "name": "type",
          "type": "integer"
        },
        "019fba9e-d807-7fa3-b913-c5170437fcdc": {
          "name": "unique",
          "type": "boolean"
        }
      },
      "name": "Index"
    },
    "019fba9e-d80b-7b50-b476-19de6062ab65": {
      "fields": {
        "019fba9e-d801-70bc-b4ab-09bb87336afd": {
          "name": "kind",
          "type": "integer"
        },
        "019fba9e-d802-78a7-930b-b5598bbf5be3": {
          "name": "payload",
          "type": "unknown"
        }
      },
      "name": "IndexConditionUnion"
    },
    "019fba9e-d80c-700f-bc3a-cc42f9742213": {
      "fields": {
        "019fba9e-d801-7633-95f4-1401d98962da": {
          "name": "w",
          "schema": {
            "id": "019fba9e-d80d-7fea-8f72-3211ec47cd29"
          },
          "type": "object"
        },
        "019fba9e-d802-7ef3-b960-9c592cc7ea15": {
          "name": "writer_sem",
          "type": "integer"
        },
        "019fba9e-d803-7735-ab01-0ee153b5c88e": {
          "name": "reader_sem",
          "type": "integer"
        },
        "019fba9e-d804-7394-8b21-d2b6cbc48abf": {
          "name": "reader_count",
          "schema": {
            "id": "019fba9e-d810-7ae5-ab22-75a2fb537f9e"
          },
          "type": "object"
        },
        "019fba9e-d805-74f1-a8e0-98abd9644dd7": {
          "name": "reader_wait",
          "schema": {
            "id": "019fba9e-d810-7ae5-ab22-75a2fb537f9e"
          },
          "type": "object"
        }
      },
      "name": "RWMutex"
    },
    "019fba9e-d80d-7fea-8f72-3211ec47cd29": {
      "fields": {
        "019fba9e-d801-7c91-990e-f06df99fff11": {
          "name": "_",
          "schema": {
            "id": "019fba9e-d80e-7d63-9cc0-02b3d8d6d890"
          },
          "type": "object"
        },
        "019fba9e-d802-7fc0-aa01-964172f5e716": {
          "name": "mu",
          "schema": {
            "id": "019fba9e-d80f-7f34-8e26-712e42932324"
          },
          "type": "object"
        }
      },
      "name": "Mutex"
    },
    "019fba9e-d80e-7d63-9cc0-02b3d8d6d890": {
      "name": "noCopy"
    },
    "019fba9e-d80f-7f34-8e26-712e42932324": {
      "fields": {
        "019fba9e-d801-7778-aa18-7b602fe13b74": {
          "name": "state",
          "type": "integer"
        },
        "019fba9e-d802-7e08-ae01-b7011342339e": {
          "name": "sema",
          "type": "integer"
        }
      },
      "name": "Mutex"
    },
    "019fba9e-d810-7ae5-ab22-75a2fb537f9e": {
      "fields": {
        "019fba9e-d801-7300-9656-6efb16ee3852": {
          "name": "_",
          "schema": {
            "id": "019fba9e-d811-77c7-a450-241382584fd6"
          },
          "type": "object"
        },
        "019fba9e-d802-7cb9-9483-998bef7c2e8c": {
          "name": "v",
          "type": "integer"
        }
      },
      "name": "Int32"
    },
    "019fba9e-d811-77c7-a450-241382584fd6": {
      "name": "noCopy"
    },
    "019fba9e-d812-7cbb-8125-a59a1ba1fada": {
      "fields": {
        "019fba9e-d801-71fc-b9de-b91976dcf20d": {
          "name": "pool",
          "schema": {
            "id": "019fba9e-d813-7211-9f7f-98029a005ec8"
          },
          "type": "object"
        }
      },
      "name": "Pool"
    },
    "019fba9e-d813-7211-9f7f-98029a005ec8": {
      "fields": {
        "019fba9e-d801-7aa7-ac77-b793e063149c": {
          "name": "no_copy",
          "schema": {
            "id": "019fba9e-d80e-7d63-9cc0-02b3d8d6d890"
          },
          "type": "object"
        },
        "019fba9e-d802-7472-b5dd-833a80ae23c7": {
          "name": "local",
          "type": "unknown"
        },
        "019fba9e-d803-7177-85c2-5cb4168eb714": {
          "name": "local_size",
          "type": "unknown"
        },
        "019fba9e-d804-727d-b4f9-2698124b2532": {
          "name": "victim",
          "type": "unknown"
        },
        "019fba9e-d805-77b2-93cd-24d4a4f5a3cb": {
          "name": "victim_size",
          "type": "unknown"
        },
        "019fba9e-d806-77d3-a6a5-802d19da8b02": {
          "name": "new",
          "type": "unknown"
        }
      },
      "name": "Pool"
    }
  },
  "version": "1.0.0"
}
```

---

### Get own API key details

**`GET`** `/system/apikeys/key/get/{key_id}`

Get own API key details

- **Handler:** `system:apikeys:key:get`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-797b-8a0f-5901caaad1ef": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7dd7-8449-b22f422fc42b"
      },
      "type": "object"
    }
  },
  "name": "APIKeyGetInput",
  "schemas": {
    "019fba9e-d801-7dd7-8449-b22f422fc42b": {
      "fields": {
        "019fba9e-d801-7d63-9d98-7a273f990a31": {
          "name": "key_id",
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
    "019fba9e-d801-73fc-a748-c1d103c70c75": {
      "name": "operations",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-701a-ba9b-cda555cd3075": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-7408-9f62-59ed3fc0e43f": {
      "name": "userId",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-70d6-8eb6-92394452ca7f": {
      "name": "prefix",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-7d25-86e5-804e5816f944": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-761c-80ab-5ae91de48ded": {
      "name": "hash",
      "required": true,
      "type": "string"
    },
    "019fba9e-d807-7044-bd31-4fcf9a7be972": {
      "name": "expiry",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d808-7b5c-b550-272de4f70702": {
      "name": "limits",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-79b1-add1-205c1961ec97"
      },
      "type": "object"
    },
    "019fba9e-d809-7952-823b-14171ca0071b": {
      "name": "last_used",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80a-7cf9-99e7-fc9fbab20b07": {
      "name": "ip",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d802-7e3c-a7cb-bae9eadbdf68"
      },
      "type": "object"
    },
    "019fba9e-d80b-7f69-bd7d-41dcdd4b836c": {
      "name": "environment",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80c-7f35-9a21-8efe840faaf9": {
      "name": "status",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80d-77d6-8329-8d938357a73a": {
      "name": "usage",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80e-7f53-8739-3164ecb550a7": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d803-78f8-9b3f-253c9c439754"
      },
      "type": "object"
    }
  },
  "name": "SystemAPIKey",
  "schemas": {
    "019fba9e-d801-79b1-add1-205c1961ec97": {
      "fields": {
        "019fba9e-d801-7ae3-b9f4-9518a1c8e799": {
          "name": "rph",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d802-77be-b9fb-a8ffccf7aa81": {
          "name": "rpm",
          "nullable": true,
          "type": "integer"
        }
      },
      "name": "Limits"
    },
    "019fba9e-d802-7e3c-a7cb-bae9eadbdf68": {
      "fields": {
        "019fba9e-d801-788b-a8ac-e45afd4ef5ad": {
          "name": "whitelist",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "IPConfig"
    },
    "019fba9e-d803-78f8-9b3f-253c9c439754": {
      "fields": {
        "019fba9e-d801-7764-b7d5-ece48e7541d7": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-7c0d-a8ae-60b6a8549596": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7e82-a0cd-b2b2eb5767cc": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7583-9d01-decdbf1e29a7": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-7890-9d15-d290019e1a29": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7f3b-903b-babe8783cd44": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemAPIKeyMetadata"
    }
  },
  "version": "1.0.0"
}
```

---

### Create a new API key

**`POST`** `/system/apikeys/key/create`

Create a new API key

- **Handler:** `system:apikeys:key:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7cd6-bfc0-df38375723e6": {
      "name": "payload",
      "required": true,
      "schema": {
        "id": "019fba9e-d801-7b3e-be47-c0fe32f9e186"
      },
      "type": "object"
    }
  },
  "name": "APIKeyCreate",
  "schemas": {
    "019fba9e-d801-7b3e-be47-c0fe32f9e186": {
      "fields": {
        "019fba9e-d801-7dd7-b0ad-89c57f21c708": {
          "name": "operations",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d802-747c-9ba8-a81d4d60ce03": {
          "name": "name",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7d7d-a33b-007919679b34": {
          "name": "environment",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d804-7c0d-a570-bd435c374480": {
          "name": "expiry",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-723d-acc9-56fcc6443cd6": {
          "name": "ip",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d802-7e3c-a7cb-bae9eadbdf68"
          },
          "type": "object"
        },
        "019fba9e-d806-75c7-93e4-95fc75061713": {
          "name": "limits",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d803-79b1-add1-205c1961ec97"
          },
          "type": "object"
        }
      },
      "name": "payload"
    },
    "019fba9e-d802-7e3c-a7cb-bae9eadbdf68": {
      "fields": {
        "019fba9e-d801-788b-a8ac-e45afd4ef5ad": {
          "name": "whitelist",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "IPConfig"
    },
    "019fba9e-d803-79b1-add1-205c1961ec97": {
      "fields": {
        "019fba9e-d801-7b43-ab22-53f847871d30": {
          "name": "rph",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d802-7635-89af-096f966eff30": {
          "name": "rpm",
          "nullable": true,
          "type": "integer"
        }
      },
      "name": "Limits"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7cb7-8248-0b4db67110f9": {
      "name": "operations",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-700c-8175-16179a87313d": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-7139-9374-220d88b3f38d": {
      "name": "userId",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-76fb-92b1-a6d60e4c1e0a": {
      "name": "prefix",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-76c4-a401-a01c35df1cf6": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-7b77-91d6-1b6b3bab593e": {
      "name": "expiry",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d807-7772-927b-c938986ce9b6": {
      "name": "limits",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-79b1-add1-205c1961ec97"
      },
      "type": "object"
    },
    "019fba9e-d808-7356-868f-d2e1464f3b88": {
      "name": "last_used",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d809-706f-b554-c28833f0ae00": {
      "name": "ip",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d802-7e3c-a7cb-bae9eadbdf68"
      },
      "type": "object"
    },
    "019fba9e-d80a-7625-a1dd-bf7f2e963099": {
      "name": "environment",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80b-78e1-85f1-8e000a0ea5be": {
      "name": "status",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80c-70d1-b816-8c95944ec6ae": {
      "name": "usage",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80d-7856-868b-f6bcd7a4dfd9": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d803-78f8-9b3f-253c9c439754"
      },
      "type": "object"
    },
    "019fba9e-d80e-7a13-b92f-a9079d36ecc8": {
      "name": "key",
      "type": "string"
    }
  },
  "name": "APIKeyCreatedOutput",
  "schemas": {
    "019fba9e-d801-79b1-add1-205c1961ec97": {
      "fields": {
        "019fba9e-d801-7ae3-b9f4-9518a1c8e799": {
          "name": "rph",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d802-77be-b9fb-a8ffccf7aa81": {
          "name": "rpm",
          "nullable": true,
          "type": "integer"
        }
      },
      "name": "Limits"
    },
    "019fba9e-d802-7e3c-a7cb-bae9eadbdf68": {
      "fields": {
        "019fba9e-d801-788b-a8ac-e45afd4ef5ad": {
          "name": "whitelist",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "IPConfig"
    },
    "019fba9e-d803-78f8-9b3f-253c9c439754": {
      "fields": {
        "019fba9e-d801-7764-b7d5-ece48e7541d7": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-7c0d-a8ae-60b6a8549596": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7e82-a0cd-b2b2eb5767cc": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7583-9d01-decdbf1e29a7": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-7890-9d15-d290019e1a29": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7f3b-903b-babe8783cd44": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemAPIKeyMetadata"
    }
  },
  "version": "1.0.0"
}
```

---

### Update API key metadata

**`PATCH`** `/system/apikeys/key/update/{key_id}`

Update API key metadata

- **Handler:** `system:apikeys:key:update`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7cd3-a53f-b539eb619ed4": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-70b6-bcda-f32005b128b3"
      },
      "type": "object"
    },
    "019fba9e-d802-7189-b3da-5b8ec0b3d1b6": {
      "name": "arguments",
      "required": true,
      "schema": {
        "id": "019fba9e-d802-73b0-82f2-a7fedbc8b317"
      },
      "type": "object"
    }
  },
  "name": "APIKeyUpdate",
  "schemas": {
    "019fba9e-d801-70b6-bcda-f32005b128b3": {
      "fields": {
        "019fba9e-d801-7c91-a11d-63715d54e5fe": {
          "name": "operations",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d802-729a-98c8-5f9ad2033ddc": {
          "name": "environment",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d803-7965-80ca-df67f621c712": {
          "name": "expiry",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d804-7ba5-b9a9-023fc5a99b37": {
          "name": "ip",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d803-7e3c-a7cb-bae9eadbdf68"
          },
          "type": "object"
        },
        "019fba9e-d805-7178-87c4-414c2bf65734": {
          "name": "limits",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d804-79b1-add1-205c1961ec97"
          },
          "type": "object"
        },
        "019fba9e-d806-73fe-9cff-d2094cd26fe1": {
          "name": "name",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d807-7f72-a767-fc2e3f53c3da": {
          "name": "status",
          "nullable": true,
          "type": "string"
        }
      },
      "name": "payload"
    },
    "019fba9e-d802-73b0-82f2-a7fedbc8b317": {
      "fields": {
        "019fba9e-d801-7955-864f-50b6a9736813": {
          "name": "key_id",
          "required": true,
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d803-7e3c-a7cb-bae9eadbdf68": {
      "fields": {
        "019fba9e-d801-76c4-9caf-eee8e998d4ec": {
          "name": "whitelist",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "IPConfig"
    },
    "019fba9e-d804-79b1-add1-205c1961ec97": {
      "fields": {
        "019fba9e-d801-769d-983a-a54a7ac25775": {
          "name": "rph",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d802-7ec3-ab45-f09016f0cd6a": {
          "name": "rpm",
          "nullable": true,
          "type": "integer"
        }
      },
      "name": "Limits"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-73fc-a748-c1d103c70c75": {
      "name": "operations",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-701a-ba9b-cda555cd3075": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-7408-9f62-59ed3fc0e43f": {
      "name": "userId",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-70d6-8eb6-92394452ca7f": {
      "name": "prefix",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-7d25-86e5-804e5816f944": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-761c-80ab-5ae91de48ded": {
      "name": "hash",
      "required": true,
      "type": "string"
    },
    "019fba9e-d807-7044-bd31-4fcf9a7be972": {
      "name": "expiry",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d808-7b5c-b550-272de4f70702": {
      "name": "limits",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-79b1-add1-205c1961ec97"
      },
      "type": "object"
    },
    "019fba9e-d809-7952-823b-14171ca0071b": {
      "name": "last_used",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80a-7cf9-99e7-fc9fbab20b07": {
      "name": "ip",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d802-7e3c-a7cb-bae9eadbdf68"
      },
      "type": "object"
    },
    "019fba9e-d80b-7f69-bd7d-41dcdd4b836c": {
      "name": "environment",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80c-7f35-9a21-8efe840faaf9": {
      "name": "status",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80d-77d6-8329-8d938357a73a": {
      "name": "usage",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80e-7f53-8739-3164ecb550a7": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d803-78f8-9b3f-253c9c439754"
      },
      "type": "object"
    }
  },
  "name": "SystemAPIKey",
  "schemas": {
    "019fba9e-d801-79b1-add1-205c1961ec97": {
      "fields": {
        "019fba9e-d801-7ae3-b9f4-9518a1c8e799": {
          "name": "rph",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d802-77be-b9fb-a8ffccf7aa81": {
          "name": "rpm",
          "nullable": true,
          "type": "integer"
        }
      },
      "name": "Limits"
    },
    "019fba9e-d802-7e3c-a7cb-bae9eadbdf68": {
      "fields": {
        "019fba9e-d801-788b-a8ac-e45afd4ef5ad": {
          "name": "whitelist",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "IPConfig"
    },
    "019fba9e-d803-78f8-9b3f-253c9c439754": {
      "fields": {
        "019fba9e-d801-7764-b7d5-ece48e7541d7": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-7c0d-a8ae-60b6a8549596": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7e82-a0cd-b2b2eb5767cc": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7583-9d01-decdbf1e29a7": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-7890-9d15-d290019e1a29": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7f3b-903b-babe8783cd44": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemAPIKeyMetadata"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete an API key

**`DELETE`** `/system/apikeys/key/delete/{key_id}`

Delete an API key

- **Handler:** `system:apikeys:key:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-76e7-aa49-5a043dffa058": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-77ef-89b7-46fa041f2291"
      },
      "type": "object"
    }
  },
  "name": "APIKeyDeleteInput",
  "schemas": {
    "019fba9e-d801-77ef-89b7-46fa041f2291": {
      "fields": {
        "019fba9e-d801-7e6d-ab88-6e652252a237": {
          "name": "key_id",
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

### Rotate API key material

**`POST`** `/system/apikeys/key/rotate/{key_id}`

Rotate API key material

- **Handler:** `system:apikeys:key:rotate`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7a26-9c86-afcdf968b61c": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-713a-9a4a-b020684bff7c"
      },
      "type": "object"
    }
  },
  "name": "APIKeyRotateInput",
  "schemas": {
    "019fba9e-d801-713a-9a4a-b020684bff7c": {
      "fields": {
        "019fba9e-d801-7df8-aa36-6313af3c1327": {
          "name": "key_id",
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
    "019fba9e-d801-7cb7-8248-0b4db67110f9": {
      "name": "operations",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-700c-8175-16179a87313d": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-7139-9374-220d88b3f38d": {
      "name": "userId",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-76fb-92b1-a6d60e4c1e0a": {
      "name": "prefix",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-76c4-a401-a01c35df1cf6": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-7b77-91d6-1b6b3bab593e": {
      "name": "expiry",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d807-7772-927b-c938986ce9b6": {
      "name": "limits",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-79b1-add1-205c1961ec97"
      },
      "type": "object"
    },
    "019fba9e-d808-7356-868f-d2e1464f3b88": {
      "name": "last_used",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d809-706f-b554-c28833f0ae00": {
      "name": "ip",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d802-7e3c-a7cb-bae9eadbdf68"
      },
      "type": "object"
    },
    "019fba9e-d80a-7625-a1dd-bf7f2e963099": {
      "name": "environment",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80b-78e1-85f1-8e000a0ea5be": {
      "name": "status",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80c-70d1-b816-8c95944ec6ae": {
      "name": "usage",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80d-7856-868b-f6bcd7a4dfd9": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d803-78f8-9b3f-253c9c439754"
      },
      "type": "object"
    },
    "019fba9e-d80e-7a13-b92f-a9079d36ecc8": {
      "name": "key",
      "type": "string"
    }
  },
  "name": "APIKeyCreatedOutput",
  "schemas": {
    "019fba9e-d801-79b1-add1-205c1961ec97": {
      "fields": {
        "019fba9e-d801-7ae3-b9f4-9518a1c8e799": {
          "name": "rph",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d802-77be-b9fb-a8ffccf7aa81": {
          "name": "rpm",
          "nullable": true,
          "type": "integer"
        }
      },
      "name": "Limits"
    },
    "019fba9e-d802-7e3c-a7cb-bae9eadbdf68": {
      "fields": {
        "019fba9e-d801-788b-a8ac-e45afd4ef5ad": {
          "name": "whitelist",
          "schema": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "name": "IPConfig"
    },
    "019fba9e-d803-78f8-9b3f-253c9c439754": {
      "fields": {
        "019fba9e-d801-7764-b7d5-ece48e7541d7": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-7c0d-a8ae-60b6a8549596": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7e82-a0cd-b2b2eb5767cc": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7583-9d01-decdbf1e29a7": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-7890-9d15-d290019e1a29": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7f3b-903b-babe8783cd44": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemAPIKeyMetadata"
    }
  },
  "version": "1.0.0"
}
```

---
