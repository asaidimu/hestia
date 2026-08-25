package system

import (
	"github.com/asaidimu/hestia/core/runtime"
	apikeysvc "github.com/asaidimu/hestia/core/system/apikeys"
	"github.com/asaidimu/hestia/core/system/audit"
	authsvc "github.com/asaidimu/hestia/core/system/auth"
	blobsvc "github.com/asaidimu/hestia/core/system/blobs"
	collectionsvc "github.com/asaidimu/hestia/core/system/collections"
	notificationsvc "github.com/asaidimu/hestia/core/system/notifications"
	operationsvc "github.com/asaidimu/hestia/core/system/operations"
	"github.com/asaidimu/hestia/core/system/policies"
	policiesvc "github.com/asaidimu/hestia/core/system/policies"
	schedulesvc "github.com/asaidimu/hestia/core/system/schedules"
	settingsvc "github.com/asaidimu/hestia/core/system/settings"
	updatessvc "github.com/asaidimu/hestia/core/system/updates"
	usersvc "github.com/asaidimu/hestia/core/system/users"
)

var allDefaultPolicyBindings = func() []policies.Policy {
	var bindings []policies.Policy
	for _, op := range allPolicyBindings {
		ruleName := op.RuleKey
		if ruleName == "" {
			ruleName = "administrator"
		}
		p := policies.Policy{
			Operation: op.Name,
			Rule:      ruleName,
			Enabled:   true,
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
	all = append(all, audit.StreamPolicyBinding()...)
	all = append(all, blobsvc.Policies()...)
	all = append(all, collectionsvc.Policies()...)
	all = append(all, usersvc.Policies()...)
	all = append(all, apikeysvc.Policies()...)
	all = append(all, settingsvc.Policies()...)
	all = append(all, notificationsvc.Policies()...)
	all = append(all, schedulesvc.Policies()...)
	all = append(all, policiesvc.Policies()...)
	all = append(all, operationsvc.Policies()...)
	all = append(all, authsvc.Policies()...)
	all = append(all, updatessvc.Policies()...)
	return all
}()

func collectAllPolicyBindings() []policies.Binding {
	return allPolicyBindings
}
