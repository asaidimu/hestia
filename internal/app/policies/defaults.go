package policies

func DefaultOperations() []PolicyOperation {
	return []PolicyOperation{
		{Name: "system:policies:operation:get", RuleKey: "administrator", Description: "Get a policy operation by name"},
		{Name: "system:policies:operation:upsert", RuleKey: "administrator", Description: "Create or update a policy operation"},
		{Name: "system:policies:operation:delete", RuleKey: "administrator", Description: "Delete a policy operation"},
		{Name: "system:policies:rule:get", RuleKey: "administrator", Description: "Get a policy rule by name"},
		{Name: "system:policies:rule:upsert", RuleKey: "administrator", Description: "Create or update a policy rule"},
		{Name: "system:policies:rule:delete", RuleKey: "administrator", Description: "Delete a policy rule"},
		{Name: "system:policies:rule:validate", RuleKey: "administrator", Description: "Validate a CEL rule expression"},
		{Name: "system:policies:rule:reload", RuleKey: "administrator", Description: "Reload policies from database"},
	}
}

func DefaultRules() []PolicyRule {
	return []PolicyRule{
		{Name: "public", RuleType: "simple", Syntax: "cel", Expression: "true", Description: "Public access — no authentication required"},
		{Name: "authenticated", RuleType: "simple", Syntax: "cel", Expression: "identity != null", Description: "Any authenticated user"},
		{Name: "password_reset", RuleType: "simple", Syntax: "cel", Expression: "identity != null && identity.token_type == 'password_reset'", Description: "Valid password-reset token type"},
		{Name: "administrator", RuleType: "simple", Syntax: "cel", Expression: "identity != null && 'administrator' in identity.permissions", Description: "Administrator-only access"},
		{Name: "blob", RuleType: "composite", Rules: &RuleNode{
			Operator: "OR",
			Conditions: []RuleNode{
				{Type: "cel", Expression: "resource.public == true"},
				{Type: "ref", Name: "administrator"},
			},
		}, Description: "Blob access — public namespace or admin"},
	}
}
