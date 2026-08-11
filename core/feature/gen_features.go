package feature

import (
	"github.com/asaidimu/hestia/core/feature/apikeys"
	"github.com/asaidimu/hestia/core/feature/audit"
	"github.com/asaidimu/hestia/core/feature/auth"
	"github.com/asaidimu/hestia/core/feature/blobs"
	"github.com/asaidimu/hestia/core/feature/collections"
	"github.com/asaidimu/hestia/core/feature/notifications"
	"github.com/asaidimu/hestia/core/feature/operations"
	"github.com/asaidimu/hestia/core/feature/policies"
	"github.com/asaidimu/hestia/core/feature/schedules"
	"github.com/asaidimu/hestia/core/feature/settings"
	"github.com/asaidimu/hestia/core/feature/users"
	"github.com/asaidimu/hestia/core/runtime"
)

var allDefaultPolicyBindings = func() []policies.Policy {
	var bindings []policies.Policy
	for _, op := range allPolicyBindings {
		ruleName := op.RuleKey
		if ruleName == "" {
			ruleName = "administrator"
		}
		p := policies.Policy{
			OperationName: op.Name,
			RuleName:      ruleName,
			Enabled:       true,
		}
		switch op.Name {
		case "system:auth:session:create":
			p.RateLimit = &runtime.RateLimitPolicy{
				Enabled:  true,
				Identity: "ip",
				Capacity: 5,
				Refill:   5,
				Period:   60,
			}
			p.Throttle = &runtime.ThrottlePolicy{
				Limit:  10,
				Window: 300,
				Action: &runtime.ThrottleActionPolicy{
					Message: "system:users:user:disable",
					Input:   map[string]any{"arguments.id": "{{ .claims.user_id }}"},
				},
			}
		case "system:auth:password:reset":
			p.RateLimit = &runtime.RateLimitPolicy{
				Enabled:  true,
				Identity: "ip",
				Capacity: 3,
				Refill:   3,
				Period:   60,
			}
		case "system:users:user:create":
			p.RateLimit = &runtime.RateLimitPolicy{
				Enabled:  true,
				Identity: "ip",
				Capacity: 3,
				Refill:   3,
				Period:   60,
			}
		}
		bindings = append(bindings, p)
	}
	return bindings
}()

var allPolicyBindings = func() []policies.Binding {
	var all []policies.Binding
	all = append(all, apikeys.PolicyBindings()...)
	all = append(all, audit.PolicyBindings()...)
	all = append(all, auth.PolicyBindings()...)
	all = append(all, blobs.PolicyBindings()...)
	all = append(all, collections.PolicyBindings()...)
	all = append(all, operations.PolicyBindings()...)
	all = append(all, policies.PolicyBindings()...)
	all = append(all, notifications.PolicyBindings()...)
	all = append(all, schedules.PolicyBindings()...)
	all = append(all, settings.PolicyBindings()...)
	all = append(all, users.PolicyBindings()...)
	return all
}()

func collectAllPolicyBindings() []policies.Binding {
	return allPolicyBindings
}
