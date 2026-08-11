package auth

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_loginOutput     = dispatch.MustFromJSON(loginOutputJSON)
	_messageOutput   = dispatch.MustFromJSON(messageOutputJSON)
	_claimsOutput    = dispatch.MustFromJSON(claimsOutputJSON)
	_elevateOutput   = dispatch.MustFromJSON(elevateOutputJSON)
)

func loginOutputSchema() *definition.Schema            { return _loginOutput }
func messageOutputSchema() *definition.Schema          { return _messageOutput }
func claimsOutputSchema() *definition.Schema           { return _claimsOutput }
func elevateOutputSchema() *definition.Schema          { return _elevateOutput }

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

var messageOutputJSON = []byte(`{
	"name": "message",
	"description": "A simple status message response",
	"version": "1.0.0",
	"fields": {
		"message": { "name": "message", "description": "Human-readable status message", "type": "string" }
	}
}`)

var elevateOutputJSON = []byte(`{
	"name": "elevate_output",
	"description": "Ephemeral API key for privilege elevation",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Elevation response document",
			"type": "object",
			"schema": { "id": "elevate_document" }
		}
	},
	"schemas": {
		"elevate_document": {
			"name": "ElevateDocument",
			"fields": {
				"key": { "name": "key", "description": "Ephemeral API key string", "type": "string" }
			}
		}
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
