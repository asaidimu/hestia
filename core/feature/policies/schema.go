package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_policyValidateOutput     = dispatch.MustFromJSON(policyValidateOutputJSON)
	_policyReloadOutput       = dispatch.MustFromJSON(policyReloadOutputJSON)
	_policyBindingOutput      = dispatch.MustFromJSON(policyBindingOutputJSON)
	_policyRuleOutput         = dispatch.MustFromJSON(policyRuleOutputJSON)
	_policyOutput             = dispatch.MustFromJSON(policyOutputJSON)
	_policyListBindingsOutput = dispatch.MustFromJSON(policyListBindingsOutputJSON)
	_policyListRulesOutput    = dispatch.MustFromJSON(policyListRulesOutputJSON)
	_policyListPoliciesOutput = dispatch.MustFromJSON(policyListPoliciesOutputJSON)
)

func policyValidateOutputSchema() *definition.Schema     { return _policyValidateOutput }
func policyReloadOutputSchema() *definition.Schema       { return _policyReloadOutput }
func policyBindingOutputSchema() *definition.Schema      { return _policyBindingOutput }
func policyRuleOutputSchema() *definition.Schema         { return _policyRuleOutput }
func policyOutputSchema() *definition.Schema             { return _policyOutput }
func policyListBindingsOutputSchema() *definition.Schema { return _policyListBindingsOutput }
func policyListRulesOutputSchema() *definition.Schema    { return _policyListRulesOutput }
func policyListPoliciesOutputSchema() *definition.Schema { return _policyListPoliciesOutput }

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

var policyBindingOutputJSON = []byte(`{
	"name": "policy_binding_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "binding_info" }
		}
	},
	"schemas": {
		"binding_info": {
			"name": "Binding",
			"fields": {
				"name": { "name": "name", "type": "string" },
				"description": { "name": "description", "type": "string" }
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
				"protected": { "name": "protected", "type": "boolean" },
				"rateLimit": { "name": "rateLimit", "type": "object", "schema": { "id": "rate_limit_config" } },
				"throttle": { "name": "throttle", "type": "object", "schema": { "id": "throttle_config" } }
			}
		}
	}
}`)

var policyListBindingsOutputJSON = []byte(`{
	"name": "policy_list_bindings_output",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "binding_list" }
		}
	},
	"schemas": {
		"binding_list": {
			"name": "BindingList",
			"fields": {
				"bindings": {
					"name": "bindings",
					"type": "array",
					"schema": { "id": "binding_info" }
				}
			}
		},
		"binding_info": {
			"name": "Binding",
			"fields": {
				"name": { "name": "name", "type": "string" },
				"description": { "name": "description", "type": "string" }
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
				"protected": { "name": "protected", "type": "boolean" },
				"rateLimit": { "name": "rateLimit", "type": "object", "schema": { "id": "rate_limit_config" } },
				"throttle": { "name": "throttle", "type": "object", "schema": { "id": "throttle_config" } }
			}
		}
	}
}`)
