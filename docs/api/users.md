# Users API

## user

### Create a new user

**`POST`** `/system/users/user/create`

Create a new user

- **Handler:** `system:users:user:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7ef2-8dac-08b21764bd18": {
      "name": "payload",
      "required": true,
      "schema": {
        "id": "019fba9e-d801-7917-8feb-557cca9207ce"
      },
      "type": "object"
    }
  },
  "name": "UserRegisterInput",
  "schemas": {
    "019fba9e-d801-7917-8feb-557cca9207ce": {
      "fields": {
        "019fba9e-d801-7f9a-a695-916df2020bdc": {
          "name": "permissions",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d802-7ca5-90e3-7ec2536de290": {
          "name": "email",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-7767-a4e9-7bdc312401e2": {
          "name": "name",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7860-a1ae-7313d6661e89": {
          "name": "password",
          "required": true,
          "type": "string"
        },
        "019fba9e-d805-7ae9-8408-1c0d164cf6c9": {
          "name": "data",
          "type": "record"
        },
        "019fba9e-d806-7550-b087-d83c13a71c90": {
          "name": "tenant_id",
          "nullable": true,
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
    "019fba9e-d801-7b73-9a49-da8c608cf40c": {
      "name": "permissions",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-705d-bf4f-82fcd6e5fd3d": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-795d-88f1-815a323adb8e": {
      "name": "email",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-73e4-aee1-2200c89568b7": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-7400-a34c-d732a98bd410": {
      "name": "password",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-773c-868e-20ebbf2d29ab": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7024-9b43-be87087f8458"
      },
      "type": "object"
    },
    "019fba9e-d807-7acb-ace1-be6053c77558": {
      "name": "data",
      "type": "record"
    },
    "019fba9e-d808-7416-b0e8-4b11cefacbb2": {
      "default": -1,
      "name": "disabled",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d809-7eb0-8079-a7e7d3926dad": {
      "name": "settings",
      "type": "record"
    },
    "019fba9e-d80a-7cbd-a7de-a608fc5bcaa5": {
      "name": "tenant_id",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80b-7709-9cea-b792246e513e": {
      "default": 0,
      "name": "token_version",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80c-712c-86f6-618bc764a67a": {
      "name": "verified",
      "nullable": true,
      "type": "boolean"
    }
  },
  "name": "SystemUser",
  "schemas": {
    "019fba9e-d801-7024-9b43-be87087f8458": {
      "fields": {
        "019fba9e-d801-7c3e-b48e-ef2fc634a322": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-706e-812e-0e5450007937": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-729f-85ab-28afa42d4136": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7db7-851a-c59a3a23ae8d": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-740c-b8cb-ade460764958": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7471-b9e6-15f9086c8316": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemUserMetadata"
    }
  },
  "version": "1.0.0"
}
```

---

### Get user by ID

**`GET`** `/system/users/user/get/{user_id}`

Get user by ID

