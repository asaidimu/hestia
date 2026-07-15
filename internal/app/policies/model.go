package policies

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const (
	operationCollName = "_policy_operation_"
	ruleCollName      = "_policy_rule_"
)

type PolicyOperation struct {
	Name        string `json:"name"`
	RuleKey     string `json:"ruleKey"`
	Description string `json:"description,omitempty"`
	IntentType  string `json:"intentType,omitempty"`
	Protected   bool   `json:"protected"`
}

// RuleNode is a node in a composite rule tree. A leaf has Type "ref" or "cel";
// a branch has Operator + Conditions (recursive). Mirrors the QDSL filter DSL.
type RuleNode struct {
	Type       string     `json:"type,omitempty"`       // "ref" | "cel" for leaves; absent means group
	Name       string     `json:"name,omitempty"`        // ref target name
	Expression string     `json:"expression,omitempty"`  // CEL expression for cel leaf
	Operator   string     `json:"operator,omitempty"`    // "AND" | "OR" | "NOT" | ... for groups
	Conditions []RuleNode `json:"conditions,omitempty"`  // sub-rules for groups
}

type PolicyRule struct {
	Name        string    `json:"name"`
	RuleType    string    `json:"ruleType"`               // "simple" | "composite"
	Syntax      string    `json:"syntax,omitempty"`       // "cel"
	Expression  string    `json:"expression,omitempty"`   // for simple rules
	Rules       *RuleNode `json:"rules,omitempty"`        // for composite rules
	Description string    `json:"description,omitempty"`
	Protected   bool      `json:"protected"`
}

type PolicyModel struct {
	persistence base.Persistence
}

func NewPolicyModel(persistence base.Persistence) *PolicyModel {
	return &PolicyModel{persistence: persistence}
}

// ── Operations ──────────────────────────────────────────────────────────────

