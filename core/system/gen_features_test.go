package system

import (
	"testing"

	"github.com/asaidimu/hestia/core/system/policies"
)

// TestDefaultPolicyBindings_EmptyRuleKeyDefaultsToAdministrator pins the
// codegen contract: an operation binding that declares no explicit RuleKey is
// protected by the "administrator" rule. This is the secure default — nothing
// is silently public.
func TestDefaultPolicyBindings_EmptyRuleKeyDefaultsToAdministrator(t *testing.T) {
	for _, p := range allDefaultPolicyBindings {
		if p.Rule == "" {
			t.Errorf("operation %q has an empty rule after defaulting (must be 'administrator')", p.Operation)
		}
		if p.Operation == "" {
			t.Errorf("default policy binding has an empty operation name")
		}
	}
}

// TestDefaultPolicyBindings_AllEnabled pins that generated default bindings
// are enabled at seed time. A disabled-by-default policy would lock out an
// operation until an admin flips it on.
func TestDefaultPolicyBindings_AllEnabled(t *testing.T) {
	for _, p := range allDefaultPolicyBindings {
		if !p.Enabled {
			t.Errorf("default binding for %q must be enabled", p.Operation)
		}
	}
}

// TestDefaultPolicyBindings_CoversEveryOperation pins that every registered
// operation binding is represented in the default policy set. An operation
// missing from the defaults has no policy and is denied at runtime — which is
// safe — but it would also be invisible to ListCapabilities and default seed.
func TestDefaultPolicyBindings_CoversEveryOperation(t *testing.T) {
	byName := make(map[string]policies.Policy, len(allDefaultPolicyBindings))
	for _, p := range allDefaultPolicyBindings {
		byName[p.Operation] = p
	}

	for _, b := range allPolicyBindings {
		if _, ok := byName[b.Name]; !ok {
			t.Errorf("operation %q has a binding but no default policy", b.Name)
		}
	}
}
