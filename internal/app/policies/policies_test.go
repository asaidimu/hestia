package policies_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/policies"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

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

func TestDBPermissionManagerRegisterScope(t *testing.T) {
	pm := policies.NewDBPermissionManager(nil)
	pm.RegisterScope("test:scope", "admin:*", "test scope description")

	caps := pm.ListCapabilities()
	if len(caps) == 0 {
		t.Fatal("ListCapabilities returned empty list after RegisterScope")
	}

	var found bool
	for _, c := range caps {
		if c.Name == "test:scope" {
			found = true
			if c.Scope != "admin:*" {
				t.Errorf("expected Scope %q, got %q", "admin:*", c.Scope)
			}
			if c.Description != "test scope description" {
				t.Errorf("expected Description %q, got %q", "test scope description", c.Description)
			}
			break
		}
	}
	if !found {
		t.Fatal("ListCapabilities does not include the registered scope")
	}
}

func TestDBPermissionManagerResolve(t *testing.T) {
	pm := policies.NewDBPermissionManager(nil)
	pm.RegisterScope("admin:read", "admin:*", "admin read scope")

	scope, err := pm.Resolve(testMessage{name: "admin:read", ctx: context.Background()})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if scope != "admin:*" {
		t.Errorf("expected scope %q, got %q", "admin:*", scope)
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
