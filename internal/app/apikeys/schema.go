package apikeys

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/app/core/schema"
)

var (
	_apiKeyListOutput  = schema.MustFromJSON(apiKeyListOutputJSON)
	_apiKeyCreateInput = schema.MustFromJSON(apiKeyCreateInputJSON)
	_apiKeyGetInput    = schema.MustFromJSON(apiKeyGetInputJSON)
	_apiKeyUpdateInput = schema.MustFromJSON(apiKeyUpdateInputJSON)
	_apiKeyDeleteInput = schema.MustFromJSON(apiKeyDeleteInputJSON)
	_apiKeyRotateInput = schema.MustFromJSON(apiKeyRotateInputJSON)
	_apiKeyOutput      = schema.MustFromJSON(apiKeyOutputJSON)
)

func apiKeyListOutputSchema() *definition.Schema  { return _apiKeyListOutput }
func apiKeyCreateInputSchema() *definition.Schema { return _apiKeyCreateInput }
func apiKeyGetInputSchema() *definition.Schema    { return _apiKeyGetInput }
func apiKeyUpdateInputSchema() *definition.Schema { return _apiKeyUpdateInput }
func apiKeyDeleteInputSchema() *definition.Schema { return _apiKeyDeleteInput }
func apiKeyRotateInputSchema() *definition.Schema { return _apiKeyRotateInput }
func apiKeyOutputSchema() *definition.Schema      { return _apiKeyOutput }

var apiKeyListOutputJSON = []byte(`{
	"name": "api_key_list",
	"description": "List of API keys",
	"version": "1.0.0",
	"fields": {
		"documents": {
			"name": "documents",
			"description": "Array of API keys",
			"type": "array",
			"schema": { "id": "api_key_document" }
		}
	},
	"schemas": {
		"api_key_document": {
			"name": "APIKeyDocument",
			"fields": {
				"_id": { "name": "_id", "description": "Unique key identifier", "type": "string" },
				"name": { "name": "name", "description": "Display name", "type": "string" },
				"prefix": { "name": "prefix", "description": "Key prefix with hint", "type": "string" },
				"status": { "name": "status", "description": "Key status", "type": "string" },
				"environment": { "name": "environment", "description": "Environment restriction", "type": "string" }
			}
		}
	}
}`)

var apiKeyCreateInputJSON = []byte(`{
	"name": "api_key_create_input",
	"description": "API key creation request",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "Key creation details",
			"type": "object",
			"schema": { "id": "api_key_create_payload" }
		}
	},
	"schemas": {
		"api_key_create_payload": {
			"name": "APIKeyCreatePayload",
			"fields": {
				"name": { "name": "name", "description": "Display name for the key", "type": "string" },
				"environment": { "name": "environment", "description": "Environment restriction", "type": "string" },
				"status": { "name": "status", "description": "Key status", "type": "string" },
				"limits": { "name": "limits", "description": "Rate limits configuration", "type": "record" },
				"ip": { "name": "ip", "description": "IP restriction rules", "type": "string" }
			}
		}
	}
}`)

var apiKeyGetInputJSON = []byte(`{
	"name": "api_key_get_input",
	"description": "API key identifier from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Request arguments",
			"type": "object",
			"schema": { "id": "api_key_get_arguments" }
		}
	},
	"schemas": {
		"api_key_get_arguments": {
			"name": "APIKeyGetArguments",
			"fields": {
				"key_id": { "name": "key_id", "description": "Unique API key identifier", "type": "string" }
			}
		}
	}
}`)

var apiKeyUpdateInputJSON = []byte(`{
	"name": "api_key_update_input",
	"description": "API key update request",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Request arguments",
			"type": "object",
			"schema": { "id": "api_key_update_arguments" }
		},
		"payload": {
			"name": "payload",
			"description": "Key fields to update",
			"type": "object",
			"schema": { "id": "api_key_update_payload" }
		}
	},
	"schemas": {
		"api_key_update_arguments": {
			"name": "APIKeyUpdateArguments",
			"fields": {
				"key_id": { "name": "key_id", "description": "Unique API key identifier", "type": "string" }
			}
		},
		"api_key_update_payload": {
			"name": "APIKeyUpdatePayload",
			"fields": {
				"name": { "name": "name", "description": "Display name", "type": "string" },
				"status": { "name": "status", "description": "Key status (active/revoked)", "type": "string" },
				"environment": { "name": "environment", "description": "Environment restriction", "type": "string" },
				"operations": { "name": "operations", "description": "Operation allowlist (empty = allow all, nil = backward compatible)", "type": "array", "schema": { "type": "string" } },
				"expiry": { "name": "expiry", "description": "Expiration timestamp (RFC3339)", "type": "string" }
			}
		}
	}
}`)

var apiKeyDeleteInputJSON = []byte(`{
	"name": "api_key_delete_input",
	"description": "API key identifier from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Request arguments",
			"type": "object",
			"schema": { "id": "api_key_delete_arguments" }
		}
	},
	"schemas": {
		"api_key_delete_arguments": {
			"name": "APIKeyDeleteArguments",
			"fields": {
				"key_id": { "name": "key_id", "description": "Unique API key identifier", "type": "string" }
			}
		}
	}
}`)

var apiKeyRotateInputJSON = []byte(`{
	"name": "api_key_rotate_input",
	"description": "API key identifier for rotation",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Request arguments",
			"type": "object",
			"schema": { "id": "api_key_rotate_arguments" }
		}
	},
	"schemas": {
		"api_key_rotate_arguments": {
			"name": "APIKeyRotateArguments",
			"fields": {
				"key_id": { "name": "key_id", "description": "Unique API key identifier", "type": "string" }
			}
		}
	}
}`)

var apiKeyOutputJSON = []byte(`{
	"name": "api_key",
	"description": "An API key with metadata",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "API key document",
			"type": "object",
			"schema": { "id": "api_key_document" }
		}
	},
	"schemas": {
		"api_key_document": {
			"name": "APIKeyDocument",
			"fields": {
				"_id": { "name": "_id", "description": "Unique key identifier", "type": "string" },
				"name": { "name": "name", "description": "Display name", "type": "string" },
				"prefix": { "name": "prefix", "description": "Key prefix with hint", "type": "string" },
				"status": { "name": "status", "description": "Key status", "type": "string" },
				"environment": { "name": "environment", "description": "Environment restriction", "type": "string" },
				"key": { "name": "key", "description": "Raw API key value (shown once on create/rotate)", "type": "string" }
			}
		}
	}
}`)
