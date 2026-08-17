package policies_test

import (
	"context"
	"os"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/policies"
	"github.com/asaidimu/hestia/core/system/policies/model"
)

func TestMain(m *testing.M) {
	data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	os.Exit(m.Run())
}

func openCollections(t *testing.T, p base.Persistence) (base.Collection, base.Collection) {
	t.Helper()
	ctx := context.Background()
	opColl, err := p.Collection(ctx, "_operation_policy_")
	if err != nil {
		t.Fatalf("open _operation_policy_ collection: %v", err)
	}
	ruleColl, err := p.Collection(ctx, "_iam_rule_")
	if err != nil {
		t.Fatalf("open _iam_rule_ collection: %v", err)
	}
	return opColl, ruleColl
}

func newTestModel(t *testing.T) *policies.PolicyModel {
	t.Helper()
	p := testutil.NewPersistence(t)
	opColl, ruleColl := openCollections(t, p)
	return policies.NewPolicyModel(opColl, ruleColl, nil)
}

func TestPolicyBindings(t *testing.T) {
	bindings := policies.Policies()
	if len(bindings) == 0 {
		t.Fatal("Policies() returned empty list")
	}
	for _, b := range bindings {
		if b.Name == "" {
			t.Error("Policies() contains a binding with empty Name")
		}
	}
}

func TestPolicyModelCreateAndListPolicies(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	pol := policies.Policy{
		Operation: "test:operation",
		Rule:      "administrator",
		Enabled:       true,
	}
	created, err := model.CreatePolicy(ctx, pol)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreatePolicy did not assign an ID")
	}
	if created.Operation != "test:operation" {
		t.Errorf("expected Operation %q, got %q", "test:operation", created.Operation)
	}

	policies, err := model.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}
	var found bool
	for _, pol := range policies {
		if pol.Operation == "test:operation" {
			found = true
			if pol.Rule != "administrator" {
				t.Errorf("expected Rule %q, got %q", "administrator", pol.Rule)
			}
			break
		}
	}
	if !found {
		t.Fatal("ListPolicies does not include the created policy")
	}
}

func TestPolicyModelDeletePolicyErrors(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	err := model.DeletePolicy(ctx, "test:delete-me")
	if err == nil {
		t.Fatal("DeletePolicy should return an error")
	}
}

func TestPolicyModelCreateAndGetRule(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	rule := policies.PolicyRule{
		Name:        "allow",
		RuleType:    "simple",
		Syntax:      "cel",
		Expression:  "true",
		Description: "allow rule",
	}
	created, err := model.CreateRule(ctx, rule)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateRule did not assign an ID")
	}

	got, err := model.GetRule(ctx, "allow")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if got.Name != "allow" {
		t.Errorf("expected rule name %q, got %q", "allow", got.Name)
	}
	if got.Expression != "true" {
		t.Errorf("expected expression %q, got %q", "true", got.Expression)
	}
}

