# Policies API

## operation

### Get policy operation

**`GET`** `/system/policies/operation/{name}`

Get policy operation

- **Handler:** `system:policies:operation:get`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_operation_get_input",
  "description": "Policy operation name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "policy_name_arguments"
      }
    }
  },
  "schemas": {
    "policy_name_arguments": {
      "name": "PolicyNameArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
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
  "name": "policy_operation_output",
  "description": "Policy operation with intents and rule references",
  "fields": {
    "document": {
      "name": "document",
      "description": "Policy operation document",
      "type": "object",
      "schema": {
        "id": "policy_operation"
      }
    }
  },
  "schemas": {
    "policy_operation": {
      "name": "PolicyOperation",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "intent_type": {
          "name": "intent_type",
          "description": "Intent type for this operation",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        },
        "rule_key": {
          "name": "rule_key",
          "description": "Associated rule key",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Create or update policy operation

**`PATCH`** `/system/policies/operation/{name}`

Create or update policy operation

- **Handler:** `system:policies:operation:upsert`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_operation_upsert_input",
  "description": "Create or update a policy operation",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "policy_name_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "operation_upsert_payload"
      }
    }
  },
  "schemas": {
    "operation_upsert_payload": {
      "name": "OperationUpsertPayload",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "intentType": {
          "name": "intentType",
          "description": "Intent type for this operation",
          "type": "string"
        },
        "ruleKey": {
          "name": "ruleKey",
          "description": "Associated rule key",
          "type": "string"
        }
      }
    },
    "policy_name_arguments": {
      "name": "PolicyNameArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
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
  "name": "policy_operation_output",
  "description": "Policy operation with intents and rule references",
  "fields": {
    "document": {
      "name": "document",
      "description": "Policy operation document",
      "type": "object",
      "schema": {
        "id": "policy_operation"
      }
    }
  },
  "schemas": {
    "policy_operation": {
      "name": "PolicyOperation",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "intent_type": {
          "name": "intent_type",
          "description": "Intent type for this operation",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        },
        "rule_key": {
          "name": "rule_key",
          "description": "Associated rule key",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Delete policy operation

**`DELETE`** `/system/policies/operation/{name}`

Delete policy operation

- **Handler:** `system:policies:operation:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_operation_delete_input",
  "description": "Policy operation name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "policy_name_arguments"
      }
    }
  },
  "schemas": {
    "policy_name_arguments": {
      "name": "PolicyNameArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        }
      }
    }
  }
}
```

---

## rule

### Get policy rule

**`GET`** `/system/policies/rule/{name}`

Get policy rule

- **Handler:** `system:policies:rule:get`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_rule_get_input",
  "description": "Policy rule name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "policy_name_arguments"
      }
    }
  },
  "schemas": {
    "policy_name_arguments": {
      "name": "PolicyNameArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
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
  "name": "policy_rule_output",
  "description": "Policy rule with expression details",
  "fields": {
    "document": {
      "name": "document",
      "description": "Policy rule document",
      "type": "object",
      "schema": {
        "id": "policy_rule"
      }
    }
  },
  "schemas": {
    "policy_rule": {
      "name": "PolicyRule",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "expression": {
          "name": "expression",
          "description": "Rule expression",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        },
        "protected": {
          "name": "protected",
          "description": "Whether the rule is protected",
          "type": "boolean"
        },
        "rule_key": {
          "name": "rule_key",
          "description": "Rule key",
          "type": "string"
        },
        "rule_type": {
          "name": "rule_type",
          "description": "Type of rule (allow/deny)",
          "type": "string"
        },
        "rules": {
          "name": "rules",
          "description": "Nested rule nodes for composite rules",
          "type": "object",
          "schema": {
            "id": "rule_node"
          }
        },
        "syntax": {
          "name": "syntax",
          "description": "Rule expression syntax",
          "type": "string"
        }
      }
    },
    "rule_node": {
      "name": "RuleNode",
      "fields": {
        "expression": {
          "name": "expression",
          "description": "Leaf rule expression",
          "type": "string"
        },
        "operator": {
          "name": "operator",
          "description": "Logical operator (and/or/not)",
          "type": "string"
        },
        "rules": {
          "name": "rules",
          "description": "Array of nested rule nodes",
          "type": "array",
          "schema": {
            "id": "rule_node"
          }
        },
        "syntax": {
          "name": "syntax",
          "description": "Leaf rule syntax",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Validate CEL rule expression

**`POST`** `/system/policies/rule/query`

Validate CEL rule expression

- **Handler:** `system:policies:rule:validate`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_validate_input",
  "description": "Policy validation request with operation and context",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "Policy name argument",
      "type": "object",
      "schema": {
        "id": "policy_validate_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Validation context",
      "type": "object",
      "schema": {
        "id": "policy_validate_payload"
      }
    }
  },
  "schemas": {
    "policy_validate_arguments": {
      "name": "PolicyValidateArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        }
      }
    },
    "policy_validate_payload": {
      "name": "PolicyValidatePayload",
      "fields": {
        "context": {
          "name": "context",
          "description": "Validation context data",
          "type": "record"
        },
        "operation": {
          "name": "operation",
          "description": "Operation to validate",
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
  "name": "policy_validate_output",
  "description": "Policy validation result",
  "fields": {
    "document": {
      "name": "document",
      "description": "Validation result document",
      "type": "object",
      "schema": {
        "id": "policy_validate_result"
      }
    }
  },
  "schemas": {
    "policy_validate_result": {
      "name": "PolicyValidateResult",
      "fields": {
        "result": {
          "name": "result",
          "description": "Validation result detail",
          "type": "string"
        },
        "valid": {
          "name": "valid",
          "description": "Whether the request is permitted",
          "type": "boolean"
        }
      }
    }
  }
}
```

---

### Create or update policy rule

**`PATCH`** `/system/policies/rule/{name}`

Create or update policy rule

- **Handler:** `system:policies:rule:upsert`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_rule_upsert_input",
  "description": "Create or update a policy rule",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "policy_name_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "rule_upsert_payload"
      }
    }
  },
  "schemas": {
    "policy_name_arguments": {
      "name": "PolicyNameArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        }
      }
    },
    "rule_upsert_payload": {
      "name": "RuleUpsertPayload",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "expression": {
          "name": "expression",
          "description": "Rule expression",
          "type": "string"
        },
        "protected": {
          "name": "protected",
          "description": "Whether the rule is protected",
          "type": "boolean"
        },
        "ruleType": {
          "name": "ruleType",
          "description": "Type of rule (allow/deny)",
          "type": "string"
        },
        "rules": {
          "name": "rules",
          "description": "Nested rule nodes (JSON)",
          "type": "record"
        },
        "syntax": {
          "name": "syntax",
          "description": "Rule expression syntax",
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
  "name": "policy_rule_output",
  "description": "Policy rule with expression details",
  "fields": {
    "document": {
      "name": "document",
      "description": "Policy rule document",
      "type": "object",
      "schema": {
        "id": "policy_rule"
      }
    }
  },
  "schemas": {
    "policy_rule": {
      "name": "PolicyRule",
      "fields": {
        "description": {
          "name": "description",
          "description": "Human-readable description",
          "type": "string"
        },
        "expression": {
          "name": "expression",
          "description": "Rule expression",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        },
        "protected": {
          "name": "protected",
          "description": "Whether the rule is protected",
          "type": "boolean"
        },
        "rule_key": {
          "name": "rule_key",
          "description": "Rule key",
          "type": "string"
        },
        "rule_type": {
          "name": "rule_type",
          "description": "Type of rule (allow/deny)",
          "type": "string"
        },
        "rules": {
          "name": "rules",
          "description": "Nested rule nodes for composite rules",
          "type": "object",
          "schema": {
            "id": "rule_node"
          }
        },
        "syntax": {
          "name": "syntax",
          "description": "Rule expression syntax",
          "type": "string"
        }
      }
    },
    "rule_node": {
      "name": "RuleNode",
      "fields": {
        "expression": {
          "name": "expression",
          "description": "Leaf rule expression",
          "type": "string"
        },
        "operator": {
          "name": "operator",
          "description": "Logical operator (and/or/not)",
          "type": "string"
        },
        "rules": {
          "name": "rules",
          "description": "Array of nested rule nodes",
          "type": "array",
          "schema": {
            "id": "rule_node"
          }
        },
        "syntax": {
          "name": "syntax",
          "description": "Leaf rule syntax",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Delete policy rule

**`DELETE`** `/system/policies/rule/{name}`

Delete policy rule

- **Handler:** `system:policies:rule:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "policy_rule_delete_input",
  "description": "Policy rule name from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "policy_name_arguments"
      }
    }
  },
  "schemas": {
    "policy_name_arguments": {
      "name": "PolicyNameArguments",
      "fields": {
        "name": {
          "name": "name",
          "description": "Policy name",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Reload policies from database

**`GET`** `/system/policies/rule`

Reload policies from database

- **Handler:** `system:policies:rule:reload`

#### Response

```json
{
  "version": "1.0.0",
  "name": "policy_reload_output",
  "description": "Reload result with operation and rule counts",
  "fields": {
    "document": {
      "name": "document",
      "description": "Reload result document",
      "type": "object",
      "schema": {
        "id": "policy_reload_result"
      }
    }
  },
  "schemas": {
    "policy_reload_result": {
      "name": "PolicyReloadResult",
      "fields": {
        "operations": {
          "name": "operations",
          "description": "Number of operations loaded",
          "type": "integer"
        },
        "rules": {
          "name": "rules",
          "description": "Number of rules loaded",
          "type": "integer"
        }
      }
    }
  }
}
```

---
