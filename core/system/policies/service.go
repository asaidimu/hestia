package policies

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/policies/model"
)

// PoliciesService is the service for the IAM policy/rule/binding domain. It
// resolves the shared PolicyModel, permission manager, and live rules from the
// runtime DI container — the same instances the module builds during boot
// (initPolicyInfra), so reads/writes go through the live repositories and stay
// in sync with the access controller.
type PoliciesService struct {
	model *PolicyModel
	perm  runtime.ReloadablePermissionManager
	live  iam.RuleSet[iam.FunctionRule]
}

func NewPoliciesService(rt abstract.Container) (*PoliciesService, error) {
	m := abstract.MustResolve[*PolicyModel](rt)
	perm := abstract.MustResolve[runtime.ReloadablePermissionManager](rt)
	live := abstract.MustResolve[iam.RuleSet[iam.FunctionRule]](rt)

	return &PoliciesService{model: m, perm: perm, live: live}, nil
}

// GetBinding returns the metadata for a registered operation binding.
//
// @hestia.register(
//
//	name="system:policies:binding:get",
//	intent="read",
//	rule="administrator",
//	description="Get binding info",
//	resource_id="name",
//
// )
func (s *PoliciesService) GetBinding(ctx context.Context, msg abstract.Message, input *model.PolicyBindingGetInput) (*model.BindingView, error) {
	b, err := s.model.GetBinding(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	return document.New(&model.BindingView{Name: b.Name, Description: b.Description}), nil
}

// ListBindings lists all registered operation bindings.
//
// @hestia.register(
//
//	name="system:policies:binding:list",
//	intent="read",
//	rule="administrator",
//	description="List all bindings",
//
// )
func (s *PoliciesService) ListBindings(ctx context.Context, msg abstract.Message, input *model.PolicyBindingListInput) (*model.BindingListDocument, error) {
	bindings, err := s.model.ListBindings(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.BindingView, len(bindings))
	for i, b := range bindings {
		items[i] = model.BindingView{Name: b.Name, Description: b.Description}
	}
	return document.New(&model.BindingListDocument{Bindings: items}), nil
}

// ValidateRule validates a CEL rule expression or rule node against live rules.
//
// @hestia.register(
//
//	name="system:policies:rule:validate",
//	intent="check",
//	rule="administrator",
//	description="Validate a CEL rule expression",
//
// )
func (s *PoliciesService) ValidateRule(ctx context.Context, msg abstract.Message, input *model.PolicyValidateInput) (*model.PolicyValidateResult, error) {
	payload := input.Payload
	if payload == nil {
		return nil, common.NewSystemError("VALIDATION_ERROR", "request body is required")
	}

	contextRaw, _ := payload["context"].(map[string]any)
	identity, _ := contextRaw["identity"].(map[string]any)
	resource, _ := contextRaw["resource"].(map[string]any)
	env, _ := contextRaw["environment"].(map[string]any)

	req := iam.AccessRequest{
		Identity:    identity,
		Resource:    resource,
		Environment: env,
	}

	switch rule := payload["rule"].(type) {
	case string:
		fn, err := CompileCEL(rule)
		if err != nil {
			return document.New(&model.PolicyValidateResult{
				Valid:  false,
				Result: false,
				Error:  err.Error(),
			}), nil
		}
		return document.New(&model.PolicyValidateResult{
			Valid:  true,
			Result: fn(req),
		}), nil
	case map[string]any:
		b, _ := json.Marshal(rule)
		var node RuleNode
		if err := json.Unmarshal(b, &node); err != nil {
			return nil, common.NewSystemError("VALIDATION_ERROR", "invalid rule node: "+err.Error())
		}
		fn, err := compileValidateNode(&node, s.live)
		if err != nil {
			return document.New(&model.PolicyValidateResult{
				Valid:  false,
				Result: false,
				Error:  err.Error(),
			}), nil
		}
		if fn == nil {
			return nil, common.NewSystemError("VALIDATION_ERROR", "invalid rule node")
		}
		return document.New(&model.PolicyValidateResult{
			Valid:  true,
			Result: fn(req),
		}), nil
	default:
		return nil, common.NewSystemError("VALIDATION_ERROR", "rule must be a CEL string or a rule object")
	}
}

// compileValidateNode recursively compiles a rule tree node into a function.
func compileValidateNode(node *RuleNode, liveRules iam.RuleSet[iam.FunctionRule]) (iam.FunctionRule, error) {
	if node == nil {
		return nil, fmt.Errorf("nil rule node")
	}
	switch node.Type {
	case "ref":
		if liveRules == nil {
			return nil, fmt.Errorf("ref %q not found — no live rules available", node.Name)
		}
		fn, ok := liveRules.Get(node.Name)
		if !ok {
			return nil, fmt.Errorf("ref %q not found in live rules", node.Name)
		}
		return fn, nil
	case "cel":
		return CompileCEL(node.Expression)
	}
	if node.Operator == "" {
		return nil, fmt.Errorf("rule node must have expression, type, or operator")
	}
	fns := make([]iam.FunctionRule, len(node.Conditions))
	for i, child := range node.Conditions {
		fn, err := compileValidateNode(&child, liveRules)
		if err != nil {
			return nil, fmt.Errorf("condition %d: %w", i, err)
		}
		fns[i] = fn
	}
	return combineRules(node.Operator, fns), nil
}

// combineRules combines a set of function rules under the given operator.
func combineRules(op string, fns []iam.FunctionRule) iam.FunctionRule {
	switch op {
	case "and":
		return func(req iam.AccessRequest) bool {
			if len(fns) == 0 {
				return false
			}
			for _, fn := range fns {
				if !fn(req) {
					return false
				}
			}
			return true
		}
	case "or":
		return func(req iam.AccessRequest) bool {
			for _, fn := range fns {
				if fn(req) {
					return true
				}
			}
			return false
		}
	}
	return func(i iam.AccessRequest) bool { return false }
}

// ListRules lists all policy rules.
//
// @hestia.register(
//
//	name="system:policies:rule:list",
//	intent="read",
//	rule="administrator",
//	description="List all rules",
//
// )
func (s *PoliciesService) ListRules(ctx context.Context, msg abstract.Message, input *model.PolicyRuleListInput) (*model.RuleListDocument, error) {
	rules, err := s.model.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.PolicyRuleView, len(rules))
	for i, rule := range rules {
		items[i] = RuleToView(rule)
	}
	return document.New(&model.RuleListDocument{Rules: items}), nil
}

