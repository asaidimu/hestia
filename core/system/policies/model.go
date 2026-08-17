package policies

import (
	"context"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/core/internal/util"
	"github.com/asaidimu/hestia/core/runtime"
)

const (
	operationCollName = "_operation_policy_"
	ruleCollName      = "_iam_rule_"
)

var (
	ErrRuleProtected       = common.NewSystemError("RULE_PROTECTED", "rule is protected and cannot be deleted")
	ErrRuleInUse           = common.NewSystemError("RULE_IN_USE", "rule is referenced by one or more policies and cannot be deleted")
	ErrPolicyAlreadyExists = common.NewSystemError("POLICY_ALREADY_EXISTS")
	ErrPolicyNotFound      = common.NewSystemError("POLICY_NOT_FOUND")
	ErrOperationNotFound   = common.NewSystemError("OPERATION_NOT_FOUND")
	ErrRuleNotFound        = common.NewSystemError("RULE_NOT_FOUND")
	ErrAccessCollection    = common.NewSystemError("ACCESS_COLLECTION")
	ErrCreateRuleDoc       = common.NewSystemError("CREATE_RULE_DOC")
	ErrMarshalRuleNode     = common.NewSystemError("MARSHAL_RULE_NODE")
)

// Binding is read-only metadata about a registered operation.
// RuleKey is the default rule name for seeding, not exposed via APIs.
type Binding struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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

// Policy binds an operation to a rule and carries optional rate-limit / throttle config.
// Persisted in _operation_policy_ collection. 1:1 with an operation.
// key is a composite of tenant_id + ":" + operation for multi-tenant lookups.
type Policy struct {
	ID        string                   `json:"id"`
	Operation string                   `json:"operation"`
	Rule      string                   `json:"rule"`
	TenantID  string                   `json:"tenantID"`
	Key       string                   `json:"key"`
	Enabled       bool                     `json:"enabled"`
	Protected     bool                     `json:"protected"`
	RateLimit     *runtime.RateLimitPolicy `json:"rateLimit,omitempty"`
	Throttle      *runtime.ThrottlePolicy  `json:"throttle,omitempty"`
}

type PolicyModel struct {
	policyColl    base.Collection
	ruleColl      base.Collection
	knownBindings []Binding
}

func NewPolicyModel(policyColl, ruleColl base.Collection, knownBindings []Binding) *PolicyModel {
	if knownBindings == nil {
		knownBindings = []Binding{}
	}
	return &PolicyModel{
		policyColl:    policyColl,
		ruleColl:      ruleColl,
		knownBindings: knownBindings,
	}
}

func (m *PolicyModel) SetKnownBindings(bindings []Binding) {
	m.knownBindings = bindings
}

func (m *PolicyModel) SetPolicyColl(c base.Collection) {
	m.policyColl = c
}

func (m *PolicyModel) SetRuleColl(c base.Collection) {
	m.ruleColl = c
}

// ── Operations (read-only, derived from knownBindings) ────────────────────

func (m *PolicyModel) ListBindings(ctx context.Context) ([]Binding, error) {
	result := make([]Binding, len(m.knownBindings))
	copy(result, m.knownBindings)
	return result, nil
}

