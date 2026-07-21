package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/schema"
)

var (
	_policyNameInput              = schema.MustFromJSON(policyNameInputJSON)
	_policyOperationGetInput      = schema.MustFromJSON(policyOperationGetInputJSON)
	_policyRuleGetInput           = schema.MustFromJSON(policyRuleGetInputJSON)
	_policyRuleDeleteInput        = schema.MustFromJSON(policyRuleDeleteInputJSON)
	_policyRuleCreateInput        = schema.MustFromJSON(policyRuleCreateInputJSON)
	_policyRuleUpdateInput        = schema.MustFromJSON(policyRuleUpdateInputJSON)
	_policyCreateInput            = schema.MustFromJSON(policyCreateInputJSON)
	_policyUpdateInput            = schema.MustFromJSON(policyUpdateInputJSON)
	_policyValidateInput          = schema.MustFromJSON(policyValidateInputJSON)
	_policyValidateOutput         = schema.MustFromJSON(policyValidateOutputJSON)
	_policyReloadOutput           = schema.MustFromJSON(policyReloadOutputJSON)
	_policyOperationOutput        = schema.MustFromJSON(policyOperationOutputJSON)
	_policyRuleOutput             = schema.MustFromJSON(policyRuleOutputJSON)
	_policyOutput                 = schema.MustFromJSON(policyOutputJSON)
	_policyListOperationsOutput   = schema.MustFromJSON(policyListOperationsOutputJSON)
	_policyListRulesOutput        = schema.MustFromJSON(policyListRulesOutputJSON)
	_policyListPoliciesOutput     = schema.MustFromJSON(policyListPoliciesOutputJSON)
)

func policyNameInputSchema() *definition.Schema               { return _policyNameInput }
func policyOperationGetInputSchema() *definition.Schema       { return _policyOperationGetInput }
func policyRuleGetInputSchema() *definition.Schema            { return _policyRuleGetInput }
func policyRuleDeleteInputSchema() *definition.Schema         { return _policyRuleDeleteInput }
func policyRuleCreateInputSchema() *definition.Schema         { return _policyRuleCreateInput }
func policyRuleUpdateInputSchema() *definition.Schema         { return _policyRuleUpdateInput }
func policyCreateInputSchema() *definition.Schema             { return _policyCreateInput }
func policyUpdateInputSchema() *definition.Schema                { return _policyUpdateInput }
func policyValidateInputSchema() *definition.Schema           { return _policyValidateInput }
func policyValidateOutputSchema() *definition.Schema          { return _policyValidateOutput }
func policyReloadOutputSchema() *definition.Schema            { return _policyReloadOutput }
func policyOperationOutputSchema() *definition.Schema         { return _policyOperationOutput }
func policyRuleOutputSchema() *definition.Schema              { return _policyRuleOutput }
func policyOutputSchema() *definition.Schema                  { return _policyOutput }
func policyListOperationsOutputSchema() *definition.Schema    { return _policyListOperationsOutput }
func policyListRulesOutputSchema() *definition.Schema         { return _policyListRulesOutput }
func policyListPoliciesOutputSchema() *definition.Schema      { return _policyListPoliciesOutput }

var policyNameInputJSON = []byte(`{
	"name": "policy_name_input",
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
				"name": { "name": "name", "type": "string" }
			}
		}
	}
}`)

var policyOperationGetInputJSON = []byte(`{
	"name": "policy_operation_get_input",
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
				"name": { "name": "name", "type": "string" }
			}
		}
	}
}`)

var policyRuleGetInputJSON = []byte(`{
	"name": "policy_rule_get_input",
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
				"name": { "name": "name", "type": "string" }
			}
		}
	}
}`)

var policyRuleDeleteInputJSON = []byte(`{
	"name": "policy_rule_delete_input",
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
				"name": { "name": "name", "type": "string" }
			}
		}
	}
}`)

