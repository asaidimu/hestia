package blobs

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.PolicyOperation {
	return []policies.PolicyOperation{
		{Name: "system:blobs:namespace:list", RuleKey: "administrator", Description: "List blob namespaces"},
		{Name: "system:blobs:namespace:create", RuleKey: "administrator", Description: "Create a blob namespace"},
		{Name: "system:blobs:namespace:delete", RuleKey: "administrator", Description: "Delete a blob namespace"},
		{Name: "system:blobs:blob:list", RuleKey: "blob", Description: "List blobs in a namespace"},
		{Name: "system:blobs:blob:head", RuleKey: "blob", Description: "Get blob metadata"},
		{Name: "system:blobs:blob:upload", RuleKey: "administrator", Description: "Upload a blob"},
		{Name: "system:blobs:blob:download", RuleKey: "blob", Description: "Download a blob"},
		{Name: "system:blobs:blob:delete", RuleKey: "administrator", Description: "Delete a blob"},
	}
}
