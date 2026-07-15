package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/internal/core/schema"
)

var (
	_policyNameInput                = schema.MustFromJSON(policyNameInputJSON)
	_ruleKeyInput                   = schema.MustFromJSON(ruleKeyInputJSON)
	_policyCreateInput              = schema.MustFromJSON(policyCreateInputJSON)
	_policyOperationOutput          = schema.MustFromJSON(policyOperationOutputJSON)
	_policyRuleOutput               = schema.MustFromJSON(policyRuleOutputJSON)
	_policyValidateInput            = schema.MustFromJSON(policyValidateInputJSON)
	_policyValidateOutput           = schema.MustFromJSON(policyValidateOutputJSON)
	_policyReloadOutput             = schema.MustFromJSON(policyReloadOutputJSON)
	_messageOutput                  = schema.MustFromJSON(messageOutputJSON)
	_policyOperationGetInput        = schema.MustFromJSON(policyOperationGetInputJSON)
	_policyRuleGetInput             = schema.MustFromJSON(policyRuleGetInputJSON)
	_policyOperationUpsertInput     = schema.MustFromJSON(policyOperationUpsertInputJSON)
	_policyOperationDeleteInput     = schema.MustFromJSON(policyOperationDeleteInputJSON)
	_policyRuleUpsertInput          = schema.MustFromJSON(policyRuleUpsertInputJSON)
	_policyRuleDeleteInput          = schema.MustFromJSON(policyRuleDeleteInputJSON)
)

func policyNameInputSchema() *definition.Schema              { return _policyNameInput }
func ruleKeyInputSchema() *definition.Schema                  { return _ruleKeyInput }
func policyCreateInputSchema() *definition.Schema             { return _policyCreateInput }
func policyOperationOutputSchema() *definition.Schema         { return _policyOperationOutput }
func policyRuleOutputSchema() *definition.Schema              { return _policyRuleOutput }
func policyValidateInputSchema() *definition.Schema           { return _policyValidateInput }
func policyValidateOutputSchema() *definition.Schema          { return _policyValidateOutput }
func policyReloadOutputSchema() *definition.Schema            { return _policyReloadOutput }
func messageOutputSchema() *definition.Schema                 { return _messageOutput }
func policyOperationGetInputSchema() *definition.Schema       { return _policyOperationGetInput }
func policyRuleGetInputSchema() *definition.Schema            { return _policyRuleGetInput }
func policyOperationUpsertInputSchema() *definition.Schema    { return _policyOperationUpsertInput }
func policyOperationDeleteInputSchema() *definition.Schema    { return _policyOperationDeleteInput }
func policyRuleUpsertInputSchema() *definition.Schema         { return _policyRuleUpsertInput }
func policyRuleDeleteInputSchema() *definition.Schema         { return _policyRuleDeleteInput }

var policyNameInputJSON = []byte(`{
	"name": "policy_name_input",
	"description": "Policy name from the path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Policy name argument",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		}
	}
}`)

var ruleKeyInputJSON = []byte(`{
	"name": "rule_key_input",
	"description": "Policy name and rule key from the path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Rule key arguments",
			"type": "object",
			"schema": { "id": "rule_key_arguments" }
		}
	},
	"schemas": {
		"rule_key_arguments": {
			"name": "RuleKeyArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" },
				"key": { "name": "key", "description": "Rule key", "type": "string" }
			}
		}
	}
}`)

var policyCreateInputJSON = []byte(`{
	"name": "policy_create_input",
	"description": "Create or update rule within a policy",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Policy name argument",
			"type": "object",
			"schema": { "id": "policy_create_arguments" }
		},
		"payload": {
			"name": "payload",
			"description": "Rule payload",
			"type": "object",
			"schema": { "id": "rule_create_payload" }
		}
	},
	"schemas": {
		"policy_create_arguments": {
			"name": "PolicyCreateArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		},
		"rule_create_payload": {
			"name": "RuleCreatePayload",
			"fields": {
				"key": { "name": "key", "description": "Rule key", "type": "string" },
				"rule_type": { "name": "rule_type", "description": "Type of rule (allow/deny)", "type": "string" },
				"syntax": { "name": "syntax", "description": "Rule expression syntax", "type": "string" },
				"expression": { "name": "expression", "description": "Rule expression", "type": "string" },
				"rules": { "name": "rules", "description": "Nested rule nodes", "type": "string" },
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"protected": { "name": "protected", "description": "Whether the rule is protected", "type": "boolean" }
			}
		}
	}
}`)

var policyOperationOutputJSON = []byte(`{
	"name": "policy_operation_output",
	"description": "Policy operation with intents and rule references",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Policy operation document",
			"type": "object",
			"schema": { "id": "policy_operation" }
		}
	},
	"schemas": {
		"policy_operation": {
			"name": "PolicyOperation",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" },
				"rule_key": { "name": "rule_key", "description": "Associated rule key", "type": "string" },
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"intent_type": { "name": "intent_type", "description": "Intent type for this operation", "type": "string" }
			}
		}
	}
}`)

