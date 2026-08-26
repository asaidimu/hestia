package policies

import (
	"context"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/policies/model"
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
	Enabled   bool                     `json:"enabled"`
	Protected bool                     `json:"protected"`
	RateLimit *runtime.RateLimitPolicy `json:"rateLimit,omitempty"`
	Throttle  *runtime.ThrottlePolicy  `json:"throttle,omitempty"`
}

var (
	ErrRuleProtected       = common.NewSystemError("RULE_PROTECTED", "rule is protected and cannot be deleted")
	ErrRuleInUse           = common.NewSystemError("RULE_IN_USE", "rule is referenced by one or more policies and cannot be deleted")
	ErrPolicyAlreadyExists = common.NewSystemError("POLICY_ALREADY_EXISTS")
	ErrPolicyNotFound      = common.NewSystemError("POLICY_NOT_FOUND")
	ErrOperationNotFound   = common.NewSystemError("OPERATION_NOT_FOUND")
	ErrRuleNotFound        = common.NewSystemError("RULE_NOT_FOUND")
	ErrMarshalRuleNode     = common.NewSystemError("MARSHAL_RULE_NODE")
)

// PolicyModel orchestrates the _operation_policy_ and _iam_rule_ collections
// through their generated ModelCollection wrappers. At boot those wrappers are
// constructed OVER the LiveRepositories backing the permission manager and
// access controller, so every write here lands in the DB and refreshes the
// shared live caches atomically — no manual invalidation needed.
type PolicyModel struct {
	opModel       *model.SystemOperationPolicys
	ruleModel     *model.SystemIamRules
	knownBindings []Binding
}

func NewPolicyModel(opModel *model.SystemOperationPolicys, ruleModel *model.SystemIamRules, knownBindings []Binding) *PolicyModel {
	if knownBindings == nil {
		knownBindings = []Binding{}
	}
	return &PolicyModel{
		opModel:       opModel,
		ruleModel:     ruleModel,
		knownBindings: knownBindings,
	}
}

func (m *PolicyModel) SetKnownBindings(bindings []Binding) {
	m.knownBindings = bindings
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
	rules, err := m.ruleModel.Read(ctx, &q)
	if err != nil {
		return nil, common.NewSystemError("LIST_RULES").WithCause(err)
	}

	result := make([]PolicyRule, 0, len(rules))
	for _, r := range rules {
		result = append(result, ruleFromGenerated(r))
	}
	return result, nil
}

func (m *PolicyModel) GetRule(ctx context.Context, name string) (PolicyRule, error) {
	q := query.NewQueryBuilder().Where("name").Eq(name).Build()
	rules, err := m.ruleModel.Read(ctx, &q)
	if err != nil {
		return PolicyRule{}, common.NewSystemError("GET_RULE").WithCause(err)
	}
	if len(rules) == 0 {
		return PolicyRule{}, ErrRuleNotFound.WithOperation("GetRule").WithMessagef("rule %q not found", name)
	}
	return ruleFromGenerated(rules[0]), nil
}

func (m *PolicyModel) CreateRule(ctx context.Context, rule PolicyRule) (PolicyRule, error) {
	doc := &model.SystemIamRule{
		Name:        rule.Name,
		RuleType:    strPtr(rule.RuleType),
		Syntax:      strPtr(rule.Syntax),
		Expression:  strPtr(rule.Expression),
		Description: strPtr(rule.Description),
		Protected:   boolPtr(rule.Protected),
	}

	rulesStr := ""
	if rule.Rules != nil {
		b, err := json.Marshal(rule.Rules)
		if err != nil {
			return PolicyRule{}, ErrMarshalRuleNode.WithCause(err)
		}
		rulesStr = string(b)
	}
	doc.Rules = &rulesStr

	created, err := m.ruleModel.Create(ctx, doc)
	if err != nil {
		return PolicyRule{}, common.NewSystemError("CREATE_RULE").WithCause(err)
	}

	return ruleFromGenerated(created), nil
}

