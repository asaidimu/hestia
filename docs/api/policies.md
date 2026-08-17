# Policies API

## binding

### Get binding info

**`GET`** `/system/policies/binding/get/{name}`

Get binding info

- **Handler:** `system:policies:binding:get`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-769d-8c49-5fb4ddc5329b": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7243-8b32-80758e3a5c26"
      },
      "type": "object"
    }
  },
  "name": "PolicyBindingGetInput",
  "schemas": {
    "019fba9e-d801-7243-8b32-80758e3a5c26": {
      "fields": {
        "019fba9e-d801-7ad5-8bc5-35b66fec3773": {
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
    "019fba9e-d801-7700-bf08-662859c995f6": {
      "name": "name",
      "type": "string"
    },
    "019fba9e-d802-77df-af62-12b81d6e27bc": {
      "name": "description",
      "type": "string"
    }
  },
  "name": "BindingView",
  "version": "1.0.0"
}
```

---

### List all bindings

**`GET`** `/system/policies/binding/list`

List all bindings

- **Handler:** `system:policies:binding:list`

#### Request Body

```json
{
  "name": "PolicyBindingListInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7df4-8e45-24af00be7f51": {
      "name": "bindings",
      "schema": {
        "id": "019fba9e-d801-79e2-a24a-d0c8e4172387"
      },
      "type": "array"
    }
  },
  "name": "BindingListDocument",
  "schemas": {
    "019fba9e-d801-79e2-a24a-d0c8e4172387": {
      "fields": {
        "019fba9e-d801-7e16-b182-3843a1558a2b": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d802-70a3-b331-d2a72341ee3f": {
          "name": "description",
          "type": "string"
        }
      },
      "name": "BindingView"
    }
  },
  "version": "1.0.0"
}
```

---

## rule

### Validate a CEL rule expression

**`POST`** `/system/policies/rule/validate`

Validate a CEL rule expression

- **Handler:** `system:policies:rule:validate`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7d3b-9728-edea0f445c3b": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "PolicyValidateInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-71af-8976-caedd2194093": {
      "name": "valid",
      "type": "boolean"
    },
    "019fba9e-d802-7b71-ad2a-6e853111a63a": {
      "name": "result",
      "type": "boolean"
    },
    "019fba9e-d803-7f28-992d-d5f66c2633f5": {
      "name": "error",
      "type": "string"
    }
  },
  "name": "PolicyValidateResult",
  "version": "1.0.0"
}
```

---

### List all rules

**`GET`** `/system/policies/rule/list`

List all rules

- **Handler:** `system:policies:rule:list`

#### Request Body

```json
{
  "name": "PolicyRuleListInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7495-8279-45e614d77b4c": {
      "name": "rules",
      "schema": {
        "id": "019fba9e-d801-7dc9-867f-b4c96b7556e4"
      },
      "type": "array"
    }
  },
  "name": "RuleListDocument",
  "schemas": {
    "019fba9e-d801-7dc9-867f-b4c96b7556e4": {
      "fields": {
        "019fba9e-d801-7c83-8b3b-33a9bf5c65f3": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-7b8e-9479-42787cb2ed16": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d803-7934-b792-5d51b4c283c7": {
          "name": "ruleType",
          "type": "string"
        },
        "019fba9e-d804-76db-ae38-5ee3e439dbd2": {
          "name": "syntax",
          "type": "string"
        },
        "019fba9e-d805-7dcc-abe1-a1a996bb6719": {
          "name": "expression",
          "type": "string"
        },
        "019fba9e-d806-7502-b7ac-960d56d839e9": {
          "name": "rules",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d802-7dd8-9e2c-794a67336148"
          },
          "type": "object"
        },
        "019fba9e-d807-7077-b47e-45ae7cf6408b": {
          "name": "description",
          "type": "string"
        },
        "019fba9e-d808-7a14-a3e5-054387e22743": {
          "name": "protected",
          "type": "boolean"
        }
      },
      "name": "PolicyRuleView"
    },
    "019fba9e-d802-7dd8-9e2c-794a67336148": {
      "fields": {
        "019fba9e-d801-7779-9b21-7d6eb7af1d55": {
          "name": "type",
          "type": "string"
        },
        "019fba9e-d802-7876-82e2-5538c0ca80e4": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d803-77fe-a2a2-5242dc6d4fb7": {
          "name": "expression",
          "type": "string"
        },
        "019fba9e-d804-7b11-8461-0e7b2c0b9964": {
          "name": "operator",
          "type": "string"
        },
        "019fba9e-d805-7c06-9a62-265c768a7397": {
          "name": "conditions",
          "schema": {
            "id": "019fba9e-d802-7dd8-9e2c-794a67336148"
          },
          "type": "array"
        }
      },
      "name": "RuleNodeView"
    }
  },
  "version": "1.0.0"
}
```

