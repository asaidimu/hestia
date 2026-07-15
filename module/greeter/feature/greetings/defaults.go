package greetings

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.PolicyOperation {
	return []policies.PolicyOperation{
		{Name: "greeter:greetings:salutation:create", RuleKey: "administrator", Description: "Create a greeting salutation"},
		{Name: "greeter:greetings:salutation:get", RuleKey: "administrator", Description: "Get a salutation by ID"},
		{Name: "greeter:greetings:salutation:list", RuleKey: "administrator", Description: "List all salutations"},
		{Name: "greeter:greetings:greeting:generate", RuleKey: "administrator", Description: "Generate a personalized greeting"},
	}
}
