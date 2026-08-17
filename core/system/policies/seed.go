package policies

import (
	"context"
	"fmt"
)

// SeedPolicies seeds the initial set of rules and policies.
// Idempotent — existing records are left unchanged.
func SeedPolicies(ctx context.Context, policyModel *PolicyModel, initialPolicies []Policy) error {
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
		if _, err := policyModel.CreateRule(ctx, rule); err != nil {
			return fmt.Errorf("seed rule %s: %w", rule.Name, err)
		}
	}

	for _, policy := range initialPolicies {
		existing, err := policyModel.GetPolicyForOperation(ctx, policy.Operation)
		if err == nil && existing.Operation != "" {
			continue
		}
		policy.Protected = true
		if _, err := policyModel.CreatePolicy(ctx, policy); err != nil {
			return fmt.Errorf("seed policy %s: %w", policy.Operation, err)
		}
	}

	return nil
}
