package policies

import (
	"context"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const (
	operationCollName = "_operation_policy_"
	ruleCollName      = "_iam_rule_"
)

var (
	ErrPolicyDeleteForbidden   = common.NewSystemError("POLICY_DELETE_FORBIDDEN", "policies cannot be deleted; disable the policy instead")
	ErrRuleProtected           = common.NewSystemError("RULE_PROTECTED", "rule is protected and cannot be deleted")
	ErrRuleInUse               = common.NewSystemError("RULE_IN_USE", "rule is referenced by one or more policies and cannot be deleted")
	ErrPolicyAlreadyExists     = common.NewSystemError("POLICY_ALREADY_EXISTS")
	ErrPolicyNotFound          = common.NewSystemError("POLICY_NOT_FOUND")
	ErrOperationNotFound       = common.NewSystemError("OPERATION_NOT_FOUND")
	ErrRuleNotFound            = common.NewSystemError("RULE_NOT_FOUND")
	ErrAccessCollection        = common.NewSystemError("ACCESS_COLLECTION")
	ErrCreateRuleDoc           = common.NewSystemError("CREATE_RULE_DOC")
	ErrMarshalRuleNode         = common.NewSystemError("MARSHAL_RULE_NODE")
)

// Operation is read-only metadata about a registered handler.
// RuleKey is the default rule name for seeding, not exposed via APIs.
type Operation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IntentType  string `json:"intentType"`
	RuleKey     string `json:"-"`
}

// RuleNode is a node in a composite rule tree. A leaf has Type "ref" or "cel";
// a branch has Operator + Conditions (recursive). Mirrors the QDSL filter DSL.
type RuleNode struct {
	Type       string     `json:"type,omitempty"`
	Name       string     `json:"name,omitempty"`
	Expression string     `json:"expression,omitempty"`
	Operator   string     `json:"operator,omitempty"`
	Conditions []RuleNode `json:"conditions,omitempty"`
}

type PolicyRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	RuleType    string    `json:"ruleType"`
	Syntax      string    `json:"syntax,omitempty"`
	Expression  string    `json:"expression,omitempty"`
	Rules       *RuleNode `json:"rules,omitempty"`
	Description string    `json:"description,omitempty"`
	Protected   bool      `json:"protected"`
}

// Policy binds an operation to a rule.
// Persisted in _operation_policy_ collection. 1:1 with an operation.
type Policy struct {
	ID            string `json:"id"`
	OperationName string `json:"operationName"`
	RuleName      string `json:"ruleName"`
	Enabled       bool   `json:"enabled"`
	Protected     bool   `json:"protected"`
}

type PolicyModel struct {
	policyColl base.Collection
	ruleColl   base.Collection
	knownOps   []Operation
}

func NewPolicyModel(policyColl, ruleColl base.Collection, knownOps []Operation) *PolicyModel {
	if knownOps == nil {
		knownOps = []Operation{}
	}
	return &PolicyModel{
		policyColl: policyColl,
		ruleColl:   ruleColl,
		knownOps:   knownOps,
	}
}

func (m *PolicyModel) SetKnownOps(ops []Operation) {
	m.knownOps = ops
}

func (m *PolicyModel) SetPolicyColl(c base.Collection) {
	m.policyColl = c
}

func (m *PolicyModel) SetRuleColl(c base.Collection) {
	m.ruleColl = c
}

// ── Operations (read-only, derived from knownOps) ─────────────────────────

func (m *PolicyModel) ListOperations(ctx context.Context) ([]Operation, error) {
	result := make([]Operation, len(m.knownOps))
	copy(result, m.knownOps)
	return result, nil
}

func (m *PolicyModel) GetOperation(ctx context.Context, name string) (Operation, error) {
	for _, op := range m.knownOps {
		if op.Name == name {
			return op, nil
		}
	}
	return Operation{}, ErrOperationNotFound.WithOperation("GetOperation").WithMessagef("operation %q not found", name)
}

// ── Rules ─────────────────────────────────────────────────────────────────