func TestPolicyModelDeleteRuleBlockedByPolicy(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	created, err := model.CreateRule(ctx, policies.PolicyRule{
		Name:       "admin",
		RuleType:   "simple",
		Expression: "true",
	})
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}
	_ = created

	_, err = model.CreatePolicy(ctx, policies.Policy{
		Operation: "test:operation",
		Rule:      "admin",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	err = model.DeleteRule(ctx, "admin")
	if err == nil {
		t.Fatal("DeleteRule should fail when rule is referenced by a policy")
	}
}

func TestSetPolicyEnabledPreservesRule(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	pol := policies.Policy{
		Operation: "test:operation",
		Rule:      "administrator",
		Enabled:       true,
	}
	created, err := model.CreatePolicy(ctx, pol)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	// Disable — verify rule is preserved
	updated, err := model.SetPolicyEnabled(ctx, "test:operation", false)
	if err != nil {
		t.Fatalf("SetPolicyEnabled(false) failed: %v", err)
	}
	if updated.Enabled != false {
		t.Errorf("expected Enabled=false, got %v", updated.Enabled)
	}
	if updated.Rule != "administrator" {
		t.Errorf("expected Rule=%q after disable, got %q", "administrator", updated.Rule)
	}
	if updated.ID != created.ID {
		t.Errorf("expected same ID %q, got %q", created.ID, updated.ID)
	}

	// Re-enable — verify rule still intact
	updated, err = model.SetPolicyEnabled(ctx, "test:operation", true)
	if err != nil {
		t.Fatalf("SetPolicyEnabled(true) failed: %v", err)
	}
	if updated.Enabled != true {
		t.Errorf("expected Enabled=true, got %v", updated.Enabled)
	}
	if updated.Rule != "administrator" {
		t.Errorf("expected Rule=%q after re-enable, got %q", "administrator", updated.Rule)
	}

	// Read fresh from DB — verify persistence
	read, err := model.GetPolicyForOperation(ctx, "test:operation")
	if err != nil {
		t.Fatalf("GetPolicyForOperation failed: %v", err)
	}
	if read.Enabled != true {
		t.Errorf("expected Enabled=true from DB, got %v", read.Enabled)
	}
	if read.Rule != "administrator" {
		t.Errorf("expected Rule=%q from DB, got %q", "administrator", read.Rule)
	}
}

func TestGoDefaultRules(t *testing.T) {
	rules := policies.GoDefaultRules()
	if rules == nil {
		t.Fatal("GoDefaultRules returned nil")
	}
	if _, ok := rules["public"]; !ok {
		t.Error("GoDefaultRules missing 'public' rule")
	}
}

func TestDefaultRules(t *testing.T) {
	rules := policies.DefaultRules()
	if len(rules) == 0 {
		t.Fatal("DefaultRules returned empty list")
	}
}

func newValidateService(t *testing.T) *policies.PoliciesService {
	t.Helper()
	p := testutil.NewPersistence(t)
	rt := runtime.NewRuntime()
	if err := rt.RegisterInstance[base.Persistence](p); err != nil {
		t.Fatalf("RegisterInstance persistence: %v", err)
	}
	if err := rt.RegisterInstance[*zap.Logger](zap.NewNop()); err != nil {
		t.Fatalf("RegisterInstance logger: %v", err)
	}
	opColl, ruleColl := openCollections(t, p)
	if err := rt.RegisterInstance[*policies.PolicyModel](policies.NewPolicyModel(opColl, ruleColl, nil)); err != nil {
		t.Fatalf("RegisterInstance policy model: %v", err)
	}
	live := policies.GoDefaultRules()
	if err := rt.RegisterInstance[iam.RuleSet[iam.FunctionRule]](live); err != nil {
		t.Fatalf("RegisterInstance live rules: %v", err)
	}
	if err := rt.RegisterInstance[runtime.ReloadablePermissionManager](policies.NewLivePermissionManager(nil, nil)); err != nil {
		t.Fatalf("RegisterInstance perm manager: %v", err)
	}
	if err := policies.RegisterService(rt); err != nil {
		t.Fatalf("RegisterService: %v", err)
	}
	if err := rt.Rebuild(); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	svc, err := policies.NewPoliciesService(rt)
	if err != nil {
		t.Fatalf("NewPoliciesService: %v", err)
	}
	return svc
}

func TestValidateRule(t *testing.T) {
	ctx := context.Background()
	svc := newValidateService(t)

	t.Run("simple CEL passes", func(t *testing.T) {
		res, err := svc.ValidateRule(ctx, nil, &model.PolicyValidateInput{Payload: map[string]any{
			"rule": "'administrator' in identity.permissions",
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"administrator"}},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Valid {
			t.Errorf("expected valid=true, got %v", res.Valid)
		}
		if !res.Result {
			t.Errorf("expected result=true, got %v", res.Result)
		}
	})

	t.Run("simple CEL fails for non-admin", func(t *testing.T) {
		res, err := svc.ValidateRule(ctx, nil, &model.PolicyValidateInput{Payload: map[string]any{
			"rule": "'administrator' in identity.permissions",
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"user"}},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Valid {
			t.Errorf("expected valid=true, got %v", res.Valid)
		}
		if res.Result {
			t.Errorf("expected result=false, got %v", res.Result)
		}
	})

	t.Run("invalid CEL returns valid=false", func(t *testing.T) {
		res, err := svc.ValidateRule(ctx, nil, &model.PolicyValidateInput{Payload: map[string]any{
			"rule":    "not valid cel {{{",
			"context": map[string]any{},
		}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Valid {
			t.Errorf("expected valid=false, got %v", res.Valid)
		}
		if res.Error == "" {
			t.Error("expected error message")
		}
	})

	t.Run("composite AND rule", func(t *testing.T) {
		res, err := svc.ValidateRule(ctx, nil, &model.PolicyValidateInput{Payload: map[string]any{
			"rule": map[string]any{
				"operator": "and",
				"conditions": []any{
					map[string]any{"type": "cel", "expression": "'administrator' in identity.permissions"},
					map[string]any{"type": "cel", "expression": "identity.user_id != ''"},
				},
			},
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"administrator"}, "user_id": "abc"},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Valid {
			t.Errorf("expected valid=true, got %v", res.Valid)
		}
		if !res.Result {
			t.Errorf("expected result=true, got %v", res.Result)
		}
	})

	t.Run("composite with ref to default rule", func(t *testing.T) {
		res, err := svc.ValidateRule(ctx, nil, &model.PolicyValidateInput{Payload: map[string]any{
			"rule": map[string]any{
				"type": "ref",
				"name": "administrator",
			},
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"administrator"}},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Valid {
			t.Errorf("expected valid=true, got %v", res.Valid)
		}
		if !res.Result {
			t.Errorf("expected result=true, got %v", res.Result)
		}
	})
}