---

### Get a policy rule

**`GET`** `/system/policies/rule/get/{name}`

Get a policy rule

- **Handler:** `system:policies:rule:get`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7442-b7fb-259b8a06715f": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-76c6-9c9f-0754b326169e"
      },
      "type": "object"
    }
  },
  "name": "PolicyRuleGetInput",
  "schemas": {
    "019fba9e-d801-76c6-9c9f-0754b326169e": {
      "fields": {
        "019fba9e-d801-7ba2-9665-b82c8d396e46": {
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
    "019fba9e-d801-7527-a399-ba48de109889": {
      "name": "id",
      "type": "string"
    },
    "019fba9e-d802-765e-9a37-9bf80aea7954": {
      "name": "name",
      "type": "string"
    },
    "019fba9e-d803-7a28-8b74-14ad9f143066": {
      "name": "ruleType",
      "type": "string"
    },
    "019fba9e-d804-7ac4-ba54-676e389a88ae": {
      "name": "syntax",
      "type": "string"
    },
    "019fba9e-d805-7520-aa49-e769835e33f0": {
      "name": "expression",
      "type": "string"
    },
    "019fba9e-d806-7142-8629-f0f78b4cde98": {
      "name": "rules",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7dd8-9e2c-794a67336148"
      },
      "type": "object"
    },
    "019fba9e-d807-7497-870c-f5b6286a2cf9": {
      "name": "description",
      "type": "string"
    },
    "019fba9e-d808-7b73-a859-7880741b6085": {
      "name": "protected",
      "type": "boolean"
    }
  },
  "name": "PolicyRuleView",
  "schemas": {
    "019fba9e-d801-7dd8-9e2c-794a67336148": {
      "fields": {
        "019fba9e-d801-7b39-8c9c-bfde89a4ab17": {
          "name": "type",
          "type": "string"
        },
        "019fba9e-d802-71eb-b3b5-0d56e5617a52": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d803-7c15-882e-026582b54283": {
          "name": "expression",
          "type": "string"
        },
        "019fba9e-d804-743f-a02c-908cd7ed1633": {
          "name": "operator",
          "type": "string"
        },
        "019fba9e-d805-7e5a-8b91-81c9b2cd389a": {
          "name": "conditions",
          "schema": {
            "id": "019fba9e-d801-7dd8-9e2c-794a67336148"
          },
          "type": "array"
        }
      },
      "name": "RuleNodeView"
    }
  },
  "version": "1.0.0"
}
```

---

### Create a policy rule

**`POST`** `/system/policies/rule/create/{name}`

Create a policy rule

- **Handler:** `system:policies:rule:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7baf-9312-36a0331e2aac": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7838-80d7-801d25c402b8"
      },
      "type": "object"
    },
    "019fba9e-d802-794b-9fa9-ffba146f5de1": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "PolicyRuleCreateInput",
  "schemas": {
    "019fba9e-d801-7838-80d7-801d25c402b8": {
      "fields": {
        "019fba9e-d801-7521-808b-1c3dc9d0fda5": {
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
    "019fba9e-d801-7527-a399-ba48de109889": {
      "name": "id",
      "type": "string"
    },
    "019fba9e-d802-765e-9a37-9bf80aea7954": {
      "name": "name",
      "type": "string"
    },
    "019fba9e-d803-7a28-8b74-14ad9f143066": {
      "name": "ruleType",
      "type": "string"
    },
    "019fba9e-d804-7ac4-ba54-676e389a88ae": {
      "name": "syntax",
      "type": "string"
    },
    "019fba9e-d805-7520-aa49-e769835e33f0": {
      "name": "expression",
      "type": "string"
    },
    "019fba9e-d806-7142-8629-f0f78b4cde98": {
      "name": "rules",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7dd8-9e2c-794a67336148"
      },
      "type": "object"
    },
    "019fba9e-d807-7497-870c-f5b6286a2cf9": {
      "name": "description",
      "type": "string"
    },
    "019fba9e-d808-7b73-a859-7880741b6085": {
      "name": "protected",
      "type": "boolean"
    }
  },
  "name": "PolicyRuleView",
  "schemas": {
    "019fba9e-d801-7dd8-9e2c-794a67336148": {
      "fields": {
        "019fba9e-d801-7b39-8c9c-bfde89a4ab17": {
          "name": "type",
          "type": "string"
        },
        "019fba9e-d802-71eb-b3b5-0d56e5617a52": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d803-7c15-882e-026582b54283": {
          "name": "expression",
          "type": "string"
        },
        "019fba9e-d804-743f-a02c-908cd7ed1633": {
          "name": "operator",
          "type": "string"
        },
        "019fba9e-d805-7e5a-8b91-81c9b2cd389a": {
          "name": "conditions",
          "schema": {
            "id": "019fba9e-d801-7dd8-9e2c-794a67336148"
          },
          "type": "array"
        }
      },
      "name": "RuleNodeView"
    }
  },
  "version": "1.0.0"
}
```

---

### Update a policy rule

**`PATCH`** `/system/policies/rule/update/{name}`

Update a policy rule

- **Handler:** `system:policies:rule:update`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-77c3-8116-c0aa6ad451e8": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-739c-8720-8cafca10a8cf"
      },
      "type": "object"
    },
    "019fba9e-d802-7ccb-acae-aef9accc5e4a": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "PolicyRuleUpdateInput",
  "schemas": {
    "019fba9e-d801-739c-8720-8cafca10a8cf": {
      "fields": {
        "019fba9e-d801-76a3-9bbc-5529834b248f": {
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
    "019fba9e-d801-7527-a399-ba48de109889": {
      "name": "id",
      "type": "string"
    },
    "019fba9e-d802-765e-9a37-9bf80aea7954": {
      "name": "name",
      "type": "string"
    },
    "019fba9e-d803-7a28-8b74-14ad9f143066": {
      "name": "ruleType",
      "type": "string"
    },
    "019fba9e-d804-7ac4-ba54-676e389a88ae": {
      "name": "syntax",
      "type": "string"
    },
    "019fba9e-d805-7520-aa49-e769835e33f0": {
      "name": "expression",
      "type": "string"
    },
    "019fba9e-d806-7142-8629-f0f78b4cde98": {
      "name": "rules",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7dd8-9e2c-794a67336148"
      },
      "type": "object"
    },
    "019fba9e-d807-7497-870c-f5b6286a2cf9": {
      "name": "description",
      "type": "string"
    },
    "019fba9e-d808-7b73-a859-7880741b6085": {
      "name": "protected",
      "type": "boolean"
    }
  },
  "name": "PolicyRuleView",
  "schemas": {
    "019fba9e-d801-7dd8-9e2c-794a67336148": {
      "fields": {
        "019fba9e-d801-7b39-8c9c-bfde89a4ab17": {
          "name": "type",
          "type": "string"
        },
        "019fba9e-d802-71eb-b3b5-0d56e5617a52": {
          "name": "name",
          "type": "string"
        },
        "019fba9e-d803-7c15-882e-026582b54283": {
          "name": "expression",
          "type": "string"
        },
        "019fba9e-d804-743f-a02c-908cd7ed1633": {
          "name": "operator",
          "type": "string"
        },
        "019fba9e-d805-7e5a-8b91-81c9b2cd389a": {
          "name": "conditions",
          "schema": {
            "id": "019fba9e-d801-7dd8-9e2c-794a67336148"
          },
          "type": "array"
        }
      },
      "name": "RuleNodeView"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete a policy rule

**`DELETE`** `/system/policies/rule/delete/{name}`

Delete a policy rule

- **Handler:** `system:policies:rule:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7ee8-bd0a-fc1f990ec3cf": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7fa5-92e8-b1f47b12bf0f"
      },
      "type": "object"
    }
  },
  "name": "PolicyRuleDeleteInput",
  "schemas": {
    "019fba9e-d801-7fa5-92e8-b1f47b12bf0f": {
      "fields": {
        "019fba9e-d801-7d76-bc1f-b9a9da88a48a": {
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
    "019fba9e-d801-7b55-a5d1-8db301af3de7": {
      "name": "message",
      "type": "string"
    },
    "019fba9e-d802-7429-b377-2745113922ee": {
      "name": "name",
      "type": "string"
    }
  },
  "name": "RuleDeletedResult",
  "version": "1.0.0"
}
```

---

## reload

### Reload policies from database

**`GET`** `/system/policies/reload`

Reload policies from database

- **Handler:** `system:policies:reload`

#### Request Body

```json
{
  "name": "PolicyReloadInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-77a0-a6f4-d685097eb3a5": {
      "name": "operations",
      "type": "integer"
    },
    "019fba9e-d802-72ce-b9e4-a498218abb15": {
      "name": "rules",
      "type": "integer"
    }
  },
  "name": "PolicyReloadResult",
  "version": "1.0.0"
}
```

---

## policy

### Create a policy binding

**`POST`** `/system/policies/policy/create/{name}`

Create a policy binding

- **Handler:** `system:policies:policy:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7006-92fd-72cad9072621": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-79b7-9282-016703df09e9"
      },
      "type": "object"
    },
    "019fba9e-d802-7d95-98d8-ebd98e0ec2b6": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "PolicyCreateInput",
  "schemas": {
    "019fba9e-d801-79b7-9282-016703df09e9": {
      "fields": {
        "019fba9e-d801-764d-a8b3-98d80fb959ca": {
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
    "019fba9e-d801-7395-b0d0-20ef64e82b6f": {
      "name": "id",
      "type": "string"
    },
    "019fba9e-d802-7347-880e-ab62c0ef5b60": {
      "name": "operation",
      "type": "string"
    },
    "019fba9e-d803-7515-8835-d9c6ef02a5fe": {
      "name": "rule",
      "type": "string"
    },
    "019fba9e-d804-720e-b1ee-7e5d26ea4729": {
      "name": "enabled",
      "type": "boolean"
    },
    "019fba9e-d805-7458-9150-9e7b8d381056": {
      "name": "rateLimit",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-794e-8542-26456f1f284e"
      },
      "type": "object"
    },
    "019fba9e-d806-7a9b-8f56-1c9ac4a550d1": {
      "name": "throttle",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d802-7013-8e6c-e401be47c313"
      },
      "type": "object"
    }
  },
  "name": "PolicyView",
  "schemas": {
    "019fba9e-d801-794e-8542-26456f1f284e": {
      "fields": {
        "019fba9e-d801-7b35-99d8-3f63c8f99eb7": {
          "name": "enabled",
          "type": "boolean"
        },
        "019fba9e-d802-7e33-aec9-97b6eb06a49a": {
          "name": "identity",
          "type": "string"
        },
        "019fba9e-d803-7a08-83e2-8ff5d4357dc5": {
          "name": "capacity",
          "type": "integer"
        },
        "019fba9e-d804-72ca-a50e-5b2e31589d37": {
          "name": "refill",
          "type": "integer"
        },
        "019fba9e-d805-7c98-af5e-c102919eab29": {
          "name": "period",
          "type": "integer"
        }
      },
      "name": "RateLimitView"
    },
    "019fba9e-d802-7013-8e6c-e401be47c313": {
      "fields": {
        "019fba9e-d801-71a4-87e2-9d4bc55c70fd": {
          "name": "limit",
          "type": "integer"
        },
        "019fba9e-d802-7f12-b8b6-dc204162dde0": {
          "name": "window",
          "type": "integer"
        },
        "019fba9e-d803-77e0-bf3e-04ca46297d53": {
          "name": "action",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d803-709c-9d48-83f4ed3a7769"
          },
          "type": "object"
        }
      },
      "name": "ThrottleView"
    },
    "019fba9e-d803-709c-9d48-83f4ed3a7769": {
      "fields": {
        "019fba9e-d801-7aab-9b88-c8e8cdaf352a": {
          "name": "message",
          "type": "string"
        },
        "019fba9e-d802-74de-a72e-1b7187b711cc": {
          "name": "input",
          "type": "record"
        }
      },
      "name": "ThrottleActionView"
    }
  },
  "version": "1.0.0"
}
```

---

### Update a policy — set rule, enabled, or both

**`PATCH`** `/system/policies/policy/update/{name}`

Update a policy — set rule, enabled, or both

- **Handler:** `system:policies:policy:update`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7a33-8059-ce1c667d5f70": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7439-a063-3370129f82e0"
      },
      "type": "object"
    },
    "019fba9e-d802-7152-893b-8b6c46b1ecf1": {
      "name": "payload",
      "type": "record"
    }
  },
  "name": "PolicyUpdateInput",
  "schemas": {
    "019fba9e-d801-7439-a063-3370129f82e0": {
      "fields": {
        "019fba9e-d801-742c-babb-449b29f80f43": {
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
    "019fba9e-d801-7395-b0d0-20ef64e82b6f": {
      "name": "id",
      "type": "string"
    },
    "019fba9e-d802-7347-880e-ab62c0ef5b60": {
      "name": "operation",
      "type": "string"
    },
    "019fba9e-d803-7515-8835-d9c6ef02a5fe": {
      "name": "rule",
      "type": "string"
    },
    "019fba9e-d804-720e-b1ee-7e5d26ea4729": {
      "name": "enabled",
      "type": "boolean"
    },
    "019fba9e-d805-7458-9150-9e7b8d381056": {
      "name": "rateLimit",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-794e-8542-26456f1f284e"
      },
      "type": "object"
    },
    "019fba9e-d806-7a9b-8f56-1c9ac4a550d1": {
      "name": "throttle",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d802-7013-8e6c-e401be47c313"
      },
      "type": "object"
    }
  },
  "name": "PolicyView",
  "schemas": {
    "019fba9e-d801-794e-8542-26456f1f284e": {
      "fields": {
        "019fba9e-d801-7b35-99d8-3f63c8f99eb7": {
          "name": "enabled",
          "type": "boolean"
        },
        "019fba9e-d802-7e33-aec9-97b6eb06a49a": {
          "name": "identity",
          "type": "string"
        },
        "019fba9e-d803-7a08-83e2-8ff5d4357dc5": {
          "name": "capacity",
          "type": "integer"
        },
        "019fba9e-d804-72ca-a50e-5b2e31589d37": {
          "name": "refill",
          "type": "integer"
        },
        "019fba9e-d805-7c98-af5e-c102919eab29": {
          "name": "period",
          "type": "integer"
        }
      },
      "name": "RateLimitView"
    },
    "019fba9e-d802-7013-8e6c-e401be47c313": {
      "fields": {
        "019fba9e-d801-71a4-87e2-9d4bc55c70fd": {
          "name": "limit",
          "type": "integer"
        },
        "019fba9e-d802-7f12-b8b6-dc204162dde0": {
          "name": "window",
          "type": "integer"
        },
        "019fba9e-d803-77e0-bf3e-04ca46297d53": {
          "name": "action",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d803-709c-9d48-83f4ed3a7769"
          },
          "type": "object"
        }
      },
      "name": "ThrottleView"
    },
    "019fba9e-d803-709c-9d48-83f4ed3a7769": {
      "fields": {
        "019fba9e-d801-7aab-9b88-c8e8cdaf352a": {
          "name": "message",
          "type": "string"
        },
        "019fba9e-d802-74de-a72e-1b7187b711cc": {
          "name": "input",
          "type": "record"
        }
      },
      "name": "ThrottleActionView"
    }
  },
  "version": "1.0.0"
}
```

