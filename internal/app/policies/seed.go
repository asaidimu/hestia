package policies

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/internal/core"
)

func SeedPolicies(ctx context.Context, policyModel *PolicyModel, allOps []PolicyOperation) error {
	ops, err := policyModel.ListOperations(ctx)
	if err != nil {
		return fmt.Errorf("check existing operations: %w", err)
	}

	existing := make(map[string]bool, len(ops))
	for _, op := range ops {
		existing[op.Name] = true
	}
	for _, op := range allOps {
		if existing[op.Name] {
			continue
		}
		op.Protected = true
		if err := policyModel.UpsertOperation(ctx, op); err != nil {
			return fmt.Errorf("seed operation %s: %w", op.Name, err)
		}
	}

	rules, err := policyModel.ListRules(ctx)
	if err != nil {
		return fmt.Errorf("check existing rules: %w", err)
	}

	existingRules := make(map[string]bool, len(rules))
	for _, r := range rules {
		existingRules[r.Name] = true
	}
	for _, rule := range DefaultRules() {
		if existingRules[rule.Name] {
			continue
		}
		rule.Protected = true
		if err := policyModel.UpsertRule(ctx, rule); err != nil {
			return fmt.Errorf("seed rule %s: %w", rule.Name, err)
		}
	}

	return nil
}

func PopulatePermissionManager(perms core.PermissionRegistrar, allOps []PolicyOperation) {
	for _, op := range allOps {
		perms.RegisterScope(op.Name, op.RuleKey, op.Description)
	}
}
