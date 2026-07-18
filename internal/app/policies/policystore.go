package policies

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
)

// PolicyStoreAdapter implements collections.PolicyStore using the system
// module's PolicyModel, LiveCollection-backed caches for both rules and
// operations.  Every write through this adapter updates the in-memory
// cache immediately — no manual reload needed.
type PolicyStoreAdapter struct {
	policyModel *PolicyModel
	permMgr     core.ReloadablePermissionManager
	liveOps     collection.LiveCollection[*OperationPolicy]
	liveRules   iam.RuleSet[iam.FunctionRule]
}

func NewPolicyStoreAdapter(policyModel *PolicyModel, permMgr core.ReloadablePermissionManager, liveOps collection.LiveCollection[*OperationPolicy], liveRules iam.RuleSet[iam.FunctionRule]) *PolicyStoreAdapter {
	return &PolicyStoreAdapter{
		policyModel: policyModel,
		permMgr:     permMgr,
		liveOps:     liveOps,
		liveRules:   liveRules,
	}
}

func (a *PolicyStoreAdapter) EnsureOperation(ctx context.Context, name, ruleKey, intentType, description string) error {
	op := OperationPolicy{
		Name:        name,
		RuleKey:     ruleKey,
		IntentType:  intentType,
		Description: description,
		Protected:   true,
	}
	if err := a.policyModel.UpsertOperation(ctx, op); err != nil {
		return err
	}
	if a.liveOps != nil {
		a.liveOps.Set(name, &op)
	}
	return nil
}

func (a *PolicyStoreAdapter) DeleteOperation(ctx context.Context, name string) error {
	if err := a.policyModel.DeleteOperation(ctx, name); err != nil {
		return err
	}
	if a.liveOps != nil {
		a.liveOps.Unset(name)
	}
	return nil
}

func (a *PolicyStoreAdapter) ForceDeleteOperation(ctx context.Context, name string) error {
	if err := a.policyModel.ForceDeleteOperation(ctx, name); err != nil {
		return err
	}
	if a.liveOps != nil {
		a.liveOps.Unset(name)
	}
	return nil
}

func (a *PolicyStoreAdapter) EnsureRule(ctx context.Context, name, expr, description string) error {
	if err := a.policyModel.UpsertRule(ctx, PolicyRule{
		Name:        name,
		RuleType:    "simple",
		Syntax:      "cel",
		Expression:  expr,
		Description: description,
	}); err != nil {
		return err
	}
	fn, err := CompileCEL(expr)
	if err != nil {
		return err
	}
	if a.liveRules != nil {
		a.liveRules.Set(name, fn)
	}
	return nil
}

func (a *PolicyStoreAdapter) DeleteRule(ctx context.Context, name string) error {
	if err := a.policyModel.DeleteRule(ctx, name); err != nil {
		return err
	}
	if a.liveRules != nil {
		a.liveRules.Unset(name)
	}
	return nil
}

func (a *PolicyStoreAdapter) ForceDeleteRule(ctx context.Context, name string) error {
	if err := a.policyModel.ForceDeleteRule(ctx, name); err != nil {
		return err
	}
	if a.liveRules != nil {
		a.liveRules.Unset(name)
	}
	return nil
}

func (a *PolicyStoreAdapter) ReloadPolicies(ctx context.Context) error {
	return a.permMgr.Reload(ctx)
}
