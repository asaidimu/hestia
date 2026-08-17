# Core API

## heartbeat

### Session keepalive — does not count as a health check

**`GET`** `/system/core/heartbeat`

Session keepalive — does not count as a health check

- **Handler:** `system:core:heartbeat`
- **Bootstrap-safe:** Yes

---

## health

### Check system health and bootstrap status

**`GET`** `/system/core/health/check`

Check system health and bootstrap status

- **Handler:** `system:core:health:check`
- **Bootstrap-safe:** Yes

#### Response

```json
{
  "fields": {
    "019fba9e-d801-72ea-81b7-b557a2d3af50": {
      "name": "ok",
      "type": "boolean"
    },
    "019fba9e-d802-7879-8a1a-051367d44466": {
      "name": "bootstrapped",
      "type": "boolean"
    }
  },
  "name": "HealthView",
  "version": "1.0.0"
}
```

---

## capability

### List all registered commands and queries with descriptions and enabled status

**`GET`** `/system/core/capability/list`

List all registered commands and queries with descriptions and enabled status

- **Handler:** `system:core:capability:list`

#### Response

```json
{
  "fields": {
    "019fba9e-d801-785b-8c10-14678cbe298b": {
      "name": "capabilities",
      "schema": {
        "id": "019fba9e-d801-73e3-8e89-76acb2d6214b"
      },
      "type": "array"
    }
  },
  "name": "CapabilitiesDocument",
  "schemas": {
    "019fba9e-d801-73e3-8e89-76acb2d6214b": {
      "fields": {
        "019fba9e-d801-79ad-a879-a2f66a95b03e": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-7c62-9eeb-2a3bc8255e25": {
          "name": "intent_type",
          "type": "string"
        },
        "019fba9e-d803-7bf6-8d2e-3eb4829e1182": {
          "name": "description",
          "type": "string"
        },
        "019fba9e-d804-71ae-ba34-f2c9935bbb0e": {
          "name": "enabled",
          "type": "boolean"
        },
        "019fba9e-d805-7e7a-b5b8-7c4b2cf0ec00": {
          "name": "bootstrap_safe",
          "type": "boolean"
        }
      },
      "name": "CapabilityItem"
    }
  },
  "version": "1.0.0"
}
```

---

### Enable or disable a registered command or query

**`PATCH`** `/system/core/capability/set/{name}`

Enable or disable a registered command or query

- **Handler:** `system:core:capability:set`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7dae-bebe-f791fd5559dd": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7bca-882c-ce0a099bff3d"
      },
      "type": "object"
    },
    "019fba9e-d802-7cb8-bae5-85d5636783f0": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d802-7b25-9af3-c57beba07969"
      },
      "type": "object"
    }
  },
  "name": "CapabilityNameInput",
  "schemas": {
    "019fba9e-d801-7bca-882c-ce0a099bff3d": {
      "fields": {
        "019fba9e-d801-71e0-82bd-7ec2ed417cb6": {
          "name": "name",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7b25-9af3-c57beba07969": {
      "fields": {
        "019fba9e-d801-7d59-adea-8707fcb6ae87": {
          "name": "enabled",
          "type": "boolean"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

---

## audit

### Log an API access entry

**`POST`** `/system/core/audit/log`

Log an API access entry

- **Handler:** `system:core:audit:log`
- **Internal:** Yes

---

## docs

### List all registered API endpoints with metadata

**`GET`** `/system/core/docs/list`

List all registered API endpoints with metadata

- **Handler:** `system:core:docs:list`
- **Bootstrap-safe:** Yes

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

## bootstrap

### Mark system as bootstrapped

**`POST`** `/system/core/bootstrap/mark`

Mark system as bootstrapped

- **Handler:** `system:core:bootstrap:mark`
- **Bootstrap-safe:** Yes
- **Internal:** Yes

---

## reset

### Reset system to initial state

**`GET`** `/system/core/reset`

Reset system to initial state

- **Handler:** `system:core:reset`

---
