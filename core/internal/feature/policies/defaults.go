package policies

func DefaultOperations() []Operation {
	return []Operation{
		{Name: "system:policies:operation:get", Description: "Get a policy operation by name"},
		{Name: "system:policies:rule:get", Description: "Get a policy rule by name"},
		{Name: "system:policies:rule:validate", Description: "Validate a CEL rule expression"},
		{Name: "system:policies:operation:list", Description: "List policy operations"},
		{Name: "system:policies:rule:list", Description: "List policy rules"},
		{Name: "system:policies:policy:list", Description: "List policy bindings"},
		{Name: "system:policies:policy:create", Description: "Create a policy binding"},
		{Name: "system:policies:policy:update", Description: "Update a policy — set ruleName, enabled, or both"},
		{Name: "system:policies:rule:create", Description: "Create a policy rule"},
		{Name: "system:policies:rule:update", Description: "Update a policy rule"},
		{Name: "system:policies:rule:delete", Description: "Delete a policy rule"},
		{Name: "system:policies:reload", Description: "Reload policies from database"},
	}
}

func DefaultRules() []PolicyRule {
	return []PolicyRule{
		{Name: "public", RuleType: "simple", Syntax: "cel", Expression: "true", Description: "Public access — no authentication required"},
		{Name: "authenticated", RuleType: "simple", Syntax: "cel", Expression: "identity != null", Description: "Any authenticated user"},
		{Name: "password_reset", RuleType: "simple", Syntax: "cel", Expression: "identity != null && identity.token_type == 'password_reset'", Description: "Valid password-reset token type"},
		{Name: "administrator", RuleType: "simple", Syntax: "cel", Expression: "identity != null && 'administrator' in identity.permissions", Description: "Administrator-only access"},
	}
}