func (m *PolicyModel) UpdateRule(ctx context.Context, name string, updates PolicyRule) (PolicyRule, error) {
	q := query.NewQueryBuilder().Where("name").Eq(name).Build()
	existing, err := m.ruleModel.Read(ctx, &q)
	if err != nil {
		return PolicyRule{}, common.NewSystemError("QUERY_RULE").WithCause(err)
	}
	if len(existing) == 0 {
		return PolicyRule{}, ErrRuleNotFound.WithOperation("UpdateRule").WithMessagef("rule %q not found", name)
	}
	doc := existing[0]

	rulesStr := ""
	if updates.Rules != nil {
		b, err := json.Marshal(updates.Rules)
		if err != nil {
			return PolicyRule{}, ErrMarshalRuleNode.WithCause(err)
		}
		rulesStr = string(b)
	}

	doc.RuleType = strPtr(updates.RuleType)
	doc.Syntax = strPtr(updates.Syntax)
	doc.Expression = strPtr(updates.Expression)
	doc.Description = strPtr(updates.Description)
	doc.Protected = boolPtr(updates.Protected)
	doc.Rules = &rulesStr

	if _, err := m.ruleModel.Update(ctx, doc.ID, doc); err != nil {
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

	if err := m.ruleModel.DeleteByID(ctx, rule.ID); err != nil {
		return common.NewSystemError("DELETE_RULE").WithCause(err)
	}
	return nil
}

// ── Policies ──────────────────────────────────────────────────────────────

func (m *PolicyModel) ListPolicies(ctx context.Context) ([]Policy, error) {
	q := query.NewQueryBuilder().Build()
	docs, err := m.opModel.Read(ctx, &q)
	if err != nil {
		return nil, common.NewSystemError("LIST_POLICIES").WithCause(err)
	}

	result := make([]Policy, 0, len(docs))
	for _, d := range docs {
		result = append(result, policyFromGenerated(d))
	}
	return result, nil
}

func (m *PolicyModel) GetPolicyForOperation(ctx context.Context, operationName string) (Policy, error) {
	// Try composite key first (new format), fall back to operation field (old format).
	compositeKey := ":" + operationName
	q := query.NewQueryBuilder().Where("key").Eq(compositeKey).Build()
	docs, err := m.opModel.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("GET_POLICY").WithCause(err)
	}
	if len(docs) == 0 {
		q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
		docs, err = m.opModel.Read(ctx, &q)
		if err != nil {
			return Policy{}, common.NewSystemError("GET_POLICY").WithCause(err)
		}
	}
	if len(docs) == 0 {
		return Policy{}, ErrPolicyNotFound.WithOperation("GetPolicyForOperation").WithMessagef("no policy for operation %q", operationName)
	}
	return policyFromGenerated(docs[0]), nil
}

func (m *PolicyModel) CreatePolicy(ctx context.Context, p Policy) (Policy, error) {
	compositeKey := p.TenantID + ":" + p.Operation
	q := query.NewQueryBuilder().Where("key").Eq(compositeKey).Build()
	existing, err := m.opModel.Read(ctx, &q)
	if err != nil {
		return Policy{}, common.NewSystemError("CHECK_EXISTING_POLICY").WithCause(err)
	}
	if len(existing) > 0 {
		return Policy{}, ErrPolicyAlreadyExists.WithOperation("CreatePolicy").WithMessagef("policy for operation %q already exists", p.Operation)
	}

	doc := &model.SystemOperationPolicy{
		Operation: p.Operation,
		Rule:      p.Rule,
		Enabled:   boolPtr(p.Enabled),
		Protected: boolPtr(p.Protected),
		Key:       &compositeKey,
	}
	if p.TenantID != "" {
		doc.TenantID = &p.TenantID
	}
	if p.RateLimit != nil {
		doc.RateLimitEnabled = boolPtr(p.RateLimit.Enabled)
		doc.RateIdentity = strPtr(p.RateLimit.Identity)
		doc.RateCapacity = f64Ptr(float64(p.RateLimit.Capacity))
		doc.RateRefill = f64Ptr(float64(p.RateLimit.Refill))
		doc.RatePeriod = f64Ptr(float64(p.RateLimit.Period))
	}
	if p.Throttle != nil {
		doc.ThrottleLimit = f64Ptr(float64(p.Throttle.Limit))
		doc.ThrottleWindow = f64Ptr(float64(p.Throttle.Window))
		if p.Throttle.Action != nil {
			doc.ThrottleActionMsg = strPtr(p.Throttle.Action.Message)
			doc.ThrottleActionInput = p.Throttle.Action.Input
		}
	}

	created, err := m.opModel.Create(ctx, doc)
	if err != nil {
		return Policy{}, common.NewSystemError("CREATE_POLICY").WithCause(err)
	}

	return policyFromGenerated(created), nil
}

func (m *PolicyModel) UpdatePolicyRule(ctx context.Context, operationName, newRuleName string) (Policy, error) {
	existing, err := m.findPolicyByOperation(ctx, operationName)
	if err != nil {
		return Policy{}, err
	}

	if _, err := m.opModel.Update(ctx, existing.ID, &model.SystemOperationPolicy{Rule: newRuleName}); err != nil {
		return Policy{}, common.NewSystemError("UPDATE_POLICY_RULE").WithCause(err)
	}

	return m.GetPolicyForOperation(ctx, operationName)
}

