package schedules

import "github.com/asaidimu/hestia/core/internal/feature/policies"

func DefaultOperations() []policies.Operation {
	return []policies.Operation{
		{Name: "system:schedules:schedule:create", RuleKey: "authenticated", Description: "Schedule a message"},
		{Name: "system:schedules:schedule:list", RuleKey: "authenticated", Description: "List scheduled messages"},
		{Name: "system:schedules:schedule:get", RuleKey: "authenticated", Description: "Get a scheduled message"},
		{Name: "system:schedules:schedule:delete", RuleKey: "authenticated", Description: "Delete a scheduled message"},
	}
}
