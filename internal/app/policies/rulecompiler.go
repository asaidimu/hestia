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
		perms, _ := ident["permissions"].([]string)
		for _, p := range perms {
			if p == "administrator" {
				return true
			}
		}
		return false
	}

	rules["blob"] = func(req iam.AccessRequest) bool {
		if res, ok := req.Resource.(map[string]any); ok {
			if pub, ok := res["public"].(bool); ok && pub {
				return true
			}
		}
		ident, _ := req.Identity.(map[string]any)
		perms, _ := ident["permissions"].([]string)
		for _, p := range perms {
			if p == "administrator" {
				return true
			}
		}
		return false
	}

	return rules
}

// CompileRules compiles a list of PolicyRule records into a FunctionRuleSet.
// Simple rules are compiled via ac.CompileCELRule.
// Composite rules are compiled by recursively resolving refs and compiling CEL leaves,
// then composing them into a single FunctionRule closure.
// Simples are compiled first so composites can reference them.
func CompileRules(ac iam.AccessController, dbRules []PolicyRule) (iam.FunctionRuleSet, error) {
	fnRules := make(iam.FunctionRuleSet)

	for _, r := range dbRules {
		if r.RuleType == "composite" {
			continue
		}
		fn, err := ac.CompileCELRule(r.Expression)
		if err != nil {
			return nil, fmt.Errorf("compile rule %q: %w", r.Name, err)
		}
		fnRules[r.Name] = fn
	}

	for _, r := range dbRules {
		if r.RuleType != "composite" {
			continue
		}
		fn, err := compileCompositeNode(r.Rules, fnRules, ac)
		if err != nil {
			return nil, fmt.Errorf("compile composite rule %q: %w", r.Name, err)
		}
		fnRules[r.Name] = fn
	}

	return fnRules, nil
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

func combineRules(op string, fns []iam.FunctionRule) iam.FunctionRule {
	switch op {
	case "AND":
		return func(req iam.AccessRequest) bool {
			for _, fn := range fns {
				if !fn(req) {
					return false
				}
			}
			return true
		}
	case "OR":
		return func(req iam.AccessRequest) bool {
			for _, fn := range fns {
				if fn(req) {
					return true
				}
			}
			return false
		}
	case "NOT":
		return func(req iam.AccessRequest) bool {
			if len(fns) == 0 {
				return true
			}
			return !fns[0](req)
		}
	case "XOR":
		return func(req iam.AccessRequest) bool {
			var count int
			for _, fn := range fns {
				if fn(req) {
					count++
				}
			}
			return count == 1
		}
	default:
		return func(req iam.AccessRequest) bool {
			for _, fn := range fns {
				if !fn(req) {
					return false
				}
			}
			return true
		}
	}
}
