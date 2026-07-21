package auth

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/schema"
)

var (
	_registerInput   = schema.MustFromJSON(registerInputJSON)
	_userOutput      = schema.MustFromJSON(userOutputJSON)
	_loginInput      = schema.MustFromJSON(loginInputJSON)
	_loginOutput     = schema.MustFromJSON(loginOutputJSON)
	_passwordReset   = schema.MustFromJSON(passwordResetInputJSON)
	_passwordConfirm = schema.MustFromJSON(passwordConfirmInputJSON)
	_bootstrapPwd    = schema.MustFromJSON(bootstrapPasswordInputJSON)
	_messageOutput   = schema.MustFromJSON(messageOutputJSON)
	_claimsOutput    = schema.MustFromJSON(claimsOutputJSON)
)

func registerInputSchema() *definition.Schema        { return _registerInput }
func userOutputSchema() *definition.Schema            { return _userOutput }
func loginInputSchema() *definition.Schema            { return _loginInput }
func loginOutputSchema() *definition.Schema           { return _loginOutput }
func passwordResetInputSchema() *definition.Schema    { return _passwordReset }
func passwordConfirmInputSchema() *definition.Schema   { return _passwordConfirm }
func bootstrapPasswordInputSchema() *definition.Schema { return _bootstrapPwd }
func messageOutputSchema() *definition.Schema         { return _messageOutput }
func claimsOutputSchema() *definition.Schema           { return _claimsOutput }

var registerInputJSON = []byte(`{
	"name": "register_input",
	"description": "User registration request",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "register_payload" }
		}
	},
	"schemas": {
		"register_payload": {
			"name": "RegisterPayload",
			"fields": {
				"email": { "name": "email", "description": "User email address", "type": "string" },
				"password": { "name": "password", "description": "User password", "type": "string" },
				"name": { "name": "name", "description": "User display name", "type": "string" }
			}
		}
	}
}`)

var userOutputJSON = []byte(`{
	"name": "user",
	"description": "A user account",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "User document",
			"type": "object",
			"schema": { "id": "user_document" }
		}
	},
	"schemas": {
		"user_document": {
			"name": "UserDocument",
			"fields": {
				"_id": { "name": "_id", "description": "Unique user identifier", "type": "string" },
				"email": { "name": "email", "description": "User email address", "type": "string" },
				"name": { "name": "name", "description": "User display name", "type": "string" },
				"permissions": { "name": "permissions", "description": "Assigned permission scopes", "type": "array", "schema": { "type": "string" } }
			}
		}
	}
}`)

var loginInputJSON = []byte(`{
	"name": "login_input",
	"description": "Login request",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "login_payload" }
		}
	},
	"schemas": {
		"login_payload": {
			"name": "LoginPayload",
			"fields": {
				"email": { "name": "email", "description": "User email address", "type": "string" },
				"password": { "name": "password", "description": "User password", "type": "string" }
			}
		}
	}
}`)

var loginOutputJSON = []byte(`{
	"name": "login_output",
	"description": "Login response with session token and user",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Login response document",
			"type": "object",
			"schema": { "id": "login_document" }
		}
	},
	"schemas": {
		"login_document": {
			"name": "LoginDocument",
			"fields": {
				"token": { "name": "token", "description": "Session token string", "type": "string" },
				"user": { "name": "user", "description": "Authenticated user data", "type": "record" }
			}
		}
	}
}`)

var passwordResetInputJSON = []byte(`{
	"name": "password_reset_input",
	"description": "Password reset request",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "password_reset_payload" }
		}
	},
	"schemas": {
		"password_reset_payload": {
			"name": "PasswordResetPayload",
			"fields": {
				"email": { "name": "email", "description": "Account email address", "type": "string" }
			}
		}
	}
}`)

var passwordConfirmInputJSON = []byte(`{
	"name": "password_confirm_input",
	"description": "Password confirmation with reset token",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "password_confirm_payload" }
		}
	},
	"schemas": {
		"password_confirm_payload": {
			"name": "PasswordConfirmPayload",
			"fields": {
				"token": { "name": "token", "description": "Reset token", "type": "string" },
				"password": { "name": "password", "description": "New password", "type": "string" }
			}
		}
	}
}`)

var bootstrapPasswordInputJSON = []byte(`{
	"name": "bootstrap_password_input",
	"description": "Bootstrap password change request",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "bootstrap_password_payload" }
		}
	},
	"schemas": {
		"bootstrap_password_payload": {
			"name": "BootstrapPasswordPayload",
			"fields": {
				"email": { "name": "email", "description": "New admin email", "type": "string" },
				"password": { "name": "password", "description": "New admin password", "type": "string" }
			}
		}
	}
}`)

var messageOutputJSON = []byte(`{
	"name": "message",
	"description": "A simple status message response",
	"version": "1.0.0",
	"fields": {
		"message": { "name": "message", "description": "Human-readable status message", "type": "string" }
	}
}`)

var claimsOutputJSON = []byte(`{
	"name": "claims",
	"description": "Token claims with user identity and metadata",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Claims document",
			"type": "object",
			"schema": { "id": "claims_document" }
		}
	},
	"schemas": {
		"claims_document": {
			"name": "ClaimsDocument",
			"fields": {
				"user_id": { "name": "user_id", "description": "Authenticated user ID", "type": "string" },
				"session_id": { "name": "session_id", "description": "Unique session identifier", "type": "string" },
				"issued_at": { "name": "issued_at", "description": "Token issuance timestamp", "type": "integer" },
				"expires_at": { "name": "expires_at", "description": "Token expiration timestamp", "type": "integer" },
				"created_at": { "name": "created_at", "description": "Session creation timestamp", "type": "integer" }
			}
		}
	}
}`)
