package schedules

import "github.com/asaidimu/hestia/core/feature/policies"

func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:schedules:schedule:create", RuleKey: "authenticated", Description: "Create a schedule"},
		{Name: "system:schedules:schedule:list", RuleKey: "authenticated", Description: "List schedules"},
		{Name: "system:schedules:schedule:get", RuleKey: "authenticated", Description: "Get a schedule"},
		{Name: "system:schedules:schedule:update", RuleKey: "authenticated", Description: "Update a schedule"},
		{Name: "system:schedules:schedule:delete", RuleKey: "authenticated", Description: "Delete a schedule"},
	}
}
