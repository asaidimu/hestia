package schema

var InputMetaSchemaJSON = []byte(`{
	"name": "InputMetaSchema",
	"description": "Meta schema used to validate that registration schemas are valid",
	"version": "1.0.0",
	"fields": {
		"name": { "name": "name", "description": "Schema name", "type": "string" },
		"description": { "name": "description", "description": "Schema description", "type": "string" },
		"version": { "name": "version", "description": "Schema version", "type": "string" },
		"fields": {
			"name": "fields",
			"description": "Schema fields",
			"type": "record",
			"schema": { "id": "fields" }
		},
		"schemas": {
			"name": "schemas",
			"description": "Nested schemas",
			"type": "record"
		}
	},
	"schemas": {
		"fields": {
			"name": "fields",
			"fields": {
				"name": {
					"name": "name",
					"description": "Allowed field names",
					"required": true,
					"type": "enum",
					"schema": { "type": "string", "values": ["arguments", "modifiers", "payload"] }
				},
				"type": {
					"name": "type",
					"description": "Allowed field types",
					"required": true,
					"type": "enum",
					"schema": { "type": "string", "values": ["object", "record"] }
				},
				"description": {
					"name": "description",
					"description": "Field description",
					"type": "string"
				},
				"schema": {
					"name": "schema",
					"description": "schema reference",
					"type": "record"
				}
			}
		}
	}
}`)