func policyFields(p Policy) *model.SystemOperationPolicy {
	f := &model.SystemOperationPolicy{}
	if p.Rule != "" {
		f.Rule = p.Rule
	}
	if p.RateLimit != nil {
		f.RateLimitEnabled = boolPtr(p.RateLimit.Enabled)
		f.RateIdentity = strPtr(p.RateLimit.Identity)
		f.RateCapacity = f64Ptr(float64(p.RateLimit.Capacity))
		f.RateRefill = f64Ptr(float64(p.RateLimit.Refill))
		f.RatePeriod = f64Ptr(float64(p.RateLimit.Period))
	}
	if p.Throttle != nil {
		f.ThrottleLimit = f64Ptr(float64(p.Throttle.Limit))
		f.ThrottleWindow = f64Ptr(float64(p.Throttle.Window))
		if p.Throttle.Action != nil {
			f.ThrottleActionMsg = strPtr(p.Throttle.Action.Message)
			f.ThrottleActionInput = p.Throttle.Action.Input
		}
	}
	return f
}

func (m *PolicyModel) UpdatePolicy(ctx context.Context, operationName string, p Policy) (Policy, error) {
	existing, err := m.findPolicyByOperation(ctx, operationName)
	if err != nil {
		return Policy{}, err
	}

	fields := policyFields(p)
	if fields.Rule == "" && fields.RateLimitEnabled == nil && fields.ThrottleLimit == nil {
		return m.GetPolicyForOperation(ctx, operationName)
	}

	if _, err := m.opModel.Update(ctx, existing.ID, fields); err != nil {
		return Policy{}, common.NewSystemError("UPDATE_POLICY").WithCause(err)
	}

	return m.GetPolicyForOperation(ctx, operationName)
}

func (m *PolicyModel) SetPolicyEnabled(ctx context.Context, operationName string, enabled bool) (Policy, error) {
	existing, err := m.findPolicyByOperation(ctx, operationName)
	if err != nil {
		return Policy{}, err
	}

	if _, err := m.opModel.Update(ctx, existing.ID, &model.SystemOperationPolicy{Enabled: boolPtr(enabled)}); err != nil {
		return Policy{}, common.NewSystemError("SET_POLICY_ENABLED").WithCause(err)
	}

	return m.GetPolicyForOperation(ctx, operationName)
}

func (m *PolicyModel) DeletePolicy(ctx context.Context, operationName string) error {
	existing, err := m.findPolicyByOperation(ctx, operationName)
	if err != nil {
		return err
	}

	if err := m.opModel.DeleteByID(ctx, existing.ID); err != nil {
		return common.NewSystemError("DELETE_POLICY").WithCause(err)
	}
	return nil
}

// findPolicyByOperation resolves a policy document by its operation field.
func (m *PolicyModel) findPolicyByOperation(ctx context.Context, operationName string) (*model.SystemOperationPolicy, error) {
	q := query.NewQueryBuilder().Where("operation").Eq(operationName).Build()
	docs, err := m.opModel.Read(ctx, &q)
	if err != nil {
		return nil, common.NewSystemError("QUERY_POLICY").WithCause(err)
	}
	if len(docs) == 0 {
		return nil, ErrPolicyNotFound.WithOperation("findPolicyByOperation").WithMessagef("no policy for operation %q", operationName)
	}
	return docs[0], nil
}

// ── Mapping helpers (generated structs ⇄ domain types) ───────────────────

func ruleFromGenerated(r *model.SystemIamRule) PolicyRule {
	result := PolicyRule{
		ID:          r.ID,
		Name:        r.Name,
		RuleType:    derefStr(r.RuleType),
		Syntax:      derefStr(r.Syntax),
		Expression:  derefStr(r.Expression),
		Description: derefStr(r.Description),
		Protected:   derefBool(r.Protected),
	}

	if r.Rules != nil && *r.Rules != "" {
		var node RuleNode
		if err := json.Unmarshal([]byte(*r.Rules), &node); err == nil {
			result.Rules = &node
		}
	}
	return result
}

func policyFromGenerated(p *model.SystemOperationPolicy) Policy {
	result := Policy{
		ID:        p.ID,
		Operation: p.Operation,
		Rule:      p.Rule,
		TenantID:  derefStr(p.TenantID),
		Key:       derefStr(p.Key),
		Enabled:   derefBool(p.Enabled),
		Protected: derefBool(p.Protected),
	}

	if p.RateLimitEnabled != nil && *p.RateLimitEnabled {
		result.RateLimit = &runtime.RateLimitPolicy{
			Enabled:  true,
			Identity: derefStr(p.RateIdentity),
			Capacity: int64(derefF64(p.RateCapacity)),
			Refill:   int64(derefF64(p.RateRefill)),
			Period:   int64(derefF64(p.RatePeriod)),
		}
	}

	if tl := derefF64(p.ThrottleLimit); tl > 0 {
		throttle := &runtime.ThrottlePolicy{
			Limit:  int64(tl),
			Window: int64(derefF64(p.ThrottleWindow)),
		}
		if msg := derefStr(p.ThrottleActionMsg); msg != "" {
			throttle.Action = &runtime.ThrottleActionPolicy{
				Message: msg,
				Input:   p.ThrottleActionInput,
			}
		}
		result.Throttle = throttle
	}

	return result
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func f64Ptr(f float64) *float64 {
	return &f
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func derefF64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
