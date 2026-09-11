package policies

func DefaultRules() []PolicyRule {
	return []PolicyRule{
		{Name: "public", RuleType: "simple", Syntax: "cel", Expression: "true", Description: "Public access — no authentication required"},
		{Name: "authenticated", RuleType: "simple", Syntax: "cel", Expression: "identity != null", Description: "Any authenticated user"},
		{Name: "password_reset", RuleType: "simple", Syntax: "cel", Expression: "identity != null && identity.token_type == 'password_reset'", Description: "Valid password-reset token type"},
		{Name: "root", RuleType: "simple", Syntax: "cel", Expression: "identity != null && 'root' in identity.permissions", Description: "Root-only access"},
	}
}