func (m *PolicyModel) GetBinding(ctx context.Context, name string) (Binding, error) {
	for _, b := range m.knownBindings {
		if b.Name == name {
			return b, nil
		}
	}
	return Binding{}, ErrOperationNotFound.WithOperation("GetBinding").WithMessagef("binding %q not found", name)
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
	fields := util.StructToMap(rule)
	delete(fields, "id")

	if rule.Rules != nil {
		b, err := json.Marshal(rule.Rules)
		if err != nil {
			return PolicyRule{}, ErrMarshalRuleNode.WithCause(err)
		}
		fields["rules"] = string(b)
	} else {
		fields["rules"] = ""
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

	fields := util.StructToMap(updates)
	delete(fields, "id")

	if updates.Rules != nil {
		b, err := json.Marshal(updates.Rules)
		if err != nil {
			return PolicyRule{}, ErrMarshalRuleNode.WithCause(err)
		}
		fields["rules"] = string(b)
	} else {
		fields["rules"] = ""
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
		if p.Rule == name {
			return ErrRuleInUse.WithOperation("DeleteRule").WithMessagef("rule %q is referenced by policy %q", name, p.Operation)
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
	// Try composite key first (new format), fall back to operation field (old format).
	compositeKey := ":" + operationName
	q := query.NewQueryBuilder().Where("key").Eq(compositeKey).Build()
	result, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("GET_POLICY").WithCause(err)
	}
	if result.Count == 0 {
		q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
		result, err = m.policyColl.Read(ctx, &q)
		if err != nil {
			return Policy{}, common.NewSystemError("GET_POLICY").WithCause(err)
		}
	}
	if result.Count == 0 {
		return Policy{}, ErrPolicyNotFound.WithOperation("GetPolicyForOperation").WithMessagef("no policy for operation %q", operationName)
	}
	return docToPolicy(result.Data[0])
}

func (m *PolicyModel) CreatePolicy(ctx context.Context, p Policy) (Policy, error) {
	compositeKey := p.TenantID + ":" + p.Operation
	q := query.NewQueryBuilder().Where("key").Eq(compositeKey).Build()
	existing, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("CHECK_EXISTING_POLICY").WithCause(err)
	}
	if existing.Count > 0 {
		return Policy{}, ErrPolicyAlreadyExists.WithOperation("CreatePolicy").WithMessagef("policy for operation %q already exists", p.Operation)
	}

	fields := map[string]any{
		"operation": p.Operation,
		"rule":      p.Rule,
		"enabled":   p.Enabled,
		"protected": p.Protected,
		"key":       compositeKey,
	}
	if p.TenantID != "" {
		fields["tenant_id"] = p.TenantID
	}
	if p.RateLimit != nil {
		fields["rate_limit_enabled"] = p.RateLimit.Enabled
		fields["rate_identity"] = p.RateLimit.Identity
		fields["rate_capacity"] = p.RateLimit.Capacity
		fields["rate_refill"] = p.RateLimit.Refill
		fields["rate_period"] = p.RateLimit.Period
	}
	if p.Throttle != nil {
		fields["throttle_limit"] = p.Throttle.Limit
		fields["throttle_window"] = p.Throttle.Window
		if p.Throttle.Action != nil {
			fields["throttle_action_msg"] = p.Throttle.Action.Message
			fields["throttle_action_input"] = p.Throttle.Action.Input
		}
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

func policyFields(p Policy) map[string]any {
	f := map[string]any{}
	if p.Rule != "" {
		f["rule"] = p.Rule
	}
	if p.RateLimit != nil {
		f["rate_limit_enabled"] = p.RateLimit.Enabled
		f["rate_identity"] = p.RateLimit.Identity
		f["rate_capacity"] = p.RateLimit.Capacity
		f["rate_refill"] = p.RateLimit.Refill
		f["rate_period"] = p.RateLimit.Period
	}
	if p.Throttle != nil {
		f["throttle_limit"] = p.Throttle.Limit
		f["throttle_window"] = p.Throttle.Window
		if p.Throttle.Action != nil {
			f["throttle_action_msg"] = p.Throttle.Action.Message
			f["throttle_action_input"] = p.Throttle.Action.Input
		}
	}
	return f
}

func (m *PolicyModel) UpdatePolicy(ctx context.Context, operationName string, p Policy) (Policy, error) {
	q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
	existing, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("QUERY_POLICY").WithCause(err)
	}
	if existing.Count == 0 {
		return Policy{}, ErrPolicyNotFound.WithOperation("UpdatePolicy").WithMessagef("no policy for operation %q", operationName)
	}

	docID := existing.Data[0].ID()
	fields := policyFields(p)

	if len(fields) == 0 {
		return m.GetPolicyForOperation(ctx, operationName)
	}

	setDoc := data.Patch(fields).Document(ctx)
	_, err = m.policyColl.Update(ctx, &base.CollectionUpdate{
		Set:    setDoc,
		Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
	})
	if err != nil {
		return Policy{}, common.NewSystemError("UPDATE_POLICY").WithCause(err)
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
	q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
	result, err := m.policyColl.Read(ctx, &q)
	if err != nil {
		return common.NewSystemError("QUERY_POLICY").WithCause(err)
	}
	if result.Count == 0 {
		return ErrPolicyNotFound.WithOperation("DeletePolicy").WithMessagef("no policy for operation %q", operationName)
	}

	filter := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(result.Data[0].ID()).Build().Filters
	deleted, err := m.policyColl.Delete(ctx, filter, false)
	if err != nil {
		return common.NewSystemError("DELETE_POLICY").WithCause(err)
	}
	if deleted == 0 {
		return ErrPolicyNotFound.WithOperation("DeletePolicy").WithMessagef("policy for operation %q not found", operationName)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

func docToPolicy(doc data.Documenter) (Policy, error) {
	operation, err := doc.GetString("operation")
	if err != nil {
		return Policy{}, err
	}
	rule, err := doc.GetString("rule")
	if err != nil {
		return Policy{}, err
	}
	enabled, _ := doc.GetBool("enabled")
	protected, _ := doc.GetBool("protected")
	tenantID, _ := doc.GetString("tenant_id")
	key, _ := doc.GetString("key")

	p := Policy{
		ID:        doc.ID(),
		Operation: operation,
		Rule:      rule,
		TenantID:  tenantID,
		Key:       key,
		Enabled:   enabled,
		Protected: protected,
	}

	if rle, _ := doc.GetBool("rate_limit_enabled"); rle {
		p.RateLimit = &runtime.RateLimitPolicy{
			Enabled:  true,
			Identity: mustGetString(doc, "rate_identity"),
			Capacity: int64(mustGetFloat(doc, "rate_capacity")),
			Refill:   int64(mustGetFloat(doc, "rate_refill")),
			Period:   int64(mustGetFloat(doc, "rate_period")),
		}
	}

	if tl, _ := doc.GetFloat64("throttle_limit"); tl > 0 {
		p.Throttle = &runtime.ThrottlePolicy{
			Limit:  int64(tl),
			Window: int64(mustGetFloat(doc, "throttle_window")),
		}
		if msg, _ := doc.GetString("throttle_action_msg"); msg != "" {
			p.Throttle.Action = &runtime.ThrottleActionPolicy{Message: msg}
			if raw, _ := doc.Get("throttle_action_input"); raw != nil {
				if m, ok := raw.(map[string]any); ok {
					p.Throttle.Action.Input = m
				}
			}
		}
	}

	return p, nil
}

func mustGetString(doc data.Documenter, field string) string {
	s, _ := doc.GetString(field)
	return s
}

func mustGetFloat(doc data.Documenter, field string) float64 {
	f, _ := doc.GetFloat64(field)
	return f
}

func docToRule(doc data.Documenter) (PolicyRule, error) {
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