---

### List all policy bindings

**`GET`** `/system/policies/policy/list`

List all policy bindings

- **Handler:** `system:policies:policy:list`

#### Request Body

```json
{
  "name": "PolicyListInput",
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7272-be41-f254f5b0f329": {
      "name": "policies",
      "schema": {
        "id": "019fba9e-d801-7f89-9da3-83d3a032d16d"
      },
      "type": "array"
    }
  },
  "name": "PolicyListDocument",
  "schemas": {
    "019fba9e-d801-7f89-9da3-83d3a032d16d": {
      "fields": {
        "019fba9e-d801-7a74-ae45-d5a85fcc6496": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-717b-af60-8a744c443ab1": {
          "name": "operation",
          "type": "string"
        },
        "019fba9e-d803-7a52-b29e-264697a452b3": {
          "name": "rule",
          "type": "string"
        },
        "019fba9e-d804-78b9-9e9f-1734c3603e70": {
          "name": "enabled",
          "type": "boolean"
        },
        "019fba9e-d805-7d7d-a931-15e414a9dc30": {
          "name": "rateLimit",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d802-794e-8542-26456f1f284e"
          },
          "type": "object"
        },
        "019fba9e-d806-7033-86a5-bb2a171af386": {
          "name": "throttle",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d803-7013-8e6c-e401be47c313"
          },
          "type": "object"
        }
      },
      "name": "PolicyView"
    },
    "019fba9e-d802-794e-8542-26456f1f284e": {
      "fields": {
        "019fba9e-d801-7fbf-9907-844f1a25b84c": {
          "name": "enabled",
          "type": "boolean"
        },
        "019fba9e-d802-72bd-a25e-08b402824f75": {
          "name": "identity",
          "type": "string"
        },
        "019fba9e-d803-7753-a2a2-3ade92f0c51b": {
          "name": "capacity",
          "type": "integer"
        },
        "019fba9e-d804-7e78-9cb8-10fb5f1ae375": {
          "name": "refill",
          "type": "integer"
        },
        "019fba9e-d805-7d6e-91e8-b197a6260cba": {
          "name": "period",
          "type": "integer"
        }
      },
      "name": "RateLimitView"
    },
    "019fba9e-d803-7013-8e6c-e401be47c313": {
      "fields": {
        "019fba9e-d801-78fa-bf2c-3715f8779457": {
          "name": "limit",
          "type": "integer"
        },
        "019fba9e-d802-7524-b198-61358ea056f2": {
          "name": "window",
          "type": "integer"
        },
        "019fba9e-d803-7681-9f48-49614328e5e5": {
          "name": "action",
          "nullable": true,
          "schema": {
            "id": "019fba9e-d804-709c-9d48-83f4ed3a7769"
          },
          "type": "object"
        }
      },
      "name": "ThrottleView"
    },
    "019fba9e-d804-709c-9d48-83f4ed3a7769": {
      "fields": {
        "019fba9e-d801-7c85-b138-d19bc57ccf90": {
          "name": "message",
          "type": "string"
        },
        "019fba9e-d802-7598-9e93-d25cbe4ff608": {
          "name": "input",
          "type": "record"
        }
      },
      "name": "ThrottleActionView"
    }
  },
  "version": "1.0.0"
}
```

---
