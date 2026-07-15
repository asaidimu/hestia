package auth

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.PolicyOperation {
	return []policies.PolicyOperation{
		{Name: "system:auth:session:create", RuleKey: "public", Description: "Authenticate and receive tokens"},
		{Name: "system:auth:user:register", RuleKey: "administrator", Description: "Register new user"},
		// TODO: Fix authentication flow so that this rule is secure. A refresh
		// token should be embeded within cookies
		{Name: "system:auth:session:refresh", RuleKey: "public", Description: "Refresh access token"},
		{Name: "system:auth:session:delete", RuleKey: "authenticated", Description: "Logout / revoke current token"},
		{Name: "system:auth:password:reset", RuleKey: "authenticated", Description: "Request password reset email"},
		{Name: "system:auth:password:confirm", RuleKey: "password_reset", Description: "Confirm password reset with token"},
		{Name: "system:auth:token:validate", RuleKey: "public", Description: "Validate a JWT access token and return claims"},
		{Name: "system:auth:token:check", RuleKey: "public", Description: "Check if a token has been blocklisted"},
		{Name: "system:auth:session:validate", RuleKey: "public", Description: "Validate a session token"},
		{Name: "system:auth:apikey:validate", RuleKey: "public", Description: "Authenticate via API key"},
		{Name: "system:auth:bootstrap:password:set", RuleKey: "administrator", Description: "Change the bootstrap admin password"},
	}
}
