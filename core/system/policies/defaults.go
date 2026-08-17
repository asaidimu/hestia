package policies

func DefaultRules() []PolicyRule {
	return []PolicyRule{
		{Name: "public", RuleType: "simple", Syntax: "cel", Expression: "true", Description: "Public access — no authentication required"},
		{Name: "authenticated", RuleType: "simple", Syntax: "cel", Expression: "identity != null", Description: "Any authenticated user"},
		{Name: "password_reset", RuleType: "simple", Syntax: "cel", Expression: "identity != null && identity.token_type == 'password_reset'", Description: "Valid password-reset token type"},
		{Name: "administrator", RuleType: "simple", Syntax: "cel", Expression: "identity != null && 'administrator' in identity.permissions", Description: "Administrator-only access"},
	}
}