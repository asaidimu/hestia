# Auth API

## session

### Authenticate and receive tokens

**`POST`** `/system/auth/session`

Authenticate and receive tokens

- **Handler:** `system:auth:session:create`
- **Bootstrap-safe:** Yes

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "login_input",
  "description": "Login request",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "login_payload"
      }
    }
  },
  "schemas": {
    "login_payload": {
      "name": "LoginPayload",
      "fields": {
        "email": {
          "name": "email",
          "description": "User email address",
          "type": "string"
        },
        "password": {
          "name": "password",
          "description": "User password",
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
  "name": "login_output",
  "description": "Login response with tokens and user",
  "fields": {
    "document": {
      "name": "document",
      "description": "Login response document",
      "type": "object",
      "schema": {
        "id": "login_document"
      }
    }
  },
  "schemas": {
    "login_document": {
      "name": "LoginDocument",
      "fields": {
        "token": {
          "name": "token",
          "description": "Token bundle",
          "type": "object",
          "schema": {
            "id": "login_token"
          }
        },
        "user": {
          "name": "user",
          "description": "Authenticated user data",
          "type": "record"
        }
      }
    },
    "login_token": {
      "name": "LoginToken",
      "fields": {
        "access": {
          "name": "access",
          "description": "JWT access token",
          "type": "string"
        },
        "refresh": {
          "name": "refresh",
          "description": "JWT refresh token",
          "type": "string"
        },
        "type": {
          "name": "type",
          "description": "Token type (Bearer)",
          "type": "string"
        },
        "validity": {
          "name": "validity",
          "description": "Token validity in seconds",
          "type": "integer"
        }
      }
    }
  }
}
```

---

### Refresh via session token

**`PATCH`** `/system/auth/session`

Refresh via session token

- **Handler:** `system:auth:session:refresh`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "refresh_session_input",
  "description": "Refresh session token request",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "refresh_session_payload"
      }
    }
  },
  "schemas": {
    "refresh_session_payload": {
      "name": "RefreshSessionPayload",
      "fields": {
        "refresh_token": {
          "name": "refresh_token",
          "description": "Session token",
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
  "name": "refresh_output",
  "description": "Token refresh response",
  "fields": {
    "document": {
      "name": "document",
      "description": "Refresh response document",
      "type": "object",
      "schema": {
        "id": "refresh_document"
      }
    }
  },
  "schemas": {
    "refresh_document": {
      "name": "RefreshDocument",
      "fields": {
        "token": {
          "name": "token",
          "description": "Token bundle",
          "type": "object",
          "schema": {
            "id": "refresh_token"
          }
        }
      }
    },
    "refresh_token": {
      "name": "RefreshToken",
      "fields": {
        "access": {
          "name": "access",
          "description": "JWT access token",
          "type": "string"
        },
        "refresh": {
          "name": "refresh",
          "description": "JWT refresh token",
          "type": "string"
        },
        "type": {
          "name": "type",
          "description": "Token type (Bearer)",
          "type": "string"
        },
        "validity": {
          "name": "validity",
          "description": "Token validity in seconds",
          "type": "integer"
        }
      }
    }
  }
}
```

---

### Logout and revoke current token

**`DELETE`** `/system/auth/session`

Logout and revoke current token

- **Handler:** `system:auth:session:delete`
- **Bootstrap-safe:** Yes

---

### Validate a session token

**`GET`** `/system/auth/session`

Validate a session token

- **Handler:** `system:auth:session:validate`

#### Response

```json
{
  "version": "1.0.0",
  "name": "claims",
  "description": "Token claims with user identity and metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "Claims document",
      "type": "object",
      "schema": {
        "id": "claims_document"
      }
    }
  },
  "schemas": {
    "claims_document": {
      "name": "ClaimsDocument",
      "fields": {
        "email": {
          "name": "email",
          "description": "User email address",
          "type": "string"
        },
        "expires_at": {
          "name": "expires_at",
          "description": "Token expiration timestamp",
          "type": "string"
        },
        "scopes": {
          "name": "scopes",
          "description": "Assigned permission scopes",
          "type": "array",
          "schema": {
            "id": "",
            "type": "string"
          }
        },
        "token_id": {
          "name": "token_id",
          "description": "Unique token identifier",
          "type": "string"
        },
        "token_type": {
          "name": "token_type",
          "description": "Token type (access/refresh)",
          "type": "string"
        },
        "user_id": {
          "name": "user_id",
          "description": "Authenticated user ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

## user

### Register a new user

**`POST`** `/system/auth/user`

Register a new user

- **Handler:** `system:auth:user:register`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "register_input",
  "description": "User registration request",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "register_payload"
      }
    }
  },
  "schemas": {
    "register_payload": {
      "name": "RegisterPayload",
      "fields": {
        "email": {
          "name": "email",
          "description": "User email address",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "User display name",
          "type": "string"
        },
        "password": {
          "name": "password",
          "description": "User password",
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
  "name": "user",
  "description": "A user account",
  "fields": {
    "document": {
      "name": "document",
      "description": "User document",
      "type": "object",
      "schema": {
        "id": "user_document"
      }
    }
  },
  "schemas": {
    "user_document": {
      "name": "UserDocument",
      "fields": {
        "_id": {
          "name": "_id",
          "description": "Unique user identifier",
          "type": "string"
        },
        "email": {
          "name": "email",
          "description": "User email address",
          "type": "string"
        },
        "name": {
          "name": "name",
          "description": "User display name",
          "type": "string"
        },
        "scopes": {
          "name": "scopes",
          "description": "Assigned permission scopes",
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

## password

### Request password reset email

**`POST`** `/system/auth/password`

Request password reset email

- **Handler:** `system:auth:password:reset`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "password_reset_input",
  "description": "Password reset request",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "password_reset_payload"
      }
    }
  },
  "schemas": {
    "password_reset_payload": {
      "name": "PasswordResetPayload",
      "fields": {
        "email": {
          "name": "email",
          "description": "Account email address",
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
  "name": "message",
  "description": "A simple status message response",
  "fields": {
    "message": {
      "name": "message",
      "description": "Human-readable status message",
      "type": "string"
    }
  }
}
```

---

### Confirm password reset with token

**`PATCH`** `/system/auth/password`

Confirm password reset with token

- **Handler:** `system:auth:password:confirm`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "password_confirm_input",
  "description": "Password confirmation with reset token",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "password_confirm_payload"
      }
    }
  },
  "schemas": {
    "password_confirm_payload": {
      "name": "PasswordConfirmPayload",
      "fields": {
        "password": {
          "name": "password",
          "description": "New password",
          "type": "string"
        },
        "token": {
          "name": "token",
          "description": "Reset token",
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
  "name": "message",
  "description": "A simple status message response",
  "fields": {
    "message": {
      "name": "message",
      "description": "Human-readable status message",
      "type": "string"
    }
  }
}
```

---

## token

### Validate a JWT access token

**`GET`** `/system/auth/token`

Validate a JWT access token

- **Handler:** `system:auth:token:validate`
- **Internal:** Yes

#### Response

```json
{
  "version": "1.0.0",
  "name": "claims",
  "description": "Token claims with user identity and metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "Claims document",
      "type": "object",
      "schema": {
        "id": "claims_document"
      }
    }
  },
  "schemas": {
    "claims_document": {
      "name": "ClaimsDocument",
      "fields": {
        "email": {
          "name": "email",
          "description": "User email address",
          "type": "string"
        },
        "expires_at": {
          "name": "expires_at",
          "description": "Token expiration timestamp",
          "type": "string"
        },
        "scopes": {
          "name": "scopes",
          "description": "Assigned permission scopes",
          "type": "array",
          "schema": {
            "id": "",
            "type": "string"
          }
        },
        "token_id": {
          "name": "token_id",
          "description": "Unique token identifier",
          "type": "string"
        },
        "token_type": {
          "name": "token_type",
          "description": "Token type (access/refresh)",
          "type": "string"
        },
        "user_id": {
          "name": "user_id",
          "description": "Authenticated user ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Check if a token is blocklisted

**`POST`** `/system/auth/token/query`

Check if a token is blocklisted

- **Handler:** `system:auth:token:check`
- **Internal:** Yes

#### Response

```json
{
  "version": "1.0.0",
  "name": "blocklist_check",
  "description": "Token blocklist check result",
  "fields": {
    "document": {
      "name": "document",
      "description": "Blocklist check result",
      "type": "object",
      "schema": {
        "id": "blocklist_document"
      }
    }
  },
  "schemas": {
    "blocklist_document": {
      "name": "BlocklistDocument",
      "fields": {
        "blocklisted": {
          "name": "blocklisted",
          "description": "Whether the token is blocklisted",
          "type": "boolean"
        }
      }
    }
  }
}
```

---

## apikey

### Validate an API key

**`GET`** `/system/auth/apikey`

Validate an API key

- **Handler:** `system:auth:apikey:validate`
- **Internal:** Yes

#### Response

```json
{
  "version": "1.0.0",
  "name": "claims",
  "description": "Token claims with user identity and metadata",
  "fields": {
    "document": {
      "name": "document",
      "description": "Claims document",
      "type": "object",
      "schema": {
        "id": "claims_document"
      }
    }
  },
  "schemas": {
    "claims_document": {
      "name": "ClaimsDocument",
      "fields": {
        "email": {
          "name": "email",
          "description": "User email address",
          "type": "string"
        },
        "expires_at": {
          "name": "expires_at",
          "description": "Token expiration timestamp",
          "type": "string"
        },
        "scopes": {
          "name": "scopes",
          "description": "Assigned permission scopes",
          "type": "array",
          "schema": {
            "id": "",
            "type": "string"
          }
        },
        "token_id": {
          "name": "token_id",
          "description": "Unique token identifier",
          "type": "string"
        },
        "token_type": {
          "name": "token_type",
          "description": "Token type (access/refresh)",
          "type": "string"
        },
        "user_id": {
          "name": "user_id",
          "description": "Authenticated user ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

## bootstrap

### Set bootstrap admin password

**`PATCH`** `/system/auth/bootstrap`

Set bootstrap admin password

- **Handler:** `system:auth:bootstrap:password:set`
- **Bootstrap-safe:** Yes

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "bootstrap_password_input",
  "description": "Bootstrap password change request",
  "fields": {
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "bootstrap_password_payload"
      }
    }
  },
  "schemas": {
    "bootstrap_password_payload": {
      "name": "BootstrapPasswordPayload",
      "fields": {
        "email": {
          "name": "email",
          "description": "New admin email",
          "type": "string"
        },
        "password": {
          "name": "password",
          "description": "New admin password",
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
  "name": "message",
  "description": "A simple status message response",
  "fields": {
    "message": {
      "name": "message",
      "description": "Human-readable status message",
      "type": "string"
    }
  }
}
```

---
