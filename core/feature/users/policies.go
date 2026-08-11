package users

import "github.com/asaidimu/hestia/core/feature/policies"

func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:users:user:create", RuleKey: "administrator", Description: "Create new user"},
		{Name: "system:users:user:query", RuleKey: "administrator", Description: "Query users with QDSL"},
		{Name: "system:users:user:get", RuleKey: "administrator", Description: "Get user by ID"},
		{Name: "system:users:user:update", RuleKey: "administrator", Description: "Update user profile"},
		{Name: "system:users:password:change", RuleKey: "administrator", Description: "Change account password"},
		{Name: "system:users:user:delete", RuleKey: "administrator", Description: "Delete user account"},
	}
}
