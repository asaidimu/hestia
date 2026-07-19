package auth

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.Operation {
	return []policies.Operation{
		{Name: "system:auth:session:create", RuleKey: "public", Description: "Authenticate and receive a session token"},
		{Name: "system:auth:user:register", RuleKey: "administrator", Description: "Register new user"},
		{Name: "system:auth:session:delete", RuleKey: "authenticated", Description: "Logout"},
		{Name: "system:auth:password:reset", RuleKey: "authenticated", Description: "Request password reset email"},
		{Name: "system:auth:password:confirm", RuleKey: "public", Description: "Confirm password reset with token"},
		{Name: "system:auth:session:validate", RuleKey: "public", Description: "Validate a session token"},
		{Name: "system:auth:apikey:validate", RuleKey: "public", Description: "Authenticate via API key"},
		{Name: "system:auth:bootstrap:password:set", RuleKey: "administrator", Description: "Change the bootstrap admin password"},
	}
}
