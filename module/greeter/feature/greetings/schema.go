package greetings

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/internal/core/schema"
)

func salutationCreateInputSchema() *definition.Schema {
	return schema.MustFromJSON([]byte(`{
		"name": "SalutationCreateInput",
		"version": "1.0.0",
		"fields": {
			"payload": {
				"name": "payload",
				"type": "object",
				"schema": { "id": "salutation_create_payload" }
			}
		},
		"schemas": {
			"salutation_create_payload": {
				"name": "SalutationCreatePayload",
				"fields": {
					"phrase": { "name": "phrase", "type": "string" },
					"creator": { "name": "creator", "type": "string" }
				}
			}
		}
	}`))
}

func salutationGetInputSchema() *definition.Schema {
	return schema.MustFromJSON([]byte(`{
		"name": "SalutationGetInput",
		"version": "1.0.0",
		"fields": {
			"arguments": {
				"name": "arguments",
				"type": "object",
				"schema": { "id": "salutation_get_args" }
			}
		},
		"schemas": {
			"salutation_get_args": {
				"name": "SalutationGetArgs",
				"fields": {
					"id": { "name": "id", "type": "string", "required": true }
				}
			}
		}
	}`))
}

func greetingGenerateInputSchema() *definition.Schema {
	return schema.MustFromJSON([]byte(`{
		"name": "GreetingGenerateInput",
		"version": "1.0.0",
		"fields": {
			"payload": {
				"name": "payload",
				"type": "object",
				"schema": { "id": "greeting_generate_payload" }
			}
		},
		"schemas": {
			"greeting_generate_payload": {
				"name": "GreetingGeneratePayload",
				"fields": {
					"name": { "name": "name", "type": "string", "required": true },
					"salutation_id": { "name": "salutation_id", "type": "string" }
				}
			}
		}
	}`))
}

func salutationOutputSchema() *definition.Schema {
	return schema.MustFromJSON([]byte(`{
		"name": "SalutationOutput",
		"fields": {
			"id": { "name": "id", "type": "string" },
			"phrase": { "name": "phrase", "type": "string" },
			"creator": { "name": "creator", "type": "string" }
		}
	}`))
}

func greetingOutputSchema() *definition.Schema {
	return schema.MustFromJSON([]byte(`{
		"name": "GreetingOutput",
		"fields": {
			"greeting": { "name": "greeting", "type": "string" }
		}
	}`))
}

func salutationListOutputSchema() *definition.Schema {
	return schema.MustFromJSON([]byte(`{
		"name": "SalutationListOutput",
		"fields": {
			"salutations": { "name": "salutations", "type": "array", "schema": { "id": "Salutation" } },
			"total": { "name": "total", "type": "integer" }
		},
		"schemas": {
			"Salutation": {
				"name": "Salutation",
				"fields": {
					"id": { "name": "id", "type": "string" },
					"phrase": { "name": "phrase", "type": "string" },
					"creator": { "name": "creator", "type": "string" }
				}
			}
		}
	}`))
}
