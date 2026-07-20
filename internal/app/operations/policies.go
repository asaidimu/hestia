package operations

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.Operation {
	return []policies.Operation{
		{Name: "system:core:heartbeat", RuleKey: "authenticated", Description: "Session keepalive — does not count as a health check"},
		{Name: "system:core:health:check", RuleKey: "public", Description: "Check system health and bootstrap status"},
		{Name: "system:core:bootstrap:mark", RuleKey: "public", Description: "Mark system as bootstrapped"},
		{Name: "system:core:admin:reset", RuleKey: "administrator", Description: "Reset system to initial state"},
		{Name: "system:core:audit:log", RuleKey: "authenticated", Description: "Log an API access entry"},
		{Name: "system:core:docs:list", RuleKey: "public", Description: "List all registered API endpoints with metadata"},
		{Name: "system:core:capability:list", RuleKey: "administrator", Description: "List all registered commands and queries with descriptions and enabled status"},
		{Name: "system:core:capability:set", RuleKey: "administrator", Description: "Enable or disable a registered command or query"},
	}
}
