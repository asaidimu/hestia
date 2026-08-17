# Auth API

## session

### Authenticate and receive a session token

**`POST`** `/system/auth/session/create`

Authenticate and receive a session token

- **Handler:** `system:auth:session:create`
- **Bootstrap-safe:** Yes

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7a93-a95e-d13ae523fe9d": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-7efe-8371-9ef69a8bd03c"
      },
      "type": "object"
    }
  },
  "name": "LoginInput",
  "schemas": {
    "019fba9e-d801-7efe-8371-9ef69a8bd03c": {
      "fields": {
        "019fba9e-d801-78bc-b8c9-2d0d50a81572": {
          "name": "email",
          "type": "string"
        },
        "019fba9e-d802-7eb1-9768-252807e42a5b": {
          "name": "password",
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
    "019fba9e-d801-751c-beaa-13ba56df166a": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-740a-aae8-e722bf05ca9a"
      },
      "type": "object"
    }
  },
  "name": "LoginOutput",
  "schemas": {
    "019fba9e-d801-740a-aae8-e722bf05ca9a": {
      "fields": {
        "019fba9e-d801-79bb-a148-a398cb2cd906": {
          "name": "token",
          "type": "string"
        },
        "019fba9e-d802-75db-a5bc-81bae5df4ef0": {
          "name": "user",
          "type": "record"
        }
      },
      "name": "LoginDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---

### Logout

**`DELETE`** `/system/auth/session/delete`

Logout

- **Handler:** `system:auth:session:delete`
- **Bootstrap-safe:** Yes

#### Request Body

```json
{
  "name": "DeleteSessionInput",
  "version": "1.0.0"
}
```

---

### Validate a session token

**`GET`** `/system/auth/session/validate`

Validate a session token

- **Handler:** `system:auth:session:validate`
- **Internal:** Yes

#### Response

```json
{
  "fields": {
    "019fba9e-d801-77d5-b24d-88f0c2c151ed": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-7224-9c13-8fac38791dbe"
      },
      "type": "object"
    }
  },
  "name": "ClaimsOutput",
  "schemas": {
    "019fba9e-d801-7224-9c13-8fac38791dbe": {
      "fields": {
        "019fba9e-d801-73da-a0fa-315a3e13b64c": {
          "name": "user_id",
          "type": "string"
        },
        "019fba9e-d802-7033-8709-835d6ba81c19": {
          "name": "session_id",
          "type": "string"
        },
        "019fba9e-d803-7a4d-9cd1-7b3b8a608f7b": {
          "name": "issued_at",
          "type": "integer"
        },
        "019fba9e-d804-7c77-ad70-2ab3df763b40": {
          "name": "expires_at",
          "type": "integer"
        },
        "019fba9e-d805-74d5-bc9f-5fbfd67c3596": {
          "name": "created_at",
          "type": "integer"
        }
      },
      "name": "ClaimsDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---

## password

### Request password reset email

**`POST`** `/system/auth/password/reset`

Request password reset email

- **Handler:** `system:auth:password:reset`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7a72-91c5-a1009408610d": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-724d-b83b-992e799e1eb2"
      },
      "type": "object"
    }
  },
  "name": "PasswordResetInput",
  "schemas": {
    "019fba9e-d801-724d-b83b-992e799e1eb2": {
      "fields": {
        "019fba9e-d801-7790-8c10-af01000d7364": {
          "name": "email",
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
    "019fba9e-d801-774e-9b13-f3fb3563685c": {
      "name": "message",
      "type": "string"
    }
  },
  "name": "MessageOutput",
  "version": "1.0.0"
}
```

---

### Confirm password reset with token

**`PATCH`** `/system/auth/password/confirm`

Confirm password reset with token

- **Handler:** `system:auth:password:confirm`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-70f0-8af5-45599beb01dd": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-7201-96fb-cf7c6e0ef2aa"
      },
      "type": "object"
    }
  },
  "name": "PasswordConfirmInput",
  "schemas": {
    "019fba9e-d801-7201-96fb-cf7c6e0ef2aa": {
      "fields": {
        "019fba9e-d801-788a-b400-21038863f10b": {
          "name": "token",
          "type": "string"
        },
        "019fba9e-d802-74b5-b47f-86c33fc4705d": {
          "name": "password",
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
    "019fba9e-d801-774e-9b13-f3fb3563685c": {
      "name": "message",
      "type": "string"
    }
  },
  "name": "MessageOutput",
  "version": "1.0.0"
}
```

---

## apikey

### Validate an API key

**`GET`** `/system/auth/apikey/validate`

Validate an API key

- **Handler:** `system:auth:apikey:validate`
- **Internal:** Yes

#### Response

```json
{
  "fields": {
    "019fba9e-d801-77d5-b24d-88f0c2c151ed": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-7224-9c13-8fac38791dbe"
      },
      "type": "object"
    }
  },
  "name": "ClaimsOutput",
  "schemas": {
    "019fba9e-d801-7224-9c13-8fac38791dbe": {
      "fields": {
        "019fba9e-d801-73da-a0fa-315a3e13b64c": {
          "name": "user_id",
          "type": "string"
        },
        "019fba9e-d802-7033-8709-835d6ba81c19": {
          "name": "session_id",
          "type": "string"
        },
        "019fba9e-d803-7a4d-9cd1-7b3b8a608f7b": {
          "name": "issued_at",
          "type": "integer"
        },
        "019fba9e-d804-7c77-ad70-2ab3df763b40": {
          "name": "expires_at",
          "type": "integer"
        },
        "019fba9e-d805-74d5-bc9f-5fbfd67c3596": {
          "name": "created_at",
          "type": "integer"
        }
      },
      "name": "ClaimsDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---

## bootstrap

### Set bootstrap admin password

**`PATCH`** `/system/auth/bootstrap/password/set`

Set bootstrap admin password

- **Handler:** `system:auth:bootstrap:password:set`
- **Bootstrap-safe:** Yes

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7062-82e3-387dc3f59bb3": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-7468-a9fe-a0f989eeacf4"
      },
      "type": "object"
    }
  },
  "name": "BootstrapPasswordInput",
  "schemas": {
    "019fba9e-d801-7468-a9fe-a0f989eeacf4": {
      "fields": {
        "019fba9e-d801-7f61-8de7-5c88a4c67026": {
          "name": "email",
          "type": "string"
        },
        "019fba9e-d802-75b8-bcf0-aa09314cda9f": {
          "name": "password",
          "type": "string"
        },
        "019fba9e-d803-7e57-8151-5145ee97b082": {
          "name": "caller_id",
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
    "019fba9e-d801-774e-9b13-f3fb3563685c": {
      "name": "message",
      "type": "string"
    }
  },
  "name": "MessageOutput",
  "version": "1.0.0"
}
```

---

## token

### Issue an ephemeral API key for privilege elevation

**`POST`** `/system/auth/token/elevate`

Issue an ephemeral API key for privilege elevation

- **Handler:** `system:auth:token:elevate`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7569-a78c-c495be5275ee": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d801-7041-9540-bc83d5ed480b"
      },
      "type": "object"
    }
  },
  "name": "ElevateInput",
  "schemas": {
    "019fba9e-d801-7041-9540-bc83d5ed480b": {
      "fields": {
        "019fba9e-d801-720a-96f2-c83b5de18367": {
          "name": "email",
          "type": "string"
        },
        "019fba9e-d802-736f-a260-5a4991fbfc4b": {
          "name": "password",
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
    "019fba9e-d801-7cf9-af16-cd1cba54f07c": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-76eb-9002-c7c67184b522"
      },
      "type": "object"
    }
  },
  "name": "ElevateOutput",
  "schemas": {
    "019fba9e-d801-76eb-9002-c7c67184b522": {
      "fields": {
        "019fba9e-d801-7cc2-b2da-060f24fe6c89": {
          "name": "key",
          "type": "string"
        }
      },
      "name": "ElevateDocumentView"
    }
  },
  "version": "1.0.0"
}
```

---
