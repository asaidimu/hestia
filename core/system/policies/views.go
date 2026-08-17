package policies

import "github.com/asaidimu/hestia/core/system/policies/model"

func policyToView(p Policy) model.PolicyView {
	v := model.PolicyView{
		ID:        p.ID,
		Operation: p.Operation,
		Rule:      p.Rule,
		Enabled:   p.Enabled,
	}
	if p.RateLimit != nil {
		v.RateLimit = &model.RateLimitView{
			Enabled:  p.RateLimit.Enabled,
			Identity: p.RateLimit.Identity,
			Capacity: p.RateLimit.Capacity,
			Refill:   p.RateLimit.Refill,
			Period:   p.RateLimit.Period,
		}
	}
	if p.Throttle != nil {
		tv := &model.ThrottleView{Limit: p.Throttle.Limit, Window: p.Throttle.Window}
		if p.Throttle.Action != nil {
			tv.Action = &model.ThrottleActionView{
				Message: p.Throttle.Action.Message,
				Input:   p.Throttle.Action.Input,
			}
		}
		v.Throttle = tv
	}
	return v
}

func ruleToView(r PolicyRule) model.PolicyRuleView {
	v := model.PolicyRuleView{
		ID:          r.ID,
		Name:        r.Name,
		RuleType:    r.RuleType,
		Syntax:      r.Syntax,
		Expression:  r.Expression,
		Description: r.Description,
		Protected:   r.Protected,
	}
	if r.Rules != nil {
		v.Rules = ruleNodeToView(r.Rules)
	}
	return v
}

func ruleNodeToView(n *RuleNode) *model.RuleNodeView {
	if n == nil {
		return nil
	}
	v := &model.RuleNodeView{
		Type:       n.Type,
		Name:       n.Name,
		Expression: n.Expression,
		Operator:   n.Operator,
	}
	if len(n.Conditions) > 0 {
		v.Conditions = make([]model.RuleNodeView, len(n.Conditions))
		for i := range n.Conditions {
			v.Conditions[i] = *ruleNodeToView(&n.Conditions[i])
		}
	}
	return v
}

// RuleToView converts a domain rule to its wire view.
func RuleToView(r PolicyRule) model.PolicyRuleView {
	return ruleToView(r)
}

// PolicyToView converts a domain policy to its wire view.
func PolicyToView(p Policy) model.PolicyView {
	return policyToView(p)
}