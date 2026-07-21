package policies

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
)

func NewGetOperationHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		op, err := policyModel.GetOperation(ctx, name)
		if err != nil {
			return nil, err
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"name":        op.Name,
				"description": op.Description,
				"intentType":  op.IntentType,
			}, ctx),
		}, nil
	}
}

func NewListOperationsHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		ops, err := policyModel.ListOperations(ctx)
		if err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, len(ops))
		for _, op := range ops {
			items = append(items, map[string]any{
				"name":        op.Name,
				"description": op.Description,
				"intentType":  op.IntentType,
			})
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"operations": items}, ctx),
		}, nil
	}
}

func NewCreatePolicyHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		operationName, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		ruleName, _ := body["ruleName"].(string)

		p := Policy{
			OperationName: operationName,
			RuleName:      ruleName,
			Enabled:       true,
		}
		created, err := policyModel.CreatePolicy(ctx, p)
		if err != nil {
			return nil, err
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"id":            created.ID,
				"operationName": created.OperationName,
				"ruleName":      created.RuleName,
				"enabled":       created.Enabled,
			}, ctx),
		}, nil
	}
}

func NewUpdatePolicyHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		operationName, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)

		var updated Policy
		hasUpdate := false

		if _, ok := body["ruleName"]; ok {
			ruleName, _ := body["ruleName"].(string)
			var err error
			updated, err = policyModel.UpdatePolicyRule(ctx, operationName, ruleName)
			if err != nil {
				return nil, err
			}
			hasUpdate = true
		}
		if _, ok := body["enabled"]; ok {
			enabled, _ := body["enabled"].(bool)
			var err error
			updated, err = policyModel.SetPolicyEnabled(ctx, operationName, enabled)
			if err != nil {
				return nil, err
			}
			hasUpdate = true
		}
		if !hasUpdate {
			var err error
			updated, err = policyModel.GetPolicyForOperation(ctx, operationName)
			if err != nil {
				return nil, err
			}
		}

		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"id":            updated.ID,
				"operationName": updated.OperationName,
				"ruleName":      updated.RuleName,
				"enabled":       updated.Enabled,
			}, ctx),
		}, nil
	}
}

func NewDeletePolicyHandler() runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		return nil, ErrPolicyDeleteForbidden.WithOperation("DeletePolicy")
	}
}

func NewListPoliciesHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		policies, err := policyModel.ListPolicies(ctx)
		if err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, len(policies))
		for _, p := range policies {
			items = append(items, map[string]any{
				"id":            p.ID,
				"operationName": p.OperationName,
				"ruleName":      p.RuleName,
				"enabled":       p.Enabled,
			})
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"policies": items}, ctx),
		}, nil
	}
}

func NewCreateRuleHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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

		b, _ := json.Marshal(created)
		var m map[string]any
		json.Unmarshal(b, &m)
		return &registration.Result{
			Document: data.MustNewDocument(m, ctx),
		}, nil
	}
}

func NewUpdateRuleHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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

		b, _ := json.Marshal(updated)
		var m map[string]any
		json.Unmarshal(b, &m)
		return &registration.Result{
			Document: data.MustNewDocument(m, ctx),
		}, nil
	}
}

func NewGetRuleHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		rule, err := policyModel.GetRule(ctx, name)
		if err != nil {
			return nil, err
		}
		b, _ := json.Marshal(rule)
		var m map[string]any
		json.Unmarshal(b, &m)
		return &registration.Result{
			Document: data.MustNewDocument(m, ctx),
		}, nil
	}
}

func NewListRulesHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		rules, err := policyModel.ListRules(ctx)
		if err != nil {
			return nil, err
		}

		items := make([]map[string]any, 0, len(rules))
		for _, rule := range rules {
			b, _ := json.Marshal(rule)
			var m map[string]any
			json.Unmarshal(b, &m)
			items = append(items, m)
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"rules": items}, ctx),
		}, nil
	}
}

func NewDeleteRuleHandler(policyModel *PolicyModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if err := policyModel.DeleteRule(ctx, name); err != nil {
			return nil, err
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"message": "deleted", "name": name}, ctx),
		}, nil
	}
}

func NewValidateRuleHandler(liveRules iam.RuleSet[iam.FunctionRule]) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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
				return &registration.Result{
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
				return &registration.Result{
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

		return &registration.Result{
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

func NewReloadPoliciesHandler(policyModel *PolicyModel, permManager runtime.ReloadablePermissionManager, liveRules iam.RuleSet[iam.FunctionRule]) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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

		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"operations": len(permManager.ListCapabilities()),
				"rules":      ruleCount,
			}, ctx),
		}, nil
	}
}
