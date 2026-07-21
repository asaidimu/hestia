package users

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/schema"
)

var (
	_userRegisterInput       = schema.MustFromJSON(userRegisterInputJSON)
	_userLoginInput          = schema.MustFromJSON(userLoginInputJSON)
	_userQueryInput          = schema.MustFromJSON(userQueryInputJSON)
	_userQueryOutput         = schema.MustFromJSON(userQueryOutputJSON)
	_userOutput              = schema.MustFromJSON(userOutputJSON)
	_userNameInput           = schema.MustFromJSON(userNameInputJSON)
	_userUpdateInput         = schema.MustFromJSON(userUpdateInputJSON)
	_userGetInput            = schema.MustFromJSON(userGetInputJSON)
	_userChangePasswordInput = schema.MustFromJSON(userChangePasswordInputJSON)
	_userDeleteInput         = schema.MustFromJSON(userDeleteInputJSON)
	_messageOutput           = schema.MustFromJSON(messageOutputJSON)
)

func userRegisterInputSchema() *definition.Schema       { return _userRegisterInput }
func userLoginInputSchema() *definition.Schema          { return _userLoginInput }
func userQueryInputSchema() *definition.Schema          { return _userQueryInput }
func userQueryOutputSchema() *definition.Schema         { return _userQueryOutput }
func userOutputSchema() *definition.Schema               { return _userOutput }
func userNameInputSchema() *definition.Schema            { return _userNameInput }
func userUpdateInputSchema() *definition.Schema          { return _userUpdateInput }
func userGetInputSchema() *definition.Schema             { return _userGetInput }
func userChangePasswordInputSchema() *definition.Schema  { return _userChangePasswordInput }
func userDeleteInputSchema() *definition.Schema          { return _userDeleteInput }
func messageOutputSchema() *definition.Schema            { return _messageOutput }

var userRegisterInputJSON = []byte(`{
	"name": "user_register",
	"description": "Register a new user",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "The user registration payload",
			"type": "object",
			"schema": { "id": "user_register_payload" }
		}
	},
	"schemas": {
		"user_register_payload": {
			"name": "User Register Payload",
			"fields": {
				"username": { "name": "username", "description": "Desired username", "type": "string" },
				"password": { "name": "password", "description": "Desired password", "type": "string" },
				"display_name": { "name": "display_name", "description": "Display name for the user", "type": "string" }
			}
		}
	}
}`)

var userLoginInputJSON = []byte(`{
	"name": "user_login",
	"description": "Authenticate a user and receive a session token",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "The user login payload",
			"type": "object",
			"schema": { "id": "user_login_payload" }
		}
	},
	"schemas": {
		"user_login_payload": {
			"name": "User Login Payload",
			"fields": {
				"username": { "name": "username", "description": "The user's username", "type": "string" },
				"password": { "name": "password", "description": "The user's password", "type": "string" }
			}
		}
	}
}`)

var userQueryInputJSON = []byte(`{
	"name": "user_query",
	"description": "Query users with optional filters",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "Query payload",
			"type": "object",
			"schema": { "id": "user_query_payload" }
		}
	},
	"schemas": {
		"user_query_payload": {
			"name": "UserQueryPayload",
			"fields": {
				"username": { "name": "username", "description": "Filter by username (prefix match)", "type": "string" },
				"limit": { "name": "limit", "description": "Maximum number of results", "type": "integer" },
				"cursor": { "name": "cursor", "description": "Pagination cursor", "type": "string" }
			}
		}
	}
}`)

var userQueryOutputJSON = []byte(`{
	"name": "user_query_result",
	"description": "Paginated user query result",
	"version": "1.0.0",
	"fields": {
		"page": {
			"name": "page",
			"description": "Paginated list of users",
			"type": "object",
			"schema": { "id": "user_page" }
		}
	},
	"schemas": {
		"user_page": {
			"name": "UserPage",
			"fields": {
				"documents": {
					"name": "documents",
					"description": "Array of user documents",
					"type": "array",
					"schema": { "id": "user_document" }
				},
				"pagination": {
					"name": "pagination",
					"description": "Pagination metadata",
					"type": "object",
					"schema": { "id": "pagination_meta" }
				}
			}
		},
		"pagination_meta": {
			"name": "PaginationMeta",
			"fields": {
				"total": { "name": "total", "description": "Total number of matching documents", "type": "integer" },
				"cursor": { "name": "cursor", "description": "Cursor for next page", "type": "string" },
				"limit": { "name": "limit", "description": "Number of results requested", "type": "integer" }
			}
		},
		"user_document": {
			"name": "UserDocument",
			"fields": {
				"id": { "name": "id", "description": "User ID", "type": "string" },
				"username": { "name": "username", "description": "Username", "type": "string" },
				"display_name": { "name": "display_name", "description": "Display name", "type": "string" },
				"created_at": { "name": "created_at", "description": "Creation timestamp", "type": "string" },
				"updated_at": { "name": "updated_at", "description": "Last update timestamp", "type": "string" },
				"disabled": { "name": "disabled", "description": "Whether the user is disabled", "type": "boolean" }
			}
		}
	}
}`)

