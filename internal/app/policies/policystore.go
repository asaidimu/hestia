package policies

import (
	"context"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
)

// PolicyStoreAdapter implements collections.PolicyStore using the system
// module's PolicyModel.  Every write through the PolicyModel goes through
// the LiveRepository-backed base.Collection, which auto-syncs the in-memory
// cache — no manual cache updates needed.
type PolicyStoreAdapter struct {
	policyModel *PolicyModel
	permMgr     core.ReloadablePermissionManager
	liveRules   iam.RuleSet[iam.FunctionRule]
}

func NewPolicyStoreAdapter(policyModel *PolicyModel, permMgr core.ReloadablePermissionManager, liveRules iam.RuleSet[iam.FunctionRule]) *PolicyStoreAdapter {
	return &PolicyStoreAdapter{
		policyModel: policyModel,
		permMgr:     permMgr,
		liveRules:   liveRules,
	}
}

// EnsureOperation creates or updates a policy for the given operation.
func (a *PolicyStoreAdapter) EnsureOperation(ctx context.Context, name, ruleName, intentType, description string) error {
	_, err := a.policyModel.GetPolicyForOperation(ctx, name)
	if err != nil {
		policy := Policy{
			OperationName: name,
			RuleName:      ruleName,
			Enabled:       true,
			Protected:     false,
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

// DeleteOperation disables the policy for the given operation instead of deleting it.
func (a *PolicyStoreAdapter) DeleteOperation(ctx context.Context, name string) error {
	_, err := a.policyModel.SetPolicyEnabled(ctx, name, false)
	if err != nil {
		return err
	}
	return nil
}

// ForceDeleteOperation disables the policy (same as DeleteOperation — no forced deletion).
func (a *PolicyStoreAdapter) ForceDeleteOperation(ctx context.Context, name string) error {
	return a.DeleteOperation(ctx, name)
}

func (a *PolicyStoreAdapter) EnsureRule(ctx context.Context, name, expr, description string) error {
	existing, err := a.policyModel.GetRule(ctx, name)
	if err == nil && existing.Name != "" {
		return nil
	}

	rule := PolicyRule{
		Name:        name,
		RuleType:    "simple",
		Syntax:      "cel",
		Expression:  expr,
		Description: description,
	}
	if _, err := a.policyModel.CreateRule(ctx, rule); err != nil {
		return err
	}
	return nil
}

func (a *PolicyStoreAdapter) DeleteRule(ctx context.Context, name string) error {
	if err := a.policyModel.DeleteRule(ctx, name); err != nil {
		return err
	}
	return nil
}

func (a *PolicyStoreAdapter) ForceDeleteRule(ctx context.Context, name string) error {
	return a.DeleteRule(ctx, name)
}

func (a *PolicyStoreAdapter) ReloadPolicies(ctx context.Context) error {
	return a.permMgr.Reload(ctx)
}
