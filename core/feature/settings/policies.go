package settings

import "github.com/asaidimu/hestia/core/feature/policies"

func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:settings:list", RuleKey: "administrator", Description: "List all settings"},
		{Name: "system:settings:get", RuleKey: "administrator", Description: "Get a setting by key"},
		{Name: "system:settings:set", RuleKey: "administrator", Description: "Create or update a setting"},
		{Name: "system:settings:delete", RuleKey: "administrator", Description: "Delete a setting"},
	}
}
