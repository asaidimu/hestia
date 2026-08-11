package blobs

import "github.com/asaidimu/hestia/core/feature/policies"

func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:blobs:namespace:list", RuleKey: "administrator", Description: "List blob namespaces"},
		{Name: "system:blobs:namespace:create", RuleKey: "administrator", Description: "Create a blob namespace"},
		{Name: "system:blobs:namespace:delete", RuleKey: "administrator", Description: "Delete a blob namespace"},
		{Name: "system:blobs:blob:list", RuleKey: "administrator", Description: "List blobs in a namespace"},
		{Name: "system:blobs:blob:head", RuleKey: "administrator", Description: "Get blob metadata"},
		{Name: "system:blobs:blob:upload", RuleKey: "administrator", Description: "Upload a blob"},
		{Name: "system:blobs:blob:download", RuleKey: "administrator", Description: "Download a blob"},
		{Name: "system:blobs:blob:delete", RuleKey: "administrator", Description: "Delete a blob"},
		{Name: "system:blobs:blob:update", RuleKey: "administrator", Description: "Update blob metadata"},
	}
}
