package policies

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
)

func NewUpsertOperationHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		ruleKey, _ := body["ruleKey"].(string)
		description, _ := body["description"].(string)
		intentType, _ := body["intentType"].(string)

		op := PolicyOperation{
			Name:        name,
			RuleKey:     ruleKey,
			Description: description,
			IntentType:  intentType,
		}
		if err := policyModel.UpsertOperation(ctx, op); err != nil {
			return nil, err
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"name":        op.Name,
				"ruleKey":     op.RuleKey,
				"description": op.Description,
				"intentType":  op.IntentType,
			}, ctx),
		}, nil
	}
}

func NewDeleteOperationHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if err := policyModel.DeleteOperation(ctx, name); err != nil {
			return nil, err
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"message": "deleted", "name": name}, ctx),
		}, nil
	}
}

func NewUpsertRuleHandler(policyModel *PolicyModel) core.MessageHandler {
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
		if err := policyModel.UpsertRule(ctx, rule); err != nil {
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

func NewDeleteRuleHandler(policyModel *PolicyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		existing, err := policyModel.GetRule(ctx, name)
		if err != nil {
			return nil, err
		}
		if existing.Protected {
			return nil, fmt.Errorf("rule %q is protected and cannot be deleted", name)
		}
		if err := policyModel.DeleteRule(ctx, name); err != nil {
			return nil, err
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"message": "deleted", "name": name}, ctx),
		}, nil
	}
}

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
				"ruleKey":     op.RuleKey,
				"description": op.Description,
				"intentType":  op.IntentType,
			}, ctx),
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

type ruleCompiler func(iam.AccessController, []PolicyRule) (iam.FunctionRuleSet, error)

func NewReloadPoliciesHandler(policyModel *PolicyModel, permManager core.ReloadablePermissionManager, accessController iam.AccessController, compileRules ruleCompiler) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		if err := permManager.Reload(ctx); err != nil {
			return nil, fmt.Errorf("reload permissions: %w", err)
		}

		rules, err := policyModel.ListRules(ctx)
		if err != nil {
			return nil, fmt.Errorf("list rules: %w", err)
		}

		fnRules, err := compileRules(accessController, rules)
		if err != nil {
			return nil, fmt.Errorf("compile rules: %w", err)
		}

		if err := accessController.LoadRules(fnRules); err != nil {
			return nil, fmt.Errorf("load rules: %w", err)
		}

		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"operations": len(permManager.ListCapabilities()),
				"rules":      len(fnRules),
			}, ctx),
		}, nil
	}
}
