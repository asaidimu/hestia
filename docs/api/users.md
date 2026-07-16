# Users API

## user

### Query users collection

**`POST`** `/system/users/user/query`

Query users collection

- **Handler:** `system:users:user:query`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "user_query",
  "description": "Query users with optional filters",
  "fields": {
    "payload": {
      "name": "payload",
      "description": "Query payload",
      "type": "object",
      "schema": {
        "id": "user_query_payload"
      }
    }
  },
  "schemas": {
    "user_query_payload": {
      "name": "UserQueryPayload",
      "fields": {
        "cursor": {
          "name": "cursor",
          "description": "Pagination cursor",
          "type": "string"
        },
        "limit": {
          "name": "limit",
          "description": "Maximum number of results",
          "type": "integer"
        },
        "username": {
          "name": "username",
          "description": "Filter by username (prefix match)",
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
  "name": "user_query_result",
  "description": "Paginated user query result",
  "fields": {
    "page": {
      "name": "page",
      "description": "Paginated list of users",
      "type": "object",
      "schema": {
        "id": "user_page"
      }
    }
  },
  "schemas": {
    "pagination_meta": {
      "name": "PaginationMeta",
      "fields": {
        "cursor": {
          "name": "cursor",
          "description": "Cursor for next page",
          "type": "string"
        },
        "limit": {
          "name": "limit",
          "description": "Number of results requested",
          "type": "integer"
        },
        "total": {
          "name": "total",
          "description": "Total number of matching documents",
          "type": "integer"
        }
      }
    },
    "user_document": {
      "name": "UserDocument",
      "fields": {
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "disabled": {
          "name": "disabled",
          "description": "Whether the user is disabled",
          "type": "boolean"
        },
        "display_name": {
          "name": "display_name",
          "description": "Display name",
          "type": "string"
        },
        "id": {
          "name": "id",
          "description": "User ID",
          "type": "string"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last update timestamp",
          "type": "string"
        },
        "username": {
          "name": "username",
          "description": "Username",
          "type": "string"
        }
      }
    },
    "user_page": {
      "name": "UserPage",
      "fields": {
        "documents": {
          "name": "documents",
          "description": "Array of user documents",
          "type": "array",
          "schema": {
            "id": "user_document"
          }
        },
        "pagination": {
          "name": "pagination",
          "description": "Pagination metadata",
          "type": "object",
          "schema": {
            "id": "pagination_meta"
          }
        }
      }
    }
  }
}
```

---

### Get user by ID

**`GET`** `/system/users/user/{user_id}`

Get user by ID

- **Handler:** `system:users:user:get`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "user_get_input",
  "description": "User ID from path",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "user_get_arguments"
      }
    }
  },
  "schemas": {
    "user_get_arguments": {
      "name": "UserGetArguments",
      "fields": {
        "user_id": {
          "name": "user_id",
          "description": "The user ID",
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
  "name": "user_output",
  "description": "User output schema",
  "fields": {
    "document": {
      "name": "document",
      "description": "The user document",
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
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "disabled": {
          "name": "disabled",
          "description": "Whether the user is disabled",
          "type": "boolean"
        },
        "display_name": {
          "name": "display_name",
          "description": "Display name",
          "type": "string"
        },
        "id": {
          "name": "id",
          "description": "User ID",
          "type": "string"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last update timestamp",
          "type": "string"
        },
        "username": {
          "name": "username",
          "description": "Username",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Update user

**`PATCH`** `/system/users/user/{user_id}`

Update user

- **Handler:** `system:users:user:update`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "user_update_input",
  "description": "User update with fields to modify",
  "fields": {
    "arguments": {
      "name": "arguments",
      "description": "User ID argument",
      "type": "object",
      "schema": {
        "id": "user_update_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "description": "Fields to update",
      "type": "object",
      "schema": {
        "id": "user_update_payload"
      }
    }
  },
  "schemas": {
    "user_update_arguments": {
      "name": "UserUpdateArguments",
      "fields": {
        "user_id": {
          "name": "user_id",
          "description": "The user ID",
          "type": "string"
        }
      }
    },
    "user_update_payload": {
      "name": "UserUpdatePayload",
      "fields": {
        "disabled": {
          "name": "disabled",
          "description": "Whether the user should be disabled",
          "type": "boolean"
        },
        "display_name": {
          "name": "display_name",
          "description": "New display name",
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
  "name": "user_output",
  "description": "User output schema",
  "fields": {
    "document": {
      "name": "document",
      "description": "The user document",
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
        "created_at": {
          "name": "created_at",
          "description": "Creation timestamp",
          "type": "string"
        },
        "disabled": {
          "name": "disabled",
          "description": "Whether the user is disabled",
          "type": "boolean"
        },
        "display_name": {
          "name": "display_name",
          "description": "Display name",
          "type": "string"
        },
        "id": {
          "name": "id",
          "description": "User ID",
          "type": "string"
        },
        "updated_at": {
          "name": "updated_at",
          "description": "Last update timestamp",
          "type": "string"
        },
        "username": {
          "name": "username",
          "description": "Username",
          "type": "string"
        }
      }
    }
  }
}
```

---

### Delete user

**`DELETE`** `/system/users/user/{user_id}`

Delete user

- **Handler:** `system:users:user:delete`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "user_delete_input",
  "description": "Delete user by ID",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "user_get_arguments"
      }
    },
    "modifiers": {
      "name": "modifiers",
      "type": "object",
      "schema": {
        "id": "delete_modifiers"
      }
    }
  },
  "schemas": {
    "delete_modifiers": {
      "name": "DeleteModifiers",
      "fields": {
        "permanent": {
          "name": "permanent",
          "description": "Whether to permanently delete",
          "type": "boolean"
        }
      }
    },
    "user_get_arguments": {
      "name": "UserGetArguments",
      "fields": {
        "user_id": {
          "name": "user_id",
          "description": "The user ID",
          "type": "string"
        }
      }
    }
  }
}
```

---

## password

### Change user password

**`PATCH`** `/system/users/password/{user_id}`

Change user password

- **Handler:** `system:users:password:change`

#### Request Body

```json
{
  "version": "1.0.0",
  "name": "user_change_password_input",
  "description": "Change password with current and new password",
  "fields": {
    "arguments": {
      "name": "arguments",
      "type": "object",
      "schema": {
        "id": "user_get_arguments"
      }
    },
    "payload": {
      "name": "payload",
      "type": "object",
      "schema": {
        "id": "change_password_payload"
      }
    }
  },
  "schemas": {
    "change_password_payload": {
      "name": "ChangePasswordPayload",
      "fields": {
        "current": {
          "name": "current",
          "description": "Current password",
          "type": "string"
        },
        "new": {
          "name": "new",
          "description": "New password",
          "type": "string"
        }
      }
    },
    "user_get_arguments": {
      "name": "UserGetArguments",
      "fields": {
        "user_id": {
          "name": "user_id",
          "description": "The user ID",
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
  "name": "user_message",
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
