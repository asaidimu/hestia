package collections

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.OperationPolicy {
	return []policies.OperationPolicy{
		{Name: "system:collections:collection:list", RuleKey: "administrator", Description: "List all dynamic collections"},
		{Name: "system:collections:collection:get", RuleKey: "administrator", Description: "Get a dynamic collection by name"},
		{Name: "system:collections:collection:create", RuleKey: "administrator", Description: "Create a new dynamic collection"},
		{Name: "system:collections:collection:delete", RuleKey: "administrator", Description: "Delete a dynamic collection"},
		{Name: "system:collections:document:query", RuleKey: "administrator", Description: "Query a dynamic collection with QDSL"},
		{Name: "system:collections:document:create", RuleKey: "administrator", Description: "Create a document in a dynamic collection"},
		{Name: "system:collections:document:get", RuleKey: "administrator", Description: "Get a document from a dynamic collection"},
		{Name: "system:collections:document:update", RuleKey: "administrator", Description: "Update a document in a dynamic collection"},
		{Name: "system:collections:document:delete", RuleKey: "administrator", Description: "Delete a document from a dynamic collection"},
		{Name: "system:collections:_user:read", RuleKey: "administrator", Description: "Query users collection with QDSL"},
		{Name: "system:collections:_api_key:read", RuleKey: "administrator", Description: "Query API keys collection with QDSL"},
		{Name: "system:collections:_operation_policy:read", RuleKey: "administrator", Description: "Query policy operations collection with QDSL"},
		{Name: "system:collections:_iam_rule:read", RuleKey: "administrator", Description: "Query policy rules collection with QDSL"},
		{Name: "system:collections:_access_log:read", RuleKey: "administrator", Description: "Query access logs collection with QDSL"},
	}
}
