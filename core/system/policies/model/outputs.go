package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// BindingView is the wire shape of a registered operation binding.
type BindingView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string `anansi:"name"`
	Description            string `anansi:"description"`
}

// BindingOutput is the envelope declaring the binding schema.
type BindingOutput struct {
	Document BindingView `anansi:"document"`
}

func policyBindingOutputSchema() *definition.Schema { return dispatch.SchemaFromType[BindingOutput]() }

// BindingListDocument is the body of a bindings list response.
type BindingListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Bindings               []BindingView `anansi:"bindings"`
}

// BindingListOutput is the envelope declaring the bindings list schema.
type BindingListOutput struct {
	Document BindingListDocument `anansi:"document"`
}

func policyListBindingsOutputSchema() *definition.Schema { return dispatch.SchemaFromType[BindingListOutput]() }

// RuleNodeView is the wire shape of a node in a composite rule tree.
type RuleNodeView struct {
	Type       string         `anansi:"type,omitempty"`
	Name       string         `anansi:"name,omitempty"`
	Expression string         `anansi:"expression,omitempty"`
	Operator   string         `anansi:"operator,omitempty"`
	Conditions []RuleNodeView `anansi:"conditions,omitempty"`
}

// PolicyRuleView is the wire shape of a policy rule.
type PolicyRuleView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string        `anansi:"id"`
	Name                   string        `anansi:"name"`
	RuleType               string        `anansi:"ruleType"`
	Syntax                 string        `anansi:"syntax,omitempty"`
	Expression             string        `anansi:"expression,omitempty"`
	Rules                  *RuleNodeView `anansi:"rules,omitempty"`
	Description            string        `anansi:"description,omitempty"`
	Protected              bool          `anansi:"protected"`
}

// PolicyRuleOutput is the envelope declaring the policy rule schema.
type PolicyRuleOutput struct {
	Document PolicyRuleView `anansi:"document"`
}

func policyRuleOutputSchema() *definition.Schema { return dispatch.SchemaFromType[PolicyRuleOutput]() }

// RuleListDocument is the body of a rules list response.
type RuleListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Rules                  []PolicyRuleView `anansi:"rules"`
}

// RuleListOutput is the envelope declaring the rules list schema.
type RuleListOutput struct {
	Document RuleListDocument `anansi:"document"`
}

func policyListRulesOutputSchema() *definition.Schema { return dispatch.SchemaFromType[RuleListOutput]() }

// RateLimitView is the wire shape of a policy rate-limit config.
type RateLimitView struct {
	Enabled  bool   `anansi:"enabled"`
	Identity string `anansi:"identity"`
	Capacity int64  `anansi:"capacity"`
	Refill   int64  `anansi:"refill"`
	Period   int64  `anansi:"period"`
}

// ThrottleActionView is the wire shape of a throttle action.
type ThrottleActionView struct {
	Message string         `anansi:"message"`
	Input   map[string]any `anansi:"input"`
}

// ThrottleView is the wire shape of a policy throttle config.
type ThrottleView struct {
	Limit  int64              `anansi:"limit"`
	Window int64              `anansi:"window"`
	Action *ThrottleActionView `anansi:"action"`
}

// PolicyView is the wire shape of a policy binding.
type PolicyView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string         `anansi:"id"`
	Operation              string         `anansi:"operation"`
	Rule                   string         `anansi:"rule"`
	Enabled                bool           `anansi:"enabled"`
	RateLimit              *RateLimitView `anansi:"rateLimit,omitempty"`
	Throttle               *ThrottleView  `anansi:"throttle,omitempty"`
}

// PolicyOutput is the envelope declaring the policy schema.
type PolicyOutput struct {
	Document PolicyView `anansi:"document"`
}

func policyOutputSchema() *definition.Schema { return dispatch.SchemaFromType[PolicyOutput]() }

// PolicyListDocument is the body of a policies list response.
type PolicyListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Policies               []PolicyView `anansi:"policies"`
}

// PolicyListOutput is the envelope declaring the policies list schema.
type PolicyListOutput struct {
	Document PolicyListDocument `anansi:"document"`
}

func policyListPoliciesOutputSchema() *definition.Schema { return dispatch.SchemaFromType[PolicyListOutput]() }

// PolicyValidateResult is the wire shape of a rule validation response.
type PolicyValidateResult struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Valid                  bool   `anansi:"valid"`
	Result                 bool   `anansi:"result"`
	Error                  string `anansi:"error,omitempty"`
}

// PolicyValidateOutput is the envelope declaring the rule validation schema.
type PolicyValidateOutput struct {
	Document PolicyValidateResult `anansi:"document"`
}

func policyValidateOutputSchema() *definition.Schema { return dispatch.SchemaFromType[PolicyValidateOutput]() }

// PolicyReloadResult is the wire shape of a policy reload response.
type PolicyReloadResult struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Operations             int `anansi:"operations"`
	Rules                  int `anansi:"rules"`
}

// RuleDeletedResult is the wire shape of a rule deletion response.
type RuleDeletedResult struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
	Name                   string `anansi:"name"`
}

// PolicyReloadOutput is the envelope declaring the policy reload schema.
type PolicyReloadOutput struct {
	Document PolicyReloadResult `anansi:"document"`
}

func policyReloadOutputSchema() *definition.Schema { return dispatch.SchemaFromType[PolicyReloadOutput]() }
