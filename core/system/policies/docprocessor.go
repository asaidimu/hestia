package policies

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"
	"cel.dev/cel-go/cel"

	"github.com/asaidimu/hestia/core/runtime"
)

var celEnv *cel.Env

func init() {
	var err error
	celEnv, err = cel.NewEnv(
		cel.Variable("identity", cel.AnyType),
		cel.Variable("resource", cel.AnyType),
		cel.Variable("environment", cel.AnyType),
	)
	if err != nil {
		panic(fmt.Sprintf("create CEL env: %v", err))
	}
}

func CompileCEL(expr string) (iam.FunctionRule, error) {
	ast, issues := celEnv.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, common.NewSystemError("CEL_COMPILE_FAILED", fmt.Sprintf("compile CEL %q: %v", expr, issues.Err()))
	}
	if ast.OutputType() != cel.BoolType {
		return nil, common.NewSystemError("CEL_COMPILE_FAILED", fmt.Sprintf("CEL %q must return bool, got %v", expr, ast.OutputType()))
	}
	prg, err := celEnv.Program(ast)
	if err != nil {
		return nil, common.SystemErrorFrom(err).WithOperation("CompileCEL").WithMessagef("program CEL %q", expr)
	}
	return func(req iam.AccessRequest) bool {
		vars := map[string]any{
			"identity":    req.Identity,
			"resource":    req.Resource,
			"environment": req.Environment,
		}
		result, _, err := prg.Eval(vars)
		if err != nil {
			return false
		}
		v, ok := result.Value().(bool)
		return ok && v
	}, nil
}

type RuleDocProcessor struct{}

func (p *RuleDocProcessor) Create(ctx context.Context, doc data.Documenter) (iam.FunctionRule, error) {
	return p.Compile(ctx, doc)
}

func (p *RuleDocProcessor) Destroy(ctx context.Context, fn iam.FunctionRule) error {
	return nil
}

func (p *RuleDocProcessor) Compile(ctx context.Context, doc data.Documenter) (iam.FunctionRule, error) {
	expr, err := doc.GetString("expression")
	if err != nil || expr == "" {
		return func(req iam.AccessRequest) bool { return false }, nil
	}
	return CompileCEL(expr)
}

func (p *RuleDocProcessor) CloneState(fn iam.FunctionRule) (iam.FunctionRule, error) {
	return fn, nil
}

// PolicyDocProcessor compiles _operation_policy_ documents into *Policy values.
type PolicyDocProcessor struct{}

func (p *PolicyDocProcessor) Create(ctx context.Context, doc data.Documenter) (*Policy, error) {
	return p.Compile(ctx, doc)
}

func (p *PolicyDocProcessor) Destroy(ctx context.Context, pol *Policy) error {
	return nil
}

func (p *PolicyDocProcessor) Compile(ctx context.Context, doc data.Documenter) (*Policy, error) {
	policy, err := docToPolicy(doc)
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (p *PolicyDocProcessor) CloneState(pol *Policy) (*Policy, error) {
	return pol, nil
}

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