func (m *PolicyModel) ListOperations(ctx context.Context) ([]PolicyOperation, error) {
	col, err := m.persistence.Collection(ctx, operationCollName)
	if err != nil {
		return nil, fmt.Errorf("access policy_operation collection: %w", err)
	}

	q := query.NewQueryBuilder().Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("list policy operations: %w", err)
	}

	ops := make([]PolicyOperation, 0, result.Count)
	for _, doc := range result.Data {
		op, err := docToOperation(doc)
		if err != nil {
			continue
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func (m *PolicyModel) UpsertOperation(ctx context.Context, op PolicyOperation) error {
	col, err := m.persistence.Collection(ctx, operationCollName)
	if err != nil {
		return fmt.Errorf("access policy_operation collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("name").Eq(op.Name).Build()
	existing, err := col.Read(ctx, &q)
	if err != nil {
		return fmt.Errorf("query operation: %w", err)
	}

	fields := map[string]any{
		"name":        op.Name,
		"ruleKey":     op.RuleKey,
		"description": op.Description,
		"intentType":  op.IntentType,
		"protected":   op.Protected,
	}

	if existing.Count > 0 {
		docID := existing.Data[0].ID()
		setDoc := data.Patch(fields).Document(ctx)
		_, err = col.Update(ctx, &base.CollectionUpdate{
			Set:    setDoc,
			Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
		})
		if err != nil {
			return fmt.Errorf("update operation: %w", err)
		}
		return nil
	}

	doc := data.MustNewDocument(fields)
	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create operation: %w", err)
	}
	return nil
}

func (m *PolicyModel) DeleteOperation(ctx context.Context, name string) error {
	op, err := m.GetOperation(ctx, name)
	if err != nil {
		return err
	}
	if op.Protected {
		return fmt.Errorf("cannot delete protected operation %q", name)
	}

	col, err := m.persistence.Collection(ctx, operationCollName)
	if err != nil {
		return fmt.Errorf("access policy_operation collection: %w", err)
	}

	filter := query.NewQueryBuilder().Where("name").Eq(name).Build().Filters
	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("delete operation: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("operation not found")
	}
	return nil
}

func (m *PolicyModel) GetOperation(ctx context.Context, name string) (PolicyOperation, error) {
	col, err := m.persistence.Collection(ctx, operationCollName)
	if err != nil {
		return PolicyOperation{}, fmt.Errorf("access policy_operation collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("name").Eq(name).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return PolicyOperation{}, fmt.Errorf("query operation: %w", err)
	}
	if result.Count == 0 {
		return PolicyOperation{}, fmt.Errorf("operation not found")
	}
	return docToOperation(result.Data[0])
}

func (m *PolicyModel) ForceDeleteOperation(ctx context.Context, name string) error {
	col, err := m.persistence.Collection(ctx, operationCollName)
	if err != nil {
		return fmt.Errorf("access policy_operation collection: %w", err)
	}

	filter := query.NewQueryBuilder().Where("name").Eq(name).Build().Filters
	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("delete operation: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("operation not found")
	}
	return nil
}

// ── Rules ───────────────────────────────────────────────────────────────────

func (m *PolicyModel) ListRules(ctx context.Context) ([]PolicyRule, error) {
	col, err := m.persistence.Collection(ctx, ruleCollName)
	if err != nil {
		return nil, fmt.Errorf("access policy_rule collection: %w", err)
	}

	q := query.NewQueryBuilder().Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("list policy rules: %w", err)
	}

	rules := make([]PolicyRule, 0, result.Count)
	for _, doc := range result.Data {
		r, err := docToRule(doc)
		if err != nil {
			continue
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (m *PolicyModel) GetRule(ctx context.Context, name string) (PolicyRule, error) {
	col, err := m.persistence.Collection(ctx, ruleCollName)
	if err != nil {
		return PolicyRule{}, fmt.Errorf("access policy_rule collection: %w", err)
	}
	q := query.NewQueryBuilder().Where("name").Eq(name).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return PolicyRule{}, fmt.Errorf("query rule: %w", err)
	}
	if result.Count == 0 {
		return PolicyRule{}, fmt.Errorf("rule not found")
	}
	return docToRule(result.Data[0])
}

func (m *PolicyModel) UpsertRule(ctx context.Context, rule PolicyRule) error {
	col, err := m.persistence.Collection(ctx, ruleCollName)
	if err != nil {
		return fmt.Errorf("access policy_rule collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("name").Eq(rule.Name).Build()
	existing, err := col.Read(ctx, &q)
	if err != nil {
		return fmt.Errorf("query rule: %w", err)
	}

	var rulesJSON string
	if rule.Rules != nil {
		b, err := json.Marshal(rule.Rules)
		if err != nil {
			return fmt.Errorf("marshal rules: %w", err)
		}
		rulesJSON = string(b)
	}

	fields := map[string]any{
		"name":        rule.Name,
		"ruleType":    rule.RuleType,
		"syntax":      rule.Syntax,
		"expression":  rule.Expression,
		"rules":       rulesJSON,
		"description": rule.Description,
		"protected":   rule.Protected,
	}

	if existing.Count > 0 {
		docID := existing.Data[0].ID()
		setDoc := data.Patch(fields).Document(ctx)
		_, err = col.Update(ctx, &base.CollectionUpdate{
			Set:    setDoc,
			Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
		})
		if err != nil {
			return fmt.Errorf("update rule: %w", err)
		}
		return nil
	}

	doc := data.MustNewDocument(fields)
	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create rule: %w", err)
	}
	return nil
}

func (m *PolicyModel) DeleteRule(ctx context.Context, name string) error {
	rule, err := m.GetRule(ctx, name)
	if err != nil {
		return err
	}
	if rule.Protected {
		return fmt.Errorf("cannot delete protected rule %q", name)
	}

	col, err := m.persistence.Collection(ctx, ruleCollName)
	if err != nil {
		return fmt.Errorf("access policy_rule collection: %w", err)
	}

	filter := query.NewQueryBuilder().Where("name").Eq(name).Build().Filters
	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (m *PolicyModel) ForceDeleteRule(ctx context.Context, name string) error {
	col, err := m.persistence.Collection(ctx, ruleCollName)
	if err != nil {
		return fmt.Errorf("access policy_rule collection: %w", err)
	}

	filter := query.NewQueryBuilder().Where("name").Eq(name).Build().Filters
	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func docToOperation(doc *data.Document) (PolicyOperation, error) {
	name, err := doc.GetString("name")
	if err != nil {
		return PolicyOperation{}, err
	}
	ruleKey, err := doc.GetString("ruleKey")
	if err != nil {
		return PolicyOperation{}, err
	}
	desc, _ := doc.GetString("description")
	intentType, _ := doc.GetString("intentType")
	protected, _ := doc.GetBool("protected")
	return PolicyOperation{
		Name:        name,
		RuleKey:     ruleKey,
		Description: desc,
		IntentType:  intentType,
		Protected:   protected,
	}, nil
}

func docToRule(doc *data.Document) (PolicyRule, error) {
	name, err := doc.GetString("name")
	if err != nil {
		return PolicyRule{}, err
	}
	ruleType, _ := doc.GetString("ruleType")
	syntax, _ := doc.GetString("syntax")
	expression, _ := doc.GetString("expression")
	desc, _ := doc.GetString("description")
	protected, _ := doc.GetBool("protected")

	r := PolicyRule{
		Name:        name,
		RuleType:    ruleType,
		Syntax:      syntax,
		Expression:  expression,
		Description: desc,
		Protected:   protected,
	}

	rulesStr, _ := doc.GetString("rules")
	if rulesStr != "" {
		var node RuleNode
		if err := json.Unmarshal([]byte(rulesStr), &node); err == nil {
			r.Rules = &node
		}
	}

	return r, nil
}