- **Handler:** `system:users:user:get`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7379-a873-a7e23c26f810": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7a0b-a0c5-c859d325a43a"
      },
      "type": "object"
    }
  },
  "name": "UserGetInput",
  "schemas": {
    "019fba9e-d801-7a0b-a0c5-c859d325a43a": {
      "fields": {
        "019fba9e-d801-7e40-9ca4-968566ff9685": {
          "name": "user_id",
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
    "019fba9e-d801-7b73-9a49-da8c608cf40c": {
      "name": "permissions",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-705d-bf4f-82fcd6e5fd3d": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-795d-88f1-815a323adb8e": {
      "name": "email",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-73e4-aee1-2200c89568b7": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-7400-a34c-d732a98bd410": {
      "name": "password",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-773c-868e-20ebbf2d29ab": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7024-9b43-be87087f8458"
      },
      "type": "object"
    },
    "019fba9e-d807-7acb-ace1-be6053c77558": {
      "name": "data",
      "type": "record"
    },
    "019fba9e-d808-7416-b0e8-4b11cefacbb2": {
      "default": -1,
      "name": "disabled",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d809-7eb0-8079-a7e7d3926dad": {
      "name": "settings",
      "type": "record"
    },
    "019fba9e-d80a-7cbd-a7de-a608fc5bcaa5": {
      "name": "tenant_id",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80b-7709-9cea-b792246e513e": {
      "default": 0,
      "name": "token_version",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80c-712c-86f6-618bc764a67a": {
      "name": "verified",
      "nullable": true,
      "type": "boolean"
    }
  },
  "name": "SystemUser",
  "schemas": {
    "019fba9e-d801-7024-9b43-be87087f8458": {
      "fields": {
        "019fba9e-d801-7c3e-b48e-ef2fc634a322": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-706e-812e-0e5450007937": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-729f-85ab-28afa42d4136": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7db7-851a-c59a3a23ae8d": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-740c-b8cb-ade460764958": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7471-b9e6-15f9086c8316": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemUserMetadata"
    }
  },
  "version": "1.0.0"
}
```

---

### Update user profile

**`PATCH`** `/system/users/user/update/{user_id}`

Update user profile

- **Handler:** `system:users:user:update`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7582-9ec5-2b2cf6944d30": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-72a5-8568-2bdcd0ae22cb"
      },
      "type": "object"
    },
    "019fba9e-d802-7199-9b1f-89e5f8bc3291": {
      "name": "arguments",
      "required": true,
      "schema": {
        "id": "019fba9e-d802-7c57-b877-77012594cee0"
      },
      "type": "object"
    }
  },
  "name": "UserUpdateInput",
  "schemas": {
    "019fba9e-d801-72a5-8568-2bdcd0ae22cb": {
      "fields": {
        "019fba9e-d801-7617-aeb5-ebad258321bc": {
          "name": "permissions",
          "schema": {
            "type": "string"
          },
          "type": "array"
        },
        "019fba9e-d802-7df8-a78d-655f5198f166": {
          "name": "data",
          "type": "record"
        },
        "019fba9e-d803-7199-95c7-052359b0d4be": {
          "default": -1,
          "name": "disabled",
          "nullable": true,
          "type": "integer"
        },
        "019fba9e-d804-7b10-93c5-e58dbb85d376": {
          "name": "email",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-763e-9217-085629948e65": {
          "name": "name",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7ab8-b461-80766809734b": {
          "name": "settings",
          "type": "record"
        },
        "019fba9e-d807-70f0-b213-c89b2cac405c": {
          "name": "verified",
          "nullable": true,
          "type": "boolean"
        }
      },
      "name": "payload"
    },
    "019fba9e-d802-7c57-b877-77012594cee0": {
      "fields": {
        "019fba9e-d801-732a-955e-d3da1a9d8e78": {
          "name": "user_id",
          "required": true,
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
    "019fba9e-d801-7b73-9a49-da8c608cf40c": {
      "name": "permissions",
      "schema": {
        "type": "string"
      },
      "type": "array"
    },
    "019fba9e-d802-705d-bf4f-82fcd6e5fd3d": {
      "name": "_id_",
      "required": true,
      "type": "string"
    },
    "019fba9e-d803-795d-88f1-815a323adb8e": {
      "name": "email",
      "required": true,
      "type": "string"
    },
    "019fba9e-d804-73e4-aee1-2200c89568b7": {
      "name": "name",
      "required": true,
      "type": "string"
    },
    "019fba9e-d805-7400-a34c-d732a98bd410": {
      "name": "password",
      "required": true,
      "type": "string"
    },
    "019fba9e-d806-773c-868e-20ebbf2d29ab": {
      "name": "_metadata_",
      "nullable": true,
      "schema": {
        "id": "019fba9e-d801-7024-9b43-be87087f8458"
      },
      "type": "object"
    },
    "019fba9e-d807-7acb-ace1-be6053c77558": {
      "name": "data",
      "type": "record"
    },
    "019fba9e-d808-7416-b0e8-4b11cefacbb2": {
      "default": -1,
      "name": "disabled",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d809-7eb0-8079-a7e7d3926dad": {
      "name": "settings",
      "type": "record"
    },
    "019fba9e-d80a-7cbd-a7de-a608fc5bcaa5": {
      "name": "tenant_id",
      "nullable": true,
      "type": "string"
    },
    "019fba9e-d80b-7709-9cea-b792246e513e": {
      "default": 0,
      "name": "token_version",
      "nullable": true,
      "type": "integer"
    },
    "019fba9e-d80c-712c-86f6-618bc764a67a": {
      "name": "verified",
      "nullable": true,
      "type": "boolean"
    }
  },
  "name": "SystemUser",
  "schemas": {
    "019fba9e-d801-7024-9b43-be87087f8458": {
      "fields": {
        "019fba9e-d801-7c3e-b48e-ef2fc634a322": {
          "name": "checksum",
          "required": true,
          "type": "string"
        },
        "019fba9e-d802-706e-812e-0e5450007937": {
          "name": "created",
          "required": true,
          "type": "string"
        },
        "019fba9e-d803-729f-85ab-28afa42d4136": {
          "name": "updated",
          "required": true,
          "type": "string"
        },
        "019fba9e-d804-7db7-851a-c59a3a23ae8d": {
          "name": "signature",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d805-740c-b8cb-ade460764958": {
          "name": "trace_id",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d806-7471-b9e6-15f9086c8316": {
          "name": "version",
          "required": true,
          "type": "number"
        }
      },
      "name": "SystemUserMetadata"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete user account

**`DELETE`** `/system/users/user/delete/{user_id}`

Delete user account

- **Handler:** `system:users:user:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-75b7-9651-370ed0e1b4ec": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7c5d-b770-7db110088426"
      },
      "type": "object"
    }
  },
  "name": "UserDeleteInput",
  "schemas": {
    "019fba9e-d801-7c5d-b770-7db110088426": {
      "fields": {
        "019fba9e-d801-79c5-8d19-b3bca28954f8": {
          "name": "user_id",
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

## password

### Change account password

**`PATCH`** `/system/users/password/change/{user_id}`

Change account password

- **Handler:** `system:users:password:change`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-71ae-b73d-99d634601df7": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-75a7-bbcf-fd53e2f932ea"
      },
      "type": "object"
    },
    "019fba9e-d802-731e-933b-1cd62476ffcd": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d802-758f-a202-9cec2f60d82b"
      },
      "type": "object"
    }
  },
  "name": "UserChangePasswordInput",
  "schemas": {
    "019fba9e-d801-75a7-bbcf-fd53e2f932ea": {
      "fields": {
        "019fba9e-d801-7da1-a054-34da82dd4d19": {
          "name": "user_id",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-758f-a202-9cec2f60d82b": {
      "fields": {
        "019fba9e-d801-7eb8-abf3-4444a7321857": {
          "name": "current",
          "type": "string"
        },
        "019fba9e-d802-71af-9065-52432a2d6935": {
          "name": "new",
          "type": "string"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

---