var policyRuleCreateInputJSON = []byte(`{
	"name": "policy_rule_create_input",
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
			"schema": { "id": "rule_create_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "type": "string" }
			}
		},
		"rule_create_payload": {
			"name": "RuleCreatePayload",
			"fields": {
				"ruleType": { "name": "ruleType", "type": "string" },
				"syntax": { "name": "syntax", "type": "string" },
				"expression": { "name": "expression", "type": "string" },
				"rules": { "name": "rules", "type": "record" },
				"description": { "name": "description", "type": "string" }
			}
		}
	}
}`)

var policyRuleUpdateInputJSON = []byte(`{
	"name": "policy_rule_update_input",
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
			"schema": { "id": "rule_update_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "type": "string" }
			}
		},
		"rule_update_payload": {
			"name": "RuleUpdatePayload",
			"fields": {
				"ruleType": { "name": "ruleType", "type": "string" },
				"syntax": { "name": "syntax", "type": "string" },
				"expression": { "name": "expression", "type": "string" },
				"rules": { "name": "rules", "type": "record" },
				"description": { "name": "description", "type": "string" },
				"protected": { "name": "protected", "type": "boolean" }
			}
		}
	}
}`)

var policyCreateInputJSON = []byte(`{
	"name": "policy_create_input",
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
			"schema": { "id": "policy_create_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "type": "string" }
			}
		},
		"policy_create_payload": {
			"name": "PolicyCreatePayload",
			"fields": {
				"ruleName": { "name": "ruleName", "type": "string" }
			}
		}
	}
}`)

var policyUpdateInputJSON = []byte(`{
	"name": "policy_update_input",
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
			"schema": { "id": "policy_update_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "type": "string" }
			}
		},
		"policy_update_payload": {
			"name": "PolicyUpdatePayload",
			"fields": {
				"ruleName": { "name": "ruleName", "type": "string" },
				"enabled": { "name": "enabled", "type": "boolean" }
			}
		}
	}
}`)

var policyValidateInputJSON = []byte(`{
	"name": "policy_validate_input",
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
			"schema": { "id": "policy_validate_payload" }
		}
	},
	"schemas": {
		"policy_name_arguments": {
			"name": "PolicyNameArguments",
			"fields": {
				"name": { "name": "name", "type": "string" }
			}
		},
		"policy_validate_payload": {
			"name": "PolicyValidatePayload",
			"fields": {
				"rule": { "name": "rule", "type": "string" },
				"context": { "name": "context", "type": "object", "schema": { "id": "policy_validate_context" } }
			}
		},
		"policy_validate_context": {
			"name": "PolicyValidateContext",
			"fields": {
				"identity": { "name": "identity", "type": "record" },
				"resource": { "name": "resource", "type": "record" },
				"environment": { "name": "environment", "type": "record" }
			}
		}
	}
}`)

var policyValidateOutputJSON = []byte(`{
	"name": "policy_validate_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "policy_validate_result" }
		}
	},
	"schemas": {
		"policy_validate_result": {
			"name": "PolicyValidateResult",
			"fields": {
				"valid": { "name": "valid", "type": "boolean" },
				"result": { "name": "result", "type": "boolean" }
			}
		}
	}
}`)

var policyReloadOutputJSON = []byte(`{
	"name": "policy_reload_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "policy_reload_result" }
		}
	},
	"schemas": {
		"policy_reload_result": {
			"name": "PolicyReloadResult",
			"fields": {
				"operations": { "name": "operations", "type": "integer" },
				"rules": { "name": "rules", "type": "integer" }
			}
		}
	}
}`)

var policyOperationOutputJSON = []byte(`{
	"name": "policy_operation_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "operation_info" }
		}
	},
	"schemas": {
		"operation_info": {
			"name": "Operation",
			"fields": {
				"name": { "name": "name", "type": "string" },
				"description": { "name": "description", "type": "string" },
				"intentType": { "name": "intentType", "type": "string" }
			}
		}
	}
}`)

