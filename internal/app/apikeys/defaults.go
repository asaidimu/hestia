package apikeys

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.Operation {
	return []policies.Operation{
		{Name: "system:apikeys:key:list", RuleKey: "administrator", Description: "List own API keys"},
		{Name: "system:apikeys:key:get", RuleKey: "administrator", Description: "Get own API key details"},
		{Name: "system:apikeys:key:create", RuleKey: "administrator", Description: "Create a new API key"},
		{Name: "system:apikeys:key:update", RuleKey: "administrator", Description: "Update API key metadata"},
		{Name: "system:apikeys:key:delete", RuleKey: "administrator", Description: "Delete an API key"},
		{Name: "system:apikeys:key:rotate", RuleKey: "administrator", Description: "Rotate API key material"},
	}
}
