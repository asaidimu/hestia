package policies

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/util"
	"github.com/asaidimu/hestia/core/runtime"
)

func NewGetBindingHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		b, err := policyModel.GetBinding(ctx, name)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{
				"name":        b.Name,
				"description": b.Description,
			}, ctx),
		}, nil
	}
}

func NewListBindingsHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		bindings, err := policyModel.ListBindings(ctx)
		if err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, len(bindings))
		for _, b := range bindings {
			items = append(items, map[string]any{
				"name":        b.Name,
				"description": b.Description,
			})
		}
		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{"bindings": items}, ctx),
		}, nil
	}
}

func policyToMap(p Policy) map[string]any {
	m := map[string]any{
		"id":            p.ID,
		"operationName": p.OperationName,
		"ruleName":      p.RuleName,
		"enabled":       p.Enabled,
	}
	if p.RateLimit != nil {
		m["rateLimit"] = p.RateLimit
	}
	if p.Throttle != nil {
		m["throttle"] = p.Throttle
	}
	return m
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

func NewCreatePolicyHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		operationName, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		ruleName, _ := body["ruleName"].(string)

		p := Policy{
			OperationName: operationName,
			RuleName:      ruleName,
			Enabled:       true,
			RateLimit:     parseRateLimit(body),
			Throttle:      parseThrottle(body),
		}
		created, err := policyModel.CreatePolicy(ctx, p)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{
			Document: data.MustNewDocument(policyToMap(created), ctx),
		}, nil
	}
}

func NewUpdatePolicyHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		operationName, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)

		var updated Policy
		var err error

		if _, ok := body["ruleName"]; ok {
			ruleName, _ := body["ruleName"].(string)
			updated, err = policyModel.UpdatePolicyRule(ctx, operationName, ruleName)
			if err != nil {
				return nil, err
			}
		}
		if _, ok := body["enabled"]; ok {
			enabled, _ := body["enabled"].(bool)
			updated, err = policyModel.SetPolicyEnabled(ctx, operationName, enabled)
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
			updated, err = policyModel.UpdatePolicy(ctx, operationName, update)
			if err != nil {
				return nil, err
			}
		}

		if updated.ID == "" {
			updated, err = policyModel.GetPolicyForOperation(ctx, operationName)
			if err != nil {
				return nil, err
			}
		}

		return &abstract.Result{
			Document: data.MustNewDocument(policyToMap(updated), ctx),
		}, nil
	}
}

func NewListPoliciesHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		policies, err := policyModel.ListPolicies(ctx)
		if err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, len(policies))
		for _, p := range policies {
			items = append(items, policyToMap(p))
		}
		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{"policies": items}, ctx),
		}, nil
	}
}

func NewCreateRuleHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)
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
		created, err := policyModel.CreateRule(ctx, rule)
		if err != nil {
			return nil, err
		}

		return &abstract.Result{
			Document: data.MustNewDocument(util.StructToMap(created), ctx),
		}, nil
	}
}

func NewUpdateRuleHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)
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
		updated, err := policyModel.UpdateRule(ctx, name, rule)
		if err != nil {
			return nil, err
		}

		return &abstract.Result{
			Document: data.MustNewDocument(util.StructToMap(updated), ctx),
		}, nil
	}
}

func NewGetRuleHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		rule, err := policyModel.GetRule(ctx, name)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{
			Document: data.MustNewDocument(util.StructToMap(rule), ctx),
		}, nil
	}
}

func NewListRulesHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		rules, err := policyModel.ListRules(ctx)
		if err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, len(rules))
		for _, rule := range rules {
			items = append(items, util.StructToMap(rule))
		}
		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{"rules": items}, ctx),
		}, nil
	}
}

func NewDeleteRuleHandler(policyModel *PolicyModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if err := policyModel.DeleteRule(ctx, name); err != nil {
			return nil, err
		}
		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{"message": "deleted", "name": name}, ctx),
		}, nil
	}
}

func NewValidateRuleHandler(liveRules iam.RuleSet[iam.FunctionRule]) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		payload, _ := msg.Input().GetOr("payload", nil).(map[string]any)
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

		var fn iam.FunctionRule
		switch rule := payload["rule"].(type) {
		case string:
			var err error
			fn, err = CompileCEL(rule)
			if err != nil {
				return &abstract.Result{
					Document: data.MustNewDocument(map[string]any{
						"valid":  false,
						"result": false,
						"error":  err.Error(),
					}, ctx),
				}, nil
			}
		case map[string]any:
			b, _ := json.Marshal(rule)
			var node RuleNode
			if err := json.Unmarshal(b, &node); err != nil {
				return nil, common.NewSystemError("VALIDATION_ERROR", "invalid rule node: "+err.Error())
			}
			var err error
			fn, err = compileValidateNode(&node, liveRules)
			if err != nil {
				return &abstract.Result{
					Document: data.MustNewDocument(map[string]any{
						"valid":  false,
						"result": false,
						"error":  err.Error(),
					}, ctx),
				}, nil
			}
		default:
			return nil, common.NewSystemError("VALIDATION_ERROR", "rule must be a CEL string or a rule object")
		}

		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{
				"valid":  true,
				"result": fn(req),
			}, ctx),
		}, nil
	}
}

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

func NewReloadPoliciesHandler(policyModel *PolicyModel, permManager runtime.ReloadablePermissionManager, liveRules iam.RuleSet[iam.FunctionRule]) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		if err := permManager.Reload(ctx); err != nil {
			return nil, err
		}

		dbRules, err := policyModel.ListRules(ctx)
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
			if liveRules != nil {
				liveRules.Set(r.Name, fn)
				ruleCount++
			}
		}

		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{
				"operations": len(permManager.ListCapabilities()),
				"rules":      ruleCount,
			}, ctx),
		}, nil
	}
}
