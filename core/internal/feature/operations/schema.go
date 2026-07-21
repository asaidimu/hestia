package operations

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/schema"
)

var (
	_healthOutput        = schema.MustFromJSON(healthOutputJSON)
	_documentationOutput = schema.MustFromJSON(documentationOutputJSON)
	_capabilitiesOutput  = schema.MustFromJSON(capabilitiesOutputJSON)
	_capabilityNameInput = schema.MustFromJSON(capabilityNameInputJSON)
	_messageOutput       = schema.MustFromJSON(messageOutputJSON)
)

func healthOutputSchema() *definition.Schema        { return _healthOutput }
func documentationOutputSchema() *definition.Schema  { return _documentationOutput }
func capabilitiesOutputSchema() *definition.Schema   { return _capabilitiesOutput }
func capabilityNameInputSchema() *definition.Schema  { return _capabilityNameInput }
func messageOutputSchema() *definition.Schema        { return _messageOutput }

var healthOutputJSON = []byte(`{
	"name": "health",
	"description": "System health and bootstrap status",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Health status document",
			"type": "object",
			"schema": { "id": "health_document" }
		}
	},
	"schemas": {
		"health_document": {
			"name": "Health Status",
			"fields": {
				"ok": { "name": "ok", "description": "Whether the system is healthy", "type": "boolean" },
				"bootstrapped": { "name": "bootstrapped", "description": "Whether the system has been bootstrapped", "type": "boolean" }
			}
		}
	}
}`)

var documentationOutputJSON = []byte(`{
	"name": "documentation",
	"description": "List of all registered API endpoints with metadata",
	"version": "1.0.0",
	"fields": {
		"documents": {
			"name": "documents",
			"description": "Array of endpoint metadata objects",
			"type": "array",
			"schema": { "id": "endpoint_doc" }
		}
	},
	"schemas": {
		"endpoint_doc": {
			"name": "EndpointDoc",
			"fields": {
				"name": { "name": "name", "description": "Handler name", "type": "string" },
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"enabled": { "name": "enabled", "description": "Whether the handler is enabled", "type": "boolean" },
				"intent": { "name": "intent", "description": "Intent type", "type": "string" },
				"bootstrap_safe": { "name": "bootstrap_safe", "description": "Whether safe during bootstrap", "type": "boolean" },
				"internal": { "name": "internal", "description": "Whether the handler is internal-only", "type": "boolean" },
				"http": {
					"name": "http",
					"description": "HTTP method and route",
					"type": "object",
					"schema": { "id": "http_mapping" }
				},
				"input": { "name": "input", "description": "Input schema definition", "type": "record" },
				"output": { "name": "output", "description": "Output schema definition", "type": "record" }
			}
		},
		"http_mapping": {
			"name": "HTTPMapping",
			"fields": {
				"method": { "name": "method", "description": "HTTP method", "type": "string" },
				"route": { "name": "route", "description": "HTTP route path", "type": "string" },
				"pattern": { "name": "pattern", "description": "Combined method and route", "type": "string" }
			}
		}
	}
}`)

var capabilitiesOutputJSON = []byte(`{
	"name": "capabilities",
	"description": "List of registered command and query capabilities",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Capabilities document",
			"type": "object",
			"schema": { "id": "capabilities_document" }
		}
	},
	"schemas": {
		"capabilities_document": {
			"name": "CapabilitiesDocument",
			"fields": {
				"capabilities": {
					"name": "capabilities",
					"description": "Array of handler capabilities",
					"type": "array",
					"schema": { "id": "capability_item" }
				}
			}
		},
		"capability_item": {
			"name": "CapabilityItem",
			"fields": {
				"name": { "name": "name", "description": "Handler name", "type": "string" },
				"intent_type": { "name": "intent_type", "description": "Command or query", "type": "string" },
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"enabled": { "name": "enabled", "description": "Whether the handler is enabled", "type": "boolean" }
			}
		}
	}
}`)

var capabilityNameInputJSON = []byte(`{
	"name": "capability_name_input",
	"description": "Capability identifier from the path with enabled payload",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Capability name argument",
			"type": "object",
			"schema": { "id": "capability_name_input_arguments" }
		},
		"payload": {
			"name": "payload",
			"description": "Enabled flag payload",
			"type": "object",
			"schema": { "id": "capability_name_input_payload" }
		}
	},
	"schemas": {
		"capability_name_input_arguments": {
			"name": "CapabilityNameInputArguments",
			"fields": {
				"name": { "name": "name", "description": "Name of the capability to modify", "type": "string" }
			}
		},
		"capability_name_input_payload": {
			"name": "CapabilityNameInputPayload",
			"fields": {
				"enabled": { "name": "enabled", "description": "Whether the capability is enabled", "type": "boolean" }
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
