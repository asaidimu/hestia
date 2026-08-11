package policies

import (
	"fmt"

	"github.com/asaidimu/go-iam/v2/iam"
)

// GoDefaultRules returns the built-in rules as Go functions (no CEL).
// This avoids the bug where CEL 'administrator' in identity.permissions
// incorrectly returns true for anonymous identity.
func GoDefaultRules() iam.FunctionRuleSet {
	rules := make(iam.FunctionRuleSet)

	rules["public"] = func(req iam.AccessRequest) bool {
		return true
	}

	rules["authenticated"] = func(req iam.AccessRequest) bool {
		ident, _ := req.Identity.(map[string]any)
		uid, _ := ident["user_id"].(string)
		return uid != ""
	}

	rules["password_reset"] = func(req iam.AccessRequest) bool {
		ident, _ := req.Identity.(map[string]any)
		tt, _ := ident["token_type"].(string)
		return tt == "password_reset"
	}

	rules["administrator"] = func(req iam.AccessRequest) bool {
		ident, _ := req.Identity.(map[string]any)
		switch perms := ident["permissions"].(type) {
		case []string:
			for _, p := range perms {
				if p == "administrator" {
					return true
				}
			}
		case []any:
			for _, p := range perms {
				if s, ok := p.(string); ok && s == "administrator" {
					return true
				}
			}
		}
		return false
	}

	return rules
}

func compileCompositeNode(node *RuleNode, compiled iam.FunctionRuleSet, ac iam.AccessController) (iam.FunctionRule, error) {
	if node == nil {
		return nil, fmt.Errorf("nil rule node")
	}

	switch node.Type {
	case "ref":
		fn, ok := compiled[node.Name]
		if !ok {
			return nil, fmt.Errorf("ref %q not found", node.Name)
		}
		return fn, nil
	case "cel":
		return ac.CompileCELRule(node.Expression)
	}

	if node.Operator == "" {
		return nil, fmt.Errorf("rule node must have type, expression, or operator")
	}

	fns := make([]iam.FunctionRule, len(node.Conditions))
	for i, child := range node.Conditions {
		fn, err := compileCompositeNode(&child, compiled, ac)
		if err != nil {
			return nil, fmt.Errorf("condition %d: %w", i, err)
		}
		fns[i] = fn
	}

	return combineRules(node.Operator, fns), nil
}

func combineAll(fns []iam.FunctionRule) iam.FunctionRule {
	return func(req iam.AccessRequest) bool {
		for _, fn := range fns {
			if !fn(req) {
				return false
			}
		}
		return true
	}
}

func combineAny(fns []iam.FunctionRule) iam.FunctionRule {
	return func(req iam.AccessRequest) bool {
		for _, fn := range fns {
			if fn(req) {
				return true
			}
		}
		return false
	}
}

func combineNot(fns []iam.FunctionRule) iam.FunctionRule {
	return func(req iam.AccessRequest) bool {
		if len(fns) == 0 {
			return true
		}
		return !fns[0](req)
	}
}

func combineXOR(fns []iam.FunctionRule) iam.FunctionRule {
	return func(req iam.AccessRequest) bool {
		var count int
		for _, fn := range fns {
			if fn(req) {
				count++
			}
		}
		return count == 1
	}
}

func combineRules(op string, fns []iam.FunctionRule) iam.FunctionRule {
	switch op {
	case "AND":
		return combineAll(fns)
	case "OR":
		return combineAny(fns)
	case "NOT":
		return combineNot(fns)
	case "XOR":
		return combineXOR(fns)
	default:
		return combineAll(fns)
	}
}
