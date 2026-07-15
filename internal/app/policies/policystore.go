package policies

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"
)

// PolicyStoreAdapter implements collections.PolicyStore using the system
// module's PolicyModel, PermissionManager, and AccessController.
type PolicyStoreAdapter struct {
	policyModel *PolicyModel
	permMgr     *DBPermissionManager
	ac          iam.AccessController
}

func NewPolicyStoreAdapter(policyModel *PolicyModel, permMgr *DBPermissionManager, ac iam.AccessController) *PolicyStoreAdapter {
	return &PolicyStoreAdapter{
		policyModel: policyModel,
		permMgr:     permMgr,
		ac:          ac,
	}
}

func (a *PolicyStoreAdapter) EnsureOperation(ctx context.Context, name, ruleKey, intentType, description string) error {
	return a.policyModel.UpsertOperation(ctx, PolicyOperation{
		Name:        name,
		RuleKey:     ruleKey,
		IntentType:  intentType,
		Description: description,
		Protected:   true,
	})
}

func (a *PolicyStoreAdapter) DeleteOperation(ctx context.Context, name string) error {
	return a.policyModel.DeleteOperation(ctx, name)
}

func (a *PolicyStoreAdapter) ForceDeleteOperation(ctx context.Context, name string) error {
	return a.policyModel.ForceDeleteOperation(ctx, name)
}

func (a *PolicyStoreAdapter) EnsureRule(ctx context.Context, name, expr, description string) error {
	return a.policyModel.UpsertRule(ctx, PolicyRule{
		Name:        name,
		RuleType:    "simple",
		Syntax:      "cel",
		Expression:  expr,
		Description: description,
	})
}

func (a *PolicyStoreAdapter) DeleteRule(ctx context.Context, name string) error {
	return a.policyModel.DeleteRule(ctx, name)
}

func (a *PolicyStoreAdapter) ForceDeleteRule(ctx context.Context, name string) error {
	return a.policyModel.ForceDeleteRule(ctx, name)
}

func (a *PolicyStoreAdapter) ReloadPolicies(ctx context.Context) error {
	if err := a.permMgr.Reload(ctx); err != nil {
		return fmt.Errorf("reload permissions: %w", err)
	}

	dbRules, err := a.policyModel.ListRules(ctx)
	if err != nil {
		return fmt.Errorf("list rules: %w", err)
	}

	// Start with Go default rules (no CEL bugs), then merge DB rules on top
	fnRules := GoDefaultRules()

	if len(dbRules) > 0 {
		tmpAC := iam.CreateAccessController(iam.AccessControllerOptions{
			CacheTTL: 5 * time.Second,
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))

		compiled, err := CompileRules(tmpAC, dbRules)
		if err != nil {
			return fmt.Errorf("compile rules: %w", err)
		}
		for name, fn := range compiled {
			fnRules[name] = fn
		}
	}

	a.ac.LoadRules(fnRules)
	return nil
}
