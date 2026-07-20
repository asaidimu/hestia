package policies_test

import (
	"context"
	"os"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/policies"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

func TestMain(m *testing.M) {
	data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	os.Exit(m.Run())
}

type testMessage struct {
	name string
	ctx  context.Context
}

func (m testMessage) ID() string                           { return "" }
func (m testMessage) Name() string                         { return m.name }
func (m testMessage) Context() context.Context              { return m.ctx }
func (m testMessage) Input() *data.Document                 { return data.MustNewDocument(nil, m.ctx) }
func (m testMessage) InputChannel() <-chan *data.Document   { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

var _ core.Message = testMessage{}

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
	p := persistest.NewPersistence(t)
	opColl, ruleColl := openCollections(t, p)
	return policies.NewPolicyModel(opColl, ruleColl, nil)
}

func TestDefaultOperations(t *testing.T) {
	ops := policies.DefaultOperations()
	if len(ops) == 0 {
		t.Fatal("DefaultOperations returned empty list")
	}
	for _, op := range ops {
		if op.Name == "" {
			t.Error("DefaultOperations contains an operation with empty Name")
		}
	}
}

func TestPolicyModelCreateAndListPolicies(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	pol := policies.Policy{
		OperationName: "test:operation",
		RuleName:      "administrator",
		Enabled:       true,
	}
	created, err := model.CreatePolicy(ctx, pol)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreatePolicy did not assign an ID")
	}
	if created.OperationName != "test:operation" {
		t.Errorf("expected OperationName %q, got %q", "test:operation", created.OperationName)
	}

	policies, err := model.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}
	var found bool
	for _, pol := range policies {
		if pol.OperationName == "test:operation" {
			found = true
			if pol.RuleName != "administrator" {
				t.Errorf("expected RuleName %q, got %q", "administrator", pol.RuleName)
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
		OperationName: "test:operation",
		RuleName:      "admin",
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

func TestSetPolicyEnabledPreservesRuleName(t *testing.T) {
	ctx := context.Background()
	model := newTestModel(t)

	pol := policies.Policy{
		OperationName: "test:operation",
		RuleName:      "administrator",
		Enabled:       true,
	}
	created, err := model.CreatePolicy(ctx, pol)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	// Disable — verify ruleName is preserved
	updated, err := model.SetPolicyEnabled(ctx, "test:operation", false)
	if err != nil {
		t.Fatalf("SetPolicyEnabled(false) failed: %v", err)
	}
	if updated.Enabled != false {
		t.Errorf("expected Enabled=false, got %v", updated.Enabled)
	}
	if updated.RuleName != "administrator" {
		t.Errorf("expected RuleName=%q after disable, got %q", "administrator", updated.RuleName)
	}
	if updated.ID != created.ID {
		t.Errorf("expected same ID %q, got %q", created.ID, updated.ID)
	}

	// Re-enable — verify ruleName still intact
	updated, err = model.SetPolicyEnabled(ctx, "test:operation", true)
	if err != nil {
		t.Fatalf("SetPolicyEnabled(true) failed: %v", err)
	}
	if updated.Enabled != true {
		t.Errorf("expected Enabled=true, got %v", updated.Enabled)
	}
	if updated.RuleName != "administrator" {
		t.Errorf("expected RuleName=%q after re-enable, got %q", "administrator", updated.RuleName)
	}

	// Read fresh from DB — verify persistence
	read, err := model.GetPolicyForOperation(ctx, "test:operation")
	if err != nil {
		t.Fatalf("GetPolicyForOperation failed: %v", err)
	}
	if read.Enabled != true {
		t.Errorf("expected Enabled=true from DB, got %v", read.Enabled)
	}
	if read.RuleName != "administrator" {
		t.Errorf("expected RuleName=%q from DB, got %q", "administrator", read.RuleName)
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

type testMsg struct {
	core.Message
	input *data.Document
}

func newTestMsg(ctx context.Context, payload map[string]any) testMsg {
	return testMsg{
		Message: testMessage{ctx: ctx},
		input:   data.MustNewDocument(map[string]any{"payload": payload}, ctx),
	}
}

func (m testMsg) Input() *data.Document { return m.input }

func TestValidateRuleHandler(t *testing.T) {
	ctx := context.Background()

	liveRules := policies.GoDefaultRules()

	t.Run("simple CEL passes", func(t *testing.T) {
		h := policies.NewValidateRuleHandler(liveRules)
		msg := newTestMsg(ctx, map[string]any{
			"rule": "'administrator' in identity.permissions",
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"administrator"}},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		})
		res, err := h(ctx, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		doc := res.Document.ToMap()
		if doc["valid"] != true {
			t.Errorf("expected valid=true, got %v", doc["valid"])
		}
		if doc["result"] != true {
			t.Errorf("expected result=true, got %v", doc["result"])
		}
	})

	t.Run("simple CEL fails for non-admin", func(t *testing.T) {
		h := policies.NewValidateRuleHandler(liveRules)
		msg := newTestMsg(ctx, map[string]any{
			"rule": "'administrator' in identity.permissions",
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"user"}},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		})
		res, err := h(ctx, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		doc := res.Document.ToMap()
		if doc["valid"] != true {
			t.Errorf("expected valid=true, got %v", doc["valid"])
		}
		if doc["result"] != false {
			t.Errorf("expected result=false, got %v", doc["result"])
		}
	})

	t.Run("invalid CEL returns valid=false", func(t *testing.T) {
		h := policies.NewValidateRuleHandler(liveRules)
		msg := newTestMsg(ctx, map[string]any{
			"rule":    "not valid cel {{{",
			"context": map[string]any{},
		})
		res, err := h(ctx, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		doc := res.Document.ToMap()
		if doc["valid"] != false {
			t.Errorf("expected valid=false, got %v", doc["valid"])
		}
		if doc["error"] == nil {
			t.Error("expected error message")
		}
	})

	t.Run("composite AND rule", func(t *testing.T) {
		h := policies.NewValidateRuleHandler(liveRules)
		msg := newTestMsg(ctx, map[string]any{
			"rule": map[string]any{
				"operator": "AND",
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
		})
		res, err := h(ctx, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		doc := res.Document.ToMap()
		if doc["valid"] != true {
			t.Errorf("expected valid=true, got %v", doc["valid"])
		}
		if doc["result"] != true {
			t.Errorf("expected result=true, got %v", doc["result"])
		}
	})

	t.Run("composite with ref to default rule", func(t *testing.T) {
		h := policies.NewValidateRuleHandler(liveRules)
		msg := newTestMsg(ctx, map[string]any{
			"rule": map[string]any{
				"type": "ref",
				"name": "administrator",
			},
			"context": map[string]any{
				"identity":    map[string]any{"permissions": []string{"administrator"}},
				"resource":    map[string]any{},
				"environment": map[string]any{},
			},
		})
		res, err := h(ctx, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		doc := res.Document.ToMap()
		if doc["valid"] != true {
			t.Errorf("expected valid=true, got %v", doc["valid"])
		}
		if doc["result"] != true {
			t.Errorf("expected result=true, got %v", doc["result"])
		}
	})
}
