package policies

import (
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