// GetRule returns a policy rule by name.
//
// @hestia.register(
//
//	name="system:policies:rule:get",
//	intent="read",
//	rule="administrator",
//	description="Get a policy rule",
//	resource_id="name",
//
// )
func (s *PoliciesService) GetRule(ctx context.Context, msg abstract.Message, input *model.PolicyRuleGetInput) (*model.PolicyRuleView, error) {
	rule, err := s.model.GetRule(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	v := RuleToView(rule)
	return document.New(&v), nil
}

// CreateRule creates a policy rule.
//
// @hestia.register(
//
//	name="system:policies:rule:create",
//	intent="create",
//	rule="administrator",
//	description="Create a policy rule",
//	resource_id="name",
//
// )
func (s *PoliciesService) CreateRule(ctx context.Context, msg abstract.Message, input *model.PolicyRuleCreateInput) (*model.PolicyRuleView, error) {
	name := input.Name
	body := input.Payload
	ruleType, _ := body["ruleType"].(string)
	if ruleType == "" {
		ruleType = "simple"
	}
	syntax, _ := body["syntax"].(string)
	expression, _ := body["expression"].(string)
	description, _ := body["description"].(string)

	var rules *RuleNode
	if rawRules, exists := body["rules"]; exists {
		if b, err := json.Marshal(rawRules); err == nil {
			json.Unmarshal(b, &rules)
		}
	}

	rule := PolicyRule{
		Name:        name,
		RuleType:    ruleType,
		Syntax:      syntax,
		Expression:  expression,
		Rules:       rules,
		Description: description,
	}
	created, err := s.model.CreateRule(ctx, rule)
	if err != nil {
		return nil, err
	}
	v := RuleToView(created)
	return document.New(&v), nil
}

// UpdateRule updates a policy rule.
//
// @hestia.register(
//
//	name="system:policies:rule:update",
//	intent="update",
//	rule="administrator",
//	description="Update a policy rule",
//	resource_id="name",
//
// )
func (s *PoliciesService) UpdateRule(ctx context.Context, msg abstract.Message, input *model.PolicyRuleUpdateInput) (*model.PolicyRuleView, error) {
	name := input.Name
	body := input.Payload
	ruleType, _ := body["ruleType"].(string)
	syntax, _ := body["syntax"].(string)
	expression, _ := body["expression"].(string)
	description, _ := body["description"].(string)
	protected, _ := body["protected"].(bool)

	var rules *RuleNode
	if rawRules, exists := body["rules"]; exists {
		if b, err := json.Marshal(rawRules); err == nil {
			json.Unmarshal(b, &rules)
		}
	}

	rule := PolicyRule{
		Name:        name,
		RuleType:    ruleType,
		Syntax:      syntax,
		Expression:  expression,
		Rules:       rules,
		Description: description,
		Protected:   protected,
	}
	updated, err := s.model.UpdateRule(ctx, name, rule)
	if err != nil {
		return nil, err
	}
	v := RuleToView(updated)
	return document.New(&v), nil
}

// DeleteRule deletes a policy rule.
//
// @hestia.register(
//
//	name="system:policies:rule:delete",
//	intent="delete",
//	rule="administrator",
//	description="Delete a policy rule",
//	resource_id="name",
//
// )
func (s *PoliciesService) DeleteRule(ctx context.Context, msg abstract.Message, input *model.PolicyRuleDeleteInput) (*model.RuleDeletedResult, error) {
	if err := s.model.DeleteRule(ctx, input.Name); err != nil {
		return nil, err
	}
	return document.New(&model.RuleDeletedResult{Name: input.Name}), nil
}

// Reload reloads policies from the database into the permission manager.
//
// @hestia.register(
//
//	name="system:policies:reload",
//	intent="read",
//	rule="administrator",
//	description="Reload policies from database",
//
// )
func (s *PoliciesService) Reload(ctx context.Context, msg abstract.Message, input *model.PolicyReloadInput) (*model.PolicyReloadResult, error) {
	if err := s.perm.Reload(ctx); err != nil {
		return nil, err
	}

	dbRules, err := s.model.ListRules(ctx)
	if err != nil {
		return nil, err
	}

	ruleCount := 0
	for _, r := range dbRules {
		if r.Expression == "" {
			continue
		}
		fn, err := CompileCEL(r.Expression)
		if err != nil {
			continue
		}
		if s.live != nil {
			s.live.Set(r.Name, fn)
			ruleCount++
		}
	}

	return document.New(&model.PolicyReloadResult{
		Operations: len(s.perm.ListCapabilities()),
		Rules:      ruleCount,
	}), nil
}

// CreatePolicy creates a policy binding for an operation.
//
// @hestia.register(
//
//	name="system:policies:policy:create",
//	intent="create",
//	rule="administrator",
//	description="Create a policy binding",
//	resource_id="name",
//
// )
func (s *PoliciesService) CreatePolicy(ctx context.Context, msg abstract.Message, input *model.PolicyCreateInput) (*model.PolicyView, error) {
	operationName := input.Name
	body := input.Payload
	ruleName, _ := body["rule"].(string)

	p := Policy{
		Operation: operationName,
		Rule:      ruleName,
		Enabled:   true,
		RateLimit: parseRateLimit(body),
		Throttle:  parseThrottle(body),
	}
	created, err := s.model.CreatePolicy(ctx, p)
	if err != nil {
		return nil, err
	}
	v := PolicyToView(created)
	return document.New(&v), nil
}

// UpdatePolicy updates a policy binding — rule, enabled, rateLimit, throttle.
//
// @hestia.register(
//
//	name="system:policies:policy:update",
//	intent="update",
//	rule="administrator",
//	description="Update a policy — set rule, enabled, or both",
//	resource_id="name",
//
// )
func (s *PoliciesService) UpdatePolicy(ctx context.Context, msg abstract.Message, input *model.PolicyUpdateInput) (*model.PolicyView, error) {
	operationName := input.Name
	body := input.Payload

	var updated Policy
	var err error

	if _, ok := body["rule"]; ok {
		ruleName, _ := body["rule"].(string)
		updated, err = s.model.UpdatePolicyRule(ctx, operationName, ruleName)
		if err != nil {
			return nil, err
		}
	}
	if _, ok := body["enabled"]; ok {
		enabled, _ := body["enabled"].(bool)
		updated, err = s.model.SetPolicyEnabled(ctx, operationName, enabled)
		if err != nil {
			return nil, err
		}
	}

	update := Policy{}
	if _, ok := body["rateLimit"]; ok {
		update.RateLimit = parseRateLimit(body)
	}
	if _, ok := body["throttle"]; ok {
		update.Throttle = parseThrottle(body)
	}
	if update.RateLimit != nil || update.Throttle != nil {
		updated, err = s.model.UpdatePolicy(ctx, operationName, update)
		if err != nil {
			return nil, err
		}
	}

	if updated.ID == "" {
		updated, err = s.model.GetPolicyForOperation(ctx, operationName)
		if err != nil {
			return nil, err
		}
	}

	v := PolicyToView(updated)
	return document.New(&v), nil
}

// ListPolicies lists all policy bindings.
//
// @hestia.register(
//
//	name="system:policies:policy:list",
//	intent="read",
//	rule="administrator",
//	description="List all policy bindings",
//
// )
func (s *PoliciesService) ListPolicies(ctx context.Context, msg abstract.Message, input *model.PolicyListInput) (*model.PolicyListDocument, error) {
	policies, err := s.model.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.PolicyView, len(policies))
	for i, p := range policies {
		items[i] = PolicyToView(p)
	}
	return document.New(&model.PolicyListDocument{Policies: items}), nil
}

func parseRateLimit(body map[string]any) *runtime.RateLimitPolicy {
	raw, ok := body["rateLimit"]
	if !ok {
		return nil
	}
	r, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	rl := &runtime.RateLimitPolicy{
		Enabled:  getBool(r, "enabled"),
		Identity: getString(r, "identity"),
		Capacity: int64(getFloat(r, "capacity")),
		Refill:   int64(getFloat(r, "refill")),
		Period:   int64(getFloat(r, "period")),
	}
	if !rl.Enabled {
		return nil
	}
	return rl
}

func parseThrottle(body map[string]any) *runtime.ThrottlePolicy {
	raw, ok := body["throttle"]
	if !ok {
		return nil
	}
	t, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	tp := &runtime.ThrottlePolicy{
		Limit:  int64(getFloat(t, "limit")),
		Window: int64(getFloat(t, "window")),
	}
	if actionRaw, ok := t["action"].(map[string]any); ok {
		tp.Action = &runtime.ThrottleActionPolicy{
			Message: getString(actionRaw, "message"),
			Input:   getMap(actionRaw, "input"),
		}
	}
	return tp
}

func getString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func getFloat(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	b, _ := m[key].(bool)
	return b
}

func getMap(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}
