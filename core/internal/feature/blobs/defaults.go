package blobs

import "github.com/asaidimu/hestia/core/internal/feature/policies"

func DefaultOperations() []policies.Operation {
	return []policies.Operation{
		{Name: "system:blobs:namespace:list", RuleKey: "administrator", Description: "List blob namespaces"},
		{Name: "system:blobs:namespace:create", RuleKey: "administrator", Description: "Create a blob namespace"},
		{Name: "system:blobs:namespace:delete", RuleKey: "administrator", Description: "Delete a blob namespace"},
	}
}