var userOutputJSON = []byte(`{
	"name": "user_output",
	"description": "User output schema",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "The user document",
			"type": "object",
			"schema": { "id": "user_document" }
		}
	},
	"schemas": {
		"user_document": {
			"name": "UserDocument",
			"fields": {
				"id": { "name": "id", "description": "User ID", "type": "string" },
				"username": { "name": "username", "description": "Username", "type": "string" },
				"display_name": { "name": "display_name", "description": "Display name", "type": "string" },
				"created_at": { "name": "created_at", "description": "Creation timestamp", "type": "string" },
				"updated_at": { "name": "updated_at", "description": "Last update timestamp", "type": "string" },
				"disabled": { "name": "disabled", "description": "Whether the user is disabled", "type": "boolean" }
			}
		}
	}
}`)

var userNameInputJSON = []byte(`{
	"name": "user_name_input",
	"description": "Username from the path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Username argument",
			"type": "object",
			"schema": { "id": "user_name_input_arguments" }
		}
	},
	"schemas": {
		"user_name_input_arguments": {
			"name": "UserNameInputArguments",
			"fields": {
				"username": { "name": "username", "description": "The username", "type": "string" }
			}
		}
	}
}`)

var userUpdateInputJSON = []byte(`{
	"name": "user_update_input",
	"description": "User update with fields to modify",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "User ID argument",
			"type": "object",
			"schema": { "id": "user_update_arguments" }
		},
		"payload": {
			"name": "payload",
			"description": "Fields to update",
			"type": "object",
			"schema": { "id": "user_update_payload" }
		}
	},
	"schemas": {
		"user_update_arguments": {
			"name": "UserUpdateArguments",
			"fields": {
				"user_id": { "name": "user_id", "description": "The user ID", "type": "string" }
			}
		},
		"user_update_payload": {
			"name": "UserUpdatePayload",
			"fields": {
				"display_name": { "name": "display_name", "description": "New display name", "type": "string" },
				"disabled": { "name": "disabled", "description": "Whether the user should be disabled", "type": "boolean" }
			}
		}
	}
}`)

var userGetInputJSON = []byte(`{
	"name": "user_get_input",
	"description": "User ID from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "user_get_arguments" }
		}
	},
	"schemas": {
		"user_get_arguments": {
			"name": "UserGetArguments",
			"fields": {
				"user_id": { "name": "user_id", "description": "The user ID", "type": "string" }
			}
		}
	}
}`)

var userChangePasswordInputJSON = []byte(`{
	"name": "user_change_password_input",
	"description": "Change password with current and new password",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "user_get_arguments" }
		},
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "change_password_payload" }
		}
	},
	"schemas": {
		"user_get_arguments": {
			"name": "UserGetArguments",
			"fields": {
				"user_id": { "name": "user_id", "description": "The user ID", "type": "string" }
			}
		},
		"change_password_payload": {
			"name": "ChangePasswordPayload",
			"fields": {
				"current": { "name": "current", "description": "Current password", "type": "string" },
				"new": { "name": "new", "description": "New password", "type": "string" }
			}
		}
	}
}`)

var userDeleteInputJSON = []byte(`{
	"name": "user_delete_input",
	"description": "Delete user by ID",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "user_get_arguments" }
		},
		"modifiers": {
			"name": "modifiers",
			"type": "object",
			"schema": { "id": "delete_modifiers" }
		}
	},
	"schemas": {
		"user_get_arguments": {
			"name": "UserGetArguments",
			"fields": {
				"user_id": { "name": "user_id", "description": "The user ID", "type": "string" }
			}
		},
		"delete_modifiers": {
			"name": "DeleteModifiers",
			"fields": {
				"permanent": { "name": "permanent", "description": "Whether to permanently delete", "type": "boolean" }
			}
		}
	}
}`)

var messageOutputJSON = []byte(`{
	"name": "user_message",
	"description": "A simple status message response",
	"version": "1.0.0",
	"fields": {
		"message": { "name": "message", "description": "Human-readable status message", "type": "string" }
	}
}`)