var policyRuleOutputJSON = []byte(`{
	"name": "policy_rule_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "policy_rule" }
		}
	},
	"schemas": {
		"policy_rule": {
			"name": "PolicyRule",
			"fields": {
				"id": { "name": "id", "type": "string" },
				"name": { "name": "name", "type": "string" },
				"ruleType": { "name": "ruleType", "type": "string" },
				"syntax": { "name": "syntax", "type": "string" },
				"expression": { "name": "expression", "type": "string" },
				"rules": { "name": "rules", "type": "object", "schema": { "id": "rule_node" } },
				"description": { "name": "description", "type": "string" },
				"protected": { "name": "protected", "type": "boolean" }
			}
		},
		"rule_node": {
			"name": "RuleNode",
			"fields": {
				"type": { "name": "type", "type": "string" },
				"name": { "name": "name", "type": "string" },
				"expression": { "name": "expression", "type": "string" },
				"operator": { "name": "operator", "type": "string" },
				"conditions": { "name": "conditions", "type": "array", "schema": { "id": "rule_node" } }
			}
		}
	}
}`)

var policyOutputJSON = []byte(`{
	"name": "policy_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "policy" }
		}
	},
	"schemas": {
		"policy": {
			"name": "Policy",
			"fields": {
				"id": { "name": "id", "type": "string" },
				"operationName": { "name": "operationName", "type": "string" },
				"ruleName": { "name": "ruleName", "type": "string" },
				"enabled": { "name": "enabled", "type": "boolean" },
				"protected": { "name": "protected", "type": "boolean" }
			}
		}
	}
}`)

var policyListOperationsOutputJSON = []byte(`{
	"name": "policy_list_operations_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "operation_list" }
		}
	},
	"schemas": {
		"operation_list": {
			"name": "OperationList",
			"fields": {
				"operations": {
					"name": "operations",
					"type": "array",
					"schema": { "id": "operation_info" }
				}
			}
		},
		"operation_info": {
			"name": "Operation",
			"fields": {
				"name": { "name": "name", "type": "string" },
				"description": { "name": "description", "type": "string" },
				"intentType": { "name": "intentType", "type": "string" }
			}
		}
	}
}`)

var policyListRulesOutputJSON = []byte(`{
	"name": "policy_list_rules_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "rule_list" }
		}
	},
	"schemas": {
		"rule_list": {
			"name": "RuleList",
			"fields": {
				"rules": {
					"name": "rules",
					"type": "array",
					"schema": { "id": "policy_rule" }
				}
			}
		},
		"policy_rule": {
			"name": "PolicyRule",
			"fields": {
				"id": { "name": "id", "type": "string" },
				"name": { "name": "name", "type": "string" },
				"ruleType": { "name": "ruleType", "type": "string" },
				"syntax": { "name": "syntax", "type": "string" },
				"expression": { "name": "expression", "type": "string" },
				"rules": { "name": "rules", "type": "object", "schema": { "id": "rule_node" } },
				"description": { "name": "description", "type": "string" },
				"protected": { "name": "protected", "type": "boolean" }
			}
		},
		"rule_node": {
			"name": "RuleNode",
			"fields": {
				"type": { "name": "type", "type": "string" },
				"name": { "name": "name", "type": "string" },
				"expression": { "name": "expression", "type": "string" },
				"operator": { "name": "operator", "type": "string" },
				"conditions": { "name": "conditions", "type": "array", "schema": { "id": "rule_node" } }
			}
		}
	}
}`)

var policyListPoliciesOutputJSON = []byte(`{
	"name": "policy_list_policies_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "policy_list" }
		}
	},
	"schemas": {
		"policy_list": {
			"name": "PolicyList",
			"fields": {
				"policies": {
					"name": "policies",
					"type": "array",
					"schema": { "id": "policy" }
				}
			}
		},
		"policy": {
			"name": "Policy",
			"fields": {
				"id": { "name": "id", "type": "string" },
				"operationName": { "name": "operationName", "type": "string" },
				"ruleName": { "name": "ruleName", "type": "string" },
				"enabled": { "name": "enabled", "type": "boolean" },
				"protected": { "name": "protected", "type": "boolean" }
			}
		}
	}
}`)
