package policies

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"
	"github.com/google/cel-go/cel"
)

// celEnv is the shared CEL environment used to compile policy rule expressions.
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

// CompileCEL compiles a CEL expression into an iam.FunctionRule.
func CompileCEL(expr string) (iam.FunctionRule, error) {
	ast, issues := celEnv.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile CEL %q: %v", expr, issues.Err())
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("CEL %q must return bool, got %v", expr, ast.OutputType())
	}
	prg, err := celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program CEL %q: %v", expr, err)
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

// RuleDocProcessor compiles _iam_rule_ documents into iam.FunctionRule
// values, suitable for use as the DocumentProcessor for a LiveCollection.
type RuleDocProcessor struct{}

func (p *RuleDocProcessor) Compile(ctx context.Context, doc *data.Document) (iam.FunctionRule, error) {
	expr, err := doc.GetString("expression")
	if err != nil || expr == "" {
		return func(req iam.AccessRequest) bool { return false }, nil
	}
	return CompileCEL(expr)
}

func (p *RuleDocProcessor) CloneState(fn iam.FunctionRule) (iam.FunctionRule, error) {
	return fn, nil
}

// OperationDocProcessor compiles _operation_policy_ documents into
// *OperationPolicy values, suitable for use as the DocumentProcessor
// for a LiveCollection.
type OperationDocProcessor struct{}

func (p *OperationDocProcessor) Compile(ctx context.Context, doc *data.Document) (*OperationPolicy, error) {
	op, err := docToOperation(doc)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func (p *OperationDocProcessor) CloneState(op *OperationPolicy) (*OperationPolicy, error) {
	return op, nil
}
