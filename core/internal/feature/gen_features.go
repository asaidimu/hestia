package feature

import (
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/audit"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/blobs"
	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/internal/feature/notifications"
	"github.com/asaidimu/hestia/core/internal/feature/operations"
	"github.com/asaidimu/hestia/core/internal/feature/policies"
	"github.com/asaidimu/hestia/core/internal/feature/schedules"
	"github.com/asaidimu/hestia/core/internal/feature/settings"
	"github.com/asaidimu/hestia/core/internal/feature/users"
)

var allDefaultPolicyBindings = func() []policies.Policy {
	var bindings []policies.Policy
	for _, op := range allKnownOperations {
		ruleName := op.RuleKey
		if ruleName == "" {
			ruleName = "administrator"
		}
		bindings = append(bindings, policies.Policy{
			OperationName: op.Name,
			RuleName:      ruleName,
			Enabled:       true,
		})
	}
	return bindings
}()

var allKnownOperations = func() []policies.Operation {
	var all []policies.Operation
	all = append(all, apikeys.DefaultOperations()...)
	all = append(all, audit.DefaultOperations()...)
	all = append(all, auth.DefaultOperations()...)
	all = append(all, blobs.DefaultOperations()...)
	all = append(all, collections.DefaultOperations()...)
	all = append(all, operations.DefaultOperations()...)
	all = append(all, policies.DefaultOperations()...)
	all = append(all, notifications.DefaultOperations()...)
	all = append(all, schedules.DefaultOperations()...)
	all = append(all, settings.DefaultOperations()...)
	all = append(all, users.DefaultOperations()...)
	return all
}()

func collectAllKnownOperations() []policies.Operation {
	return allKnownOperations
}


