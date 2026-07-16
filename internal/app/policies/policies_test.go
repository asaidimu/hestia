package policies_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

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

func TestPolicyModelUpsertAndListOperations(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := policies.NewPolicyModel(p)

	op := policies.PolicyOperation{
		Name:        "test:operation",
		RuleKey:     "administrator",
		Description: "test operation",
	}
	if err := model.UpsertOperation(ctx, op); err != nil {
		t.Fatalf("UpsertOperation failed: %v", err)
	}

	ops, err := model.ListOperations(ctx)
	if err != nil {
		t.Fatalf("ListOperations failed: %v", err)
	}

	var found bool
	for _, o := range ops {
		if o.Name == "test:operation" {
			found = true
			if o.RuleKey != "administrator" {
				t.Errorf("expected RuleKey %q, got %q", "administrator", o.RuleKey)
			}
			break
		}
	}
	if !found {
		t.Fatal("ListOperations does not include the upserted operation")
	}
}

func TestPolicyModelDeleteOperation(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := policies.NewPolicyModel(p)

	op := policies.PolicyOperation{
		Name:    "test:delete-me",
		RuleKey: "public",
	}
	if err := model.UpsertOperation(ctx, op); err != nil {
		t.Fatalf("UpsertOperation failed: %v", err)
	}
	if err := model.DeleteOperation(ctx, "test:delete-me"); err != nil {
		t.Fatalf("DeleteOperation failed: %v", err)
	}

	ops, err := model.ListOperations(ctx)
	if err != nil {
		t.Fatalf("ListOperations failed: %v", err)
	}
	for _, o := range ops {
		if o.Name == "test:delete-me" {
			t.Fatal("DeleteOperation did not remove the operation")
		}
	}
}

func TestPolicyModelUpsertAndGetRule(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := policies.NewPolicyModel(p)

	rule := policies.PolicyRule{
		Name:        "allow",
		RuleType:    "simple",
		Syntax:      "cel",
		Expression:  "true",
		Description: "allow rule",
	}
	if err := model.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("UpsertRule failed: %v", err)
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
