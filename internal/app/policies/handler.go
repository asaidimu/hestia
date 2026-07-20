package policies

import (
	"context"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
)

func NewGetOperationHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewListOperationsHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewCreatePolicyHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewUpdatePolicyHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewDeletePolicyHandler() core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		return nil, ErrPolicyDeleteForbidden.WithOperation("DeletePolicy")
	}
}

func NewListPoliciesHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewCreateRuleHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewUpdateRuleHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewGetRuleHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewListRulesHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewDeleteRuleHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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

func NewValidateRuleHandler() core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"valid":  true,
				"result": true,
			}, ctx),
		}, nil
	}
}

func NewReloadPoliciesHandler(policyModel *PolicyModel, permManager core.ReloadablePermissionManager, liveRules iam.RuleSet[iam.FunctionRule]) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
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
