package policies

import (
	"context"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/runtime"
)

// PolicyStoreAdapter implements collections.PolicyStore using the system
// module's PolicyModel.  Every write through the PolicyModel goes through
// the LiveRepository-backed base.Collection, which auto-syncs the in-memory
// cache — no manual cache updates needed.
type PolicyStoreAdapter struct {
	policyModel *PolicyModel
	permMgr     runtime.ReloadablePermissionManager
	liveRules   iam.RuleSet[iam.FunctionRule]
}

func NewPolicyStoreAdapter(policyModel *PolicyModel, permMgr runtime.ReloadablePermissionManager, liveRules iam.RuleSet[iam.FunctionRule]) *PolicyStoreAdapter {
	return &PolicyStoreAdapter{
		policyModel: policyModel,
		permMgr:     permMgr,
		liveRules:   liveRules,
	}
}

// EnsureBinding creates or updates a policy for the given binding.
func (a *PolicyStoreAdapter) EnsureBinding(ctx context.Context, name, ruleName string) error {
	_, err := a.policyModel.GetPolicyForOperation(ctx, name)
	if err != nil {
		policy := Policy{
			Operation: name,
			Rule:      ruleName,
			Enabled:   true,
			Protected: false,
		}
		_, err := a.policyModel.CreatePolicy(ctx, policy)
		if err != nil {
			return err
		}
		return nil
	}

	_, err = a.policyModel.UpdatePolicyRule(ctx, name, ruleName)
	if err != nil {
		return err
	}
	return nil
}

// DeleteBinding hard-deletes the policy for the given binding.
func (a *PolicyStoreAdapter) DeleteBinding(ctx context.Context, name string) error {
	return a.policyModel.DeletePolicy(ctx, name)
}

func (a *PolicyStoreAdapter) ReloadPolicies(ctx context.Context) error {
	return a.permMgr.Reload(ctx)
}