func (m *PolicyModel) ListRules(ctx context.Context) ([]PolicyRule, error) {
	q := query.NewQueryBuilder().Build()
	result, err := m.ruleColl.Read(ctx, &q)
	if err != nil {
		return nil, common.NewSystemError("LIST_RULES").WithCause(err)
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
	q := query.NewQueryBuilder().Where("name").Eq(name).Build()
	result, err := m.ruleColl.Read(ctx, &q)
	if err != nil {
		return PolicyRule{}, common.NewSystemError("GET_RULE").WithCause(err)
	}
	if result.Count == 0 {
		return PolicyRule{}, ErrRuleNotFound.WithOperation("GetRule").WithMessagef("rule %q not found", name)
	}
	return docToRule(result.Data[0])
}

func (m *PolicyModel) CreateRule(ctx context.Context, rule PolicyRule) (PolicyRule, error) {
	var rulesJSON string
	if rule.Rules != nil {
		b, err := json.Marshal(rule.Rules)
		if err != nil {
			return PolicyRule{}, ErrMarshalRuleNode.WithCause(err)
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

	doc, err := data.NewDocument(fields, ctx)
	if err != nil {
		return PolicyRule{}, ErrCreateRuleDoc.WithCause(err)
	}

	created, err := m.ruleColl.CreateOne(ctx, doc)
	if err != nil {
		return PolicyRule{}, common.NewSystemError("CREATE_RULE").WithCause(err)
	}

	return docToRule(created.Data)
}

func (m *PolicyModel) UpdateRule(ctx context.Context, name string, updates PolicyRule) (PolicyRule, error) {
	q := query.NewQueryBuilder().Where("name").Eq(name).Build()
	existing, err := m.ruleColl.Read(ctx, &q)
	if err != nil {
		return PolicyRule{}, common.NewSystemError("QUERY_RULE").WithCause(err)
	}
	if existing.Count == 0 {
		return PolicyRule{}, ErrRuleNotFound.WithOperation("UpdateRule").WithMessagef("rule %q not found", name)
	}

	docID := existing.Data[0].ID()

	var rulesJSON string
	if updates.Rules != nil {
		b, err := json.Marshal(updates.Rules)
		if err != nil {
			return PolicyRule{}, ErrMarshalRuleNode.WithCause(err)
		}
		rulesJSON = string(b)
	}

	fields := map[string]any{
		"name":        updates.Name,
		"ruleType":    updates.RuleType,
		"syntax":      updates.Syntax,
		"expression":  updates.Expression,
		"rules":       rulesJSON,
		"description": updates.Description,
		"protected":   updates.Protected,
	}

	setDoc := data.Patch(fields).Document(ctx)
	_, err = m.ruleColl.Update(ctx, &base.CollectionUpdate{
		Set:    setDoc,
		Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
	})
	if err != nil {
		return PolicyRule{}, common.NewSystemError("UPDATE_RULE").WithCause(err)
	}

	return m.GetRule(ctx, updates.Name)
}

func (m *PolicyModel) DeleteRule(ctx context.Context, name string) error {
	rule, err := m.GetRule(ctx, name)
	if err != nil {
		return err
	}
	if rule.Protected {
		return ErrRuleProtected.WithOperation("DeleteRule").WithMessagef("rule %q is protected and cannot be deleted", name)
	}

	policies, err := m.ListPolicies(ctx)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if p.RuleName == name {
			return ErrRuleInUse.WithOperation("DeleteRule").WithMessagef("rule %q is referenced by policy %q", name, p.OperationName)
		}
	}

	filter := query.NewQueryBuilder().Where("name").Eq(name).Build().Filters
	deleted, err := m.ruleColl.Delete(ctx, filter, false)
	if err != nil {
		return common.NewSystemError("DELETE_RULE").WithCause(err)
	}
	if deleted == 0 {
		return ErrRuleNotFound.WithOperation("DeleteRule").WithMessagef("rule %q not found", name)
	}
	return nil
}

// ── Policies ──────────────────────────────────────────────────────────────

func (m *PolicyModel) ListPolicies(ctx context.Context) ([]Policy, error) {
	q := query.NewQueryBuilder().Build()
	result, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return nil, common.NewSystemError("LIST_POLICIES").WithCause(err)
	}

	policies := make([]Policy, 0, result.Count)
	for _, doc := range result.Data {
		p, err := docToPolicy(doc)
		if err != nil {
			continue
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func (m *PolicyModel) GetPolicyForOperation(ctx context.Context, operationName string) (Policy, error) {
	q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
	result, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("GET_POLICY").WithCause(err)
	}
	if result.Count == 0 {
		return Policy{}, ErrPolicyNotFound.WithOperation("GetPolicyForOperation").WithMessagef("no policy for operation %q", operationName)
	}
	return docToPolicy(result.Data[0])
}

func (m *PolicyModel) CreatePolicy(ctx context.Context, p Policy) (Policy, error) {
	q := query.NewQueryBuilder().Where("operation").Eq(p.OperationName).Build()
	existing, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("CHECK_EXISTING_POLICY").WithCause(err)
	}
	if existing.Count > 0 {
		return Policy{}, ErrPolicyAlreadyExists.WithOperation("CreatePolicy").WithMessagef("policy for operation %q already exists", p.OperationName)
	}

	fields := map[string]any{
		"operation": p.OperationName,
		"rule":      p.RuleName,
		"enabled":       p.Enabled,
		"protected":     p.Protected,
	}

	doc, err := data.NewDocument(fields, ctx)
	if err != nil {
		return Policy{}, common.NewSystemError("CREATE_POLICY_DOC").WithCause(err)
	}

	created, err := m.policyColl.CreateOne(ctx, doc)
	if err != nil {
		return Policy{}, common.NewSystemError("CREATE_POLICY").WithCause(err)
	}

	return docToPolicy(created.Data)
}

func (m *PolicyModel) UpdatePolicyRule(ctx context.Context, operationName, newRuleName string) (Policy, error) {
	q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
	existing, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("QUERY_POLICY").WithCause(err)
	}
	if existing.Count == 0 {
		return Policy{}, ErrPolicyNotFound.WithOperation("UpdatePolicyRule").WithMessagef("no policy for operation %q", operationName)
	}

	docID := existing.Data[0].ID()
	setDoc := data.Patch(map[string]any{"rule": newRuleName}).Document(ctx)
	_, err = m.policyColl.Update(ctx, &base.CollectionUpdate{
		Set:    setDoc,
		Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
	})
	if err != nil {
		return Policy{}, common.NewSystemError("UPDATE_POLICY_RULE").WithCause(err)
	}

	return m.GetPolicyForOperation(ctx, operationName)
}

func (m *PolicyModel) SetPolicyEnabled(ctx context.Context, operationName string, enabled bool) (Policy, error) {
	q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
	existing, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("QUERY_POLICY").WithCause(err)
	}
	if existing.Count == 0 {
		return Policy{}, ErrPolicyNotFound.WithOperation("SetPolicyEnabled").WithMessagef("no policy for operation %q", operationName)
	}

	docID := existing.Data[0].ID()
	setDoc := data.Patch(map[string]any{"enabled": enabled}).Document(ctx)
	_, err = m.policyColl.Update(ctx, &base.CollectionUpdate{
		Set:    setDoc,
		Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
	})
	if err != nil {
		return Policy{}, common.NewSystemError("SET_POLICY_ENABLED").WithCause(err)
	}

	return m.GetPolicyForOperation(ctx, operationName)
}

func (m *PolicyModel) DeletePolicy(ctx context.Context, operationName string) error {
	return ErrPolicyDeleteForbidden.WithOperation("DeletePolicy").WithMessagef("cannot delete policy for operation %q; disable instead", operationName)
}

// ── Helpers ───────────────────────────────────────────────────────────────

func docToPolicy(doc *data.Document) (Policy, error) {
	operationName, err := doc.GetString("operation")
	if err != nil {
		return Policy{}, err
	}
	ruleName, err := doc.GetString("rule")
	if err != nil {
		return Policy{}, err
	}
	enabled, _ := doc.GetBool("enabled")
	protected, _ := doc.GetBool("protected")

	return Policy{
		ID:            doc.ID(),
		OperationName: operationName,
		RuleName:      ruleName,
		Enabled:       enabled,
		Protected:     protected,
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
		ID:          doc.ID(),
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
