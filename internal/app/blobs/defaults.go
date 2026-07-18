package blobs

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.Operation {
	return []policies.Operation{
		{Name: "system:blobs:namespace:list", RuleKey: "administrator", Description: "List blob namespaces"},
		{Name: "system:blobs:namespace:create", RuleKey: "administrator", Description: "Create a blob namespace"},
		{Name: "system:blobs:namespace:delete", RuleKey: "administrator", Description: "Delete a blob namespace"},
		{Name: "system:blobs:blob:list", Description: "List blobs in a namespace", RuleKey: "blob"},
		{Name: "system:blobs:blob:head", Description: "Get blob metadata", RuleKey: "blob"},
		{Name: "system:blobs:blob:upload", RuleKey: "administrator", Description: "Upload a blob"},
		{Name: "system:blobs:blob:download", Description: "Download a blob", RuleKey: "blob"},
		{Name: "system:blobs:blob:delete", RuleKey: "administrator", Description: "Delete a blob"},
		{Name: "system:blobs:blob:update", Description: "Update blob metadata", RuleKey: "blob"},
	}
}