var policyRuleOutputJSON = []byte(`{
	"name": "policy_rule_output",
	"description": "Policy rule with expression details",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Policy rule document",
			"type": "object",
			"schema": { "id": "policy_rule" }
		}
	},
	"schemas": {
		"policy_rule": {
			"name": "PolicyRule",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" },
				"rule_key": { "name": "rule_key", "description": "Rule key", "type": "string" },
				"rule_type": { "name": "rule_type", "description": "Type of rule (allow/deny)", "type": "string" },
				"syntax": { "name": "syntax", "description": "Rule expression syntax", "type": "string" },
				"expression": { "name": "expression", "description": "Rule expression", "type": "string" },
				"rules": {
					"name": "rules",
					"description": "Nested rule nodes for composite rules",
					"type": "object",
					"schema": { "id": "rule_node" }
				},
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"protected": { "name": "protected", "description": "Whether the rule is protected", "type": "boolean" }
			}
		},
		"rule_node": {
			"name": "RuleNode",
			"fields": {
				"operator": { "name": "operator", "description": "Logical operator (and/or/not)", "type": "string" },
				"rules": {
					"name": "rules",
					"description": "Array of nested rule nodes",
					"type": "array",
					"schema": { "id": "rule_node" }
				},
				"expression": { "name": "expression", "description": "Leaf rule expression", "type": "string" },
				"syntax": { "name": "syntax", "description": "Leaf rule syntax", "type": "string" }
			}
		}
	}
}`)

var policyValidateInputJSON = []byte(`{
	"name": "policy_validate_input",
	"description": "Policy validation request with operation and context",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Policy name argument",
			"type": "object",
			"schema": { "id": "policy_validate_arguments" }
		},
		"payload": {
			"name": "payload",
			"description": "Validation context",
			"type": "object",
			"schema": { "id": "policy_validate_payload" }
		}
	},
	"schemas": {
		"policy_validate_arguments": {
			"name": "PolicyValidateArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		},
		"policy_validate_payload": {
			"name": "PolicyValidatePayload",
			"fields": {
				"operation": { "name": "operation", "description": "Operation to validate", "type": "string" },
				"context": { "name": "context", "description": "Validation context data", "type": "record" }
			}
		}
	}
}`)

var policyValidateOutputJSON = []byte(`{
	"name": "policy_validate_output",
	"description": "Policy validation result",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Validation result document",
			"type": "object",
			"schema": { "id": "policy_validate_result" }
		}
	},
	"schemas": {
		"policy_validate_result": {
			"name": "PolicyValidateResult",
			"fields": {
				"valid": { "name": "valid", "description": "Whether the request is permitted", "type": "boolean" },
				"result": { "name": "result", "description": "Validation result detail", "type": "string" }
			}
		}
	}
}`)

var policyReloadOutputJSON = []byte(`{
	"name": "policy_reload_output",
	"description": "Reload result with operation and rule counts",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Reload result document",
			"type": "object",
			"schema": { "id": "policy_reload_result" }
		}
	},
	"schemas": {
		"policy_reload_result": {
			"name": "PolicyReloadResult",
			"fields": {
				"operations": { "name": "operations", "description": "Number of operations loaded", "type": "integer" },
				"rules": { "name": "rules", "description": "Number of rules loaded", "type": "integer" }
			}
		}
	}
}`)

var messageOutputJSON = []byte(`{
	"name": "policy_message",
	"description": "A simple status message response",
	"version": "1.0.0",
	"fields": {
		"message": { "name": "message", "description": "Human-readable status message", "type": "string" }
	}
}`)

var policyOperationGetInputJSON = []byte(`{
	"name": "policy_operation_get_input",
	"description": "Policy operation name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		}
	}
}`)

var policyRuleGetInputJSON = []byte(`{
	"name": "policy_rule_get_input",
	"description": "Policy rule name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		}
	}
}`)

var policyOperationUpsertInputJSON = []byte(`{
	"name": "policy_operation_upsert_input",
	"description": "Create or update a policy operation",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		},
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "operation_upsert_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		},
		"operation_upsert_payload": {
			"name": "OperationUpsertPayload",
			"fields": {
				"ruleKey": { "name": "ruleKey", "description": "Associated rule key", "type": "string" },
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"intentType": { "name": "intentType", "description": "Intent type for this operation", "type": "string" }
			}
		}
	}
}`)

var policyOperationDeleteInputJSON = []byte(`{
	"name": "policy_operation_delete_input",
	"description": "Policy operation name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		}
	}
}`)

var policyRuleUpsertInputJSON = []byte(`{
	"name": "policy_rule_upsert_input",
	"description": "Create or update a policy rule",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		},
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "rule_upsert_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		},
		"rule_upsert_payload": {
			"name": "RuleUpsertPayload",
			"fields": {
				"ruleType": { "name": "ruleType", "description": "Type of rule (allow/deny)", "type": "string" },
				"syntax": { "name": "syntax", "description": "Rule expression syntax", "type": "string" },
				"expression": { "name": "expression", "description": "Rule expression", "type": "string" },
				"rules": { "name": "rules", "description": "Nested rule nodes (JSON)", "type": "record" },
				"description": { "name": "description", "description": "Human-readable description", "type": "string" },
				"protected": { "name": "protected", "description": "Whether the rule is protected", "type": "boolean" }
			}
		}
	}
}`)

var policyRuleDeleteInputJSON = []byte(`{
	"name": "policy_rule_delete_input",
	"description": "Policy rule name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "policy_name_arguments" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "description": "Policy name", "type": "string" }
			}
		}
	}
}`)
