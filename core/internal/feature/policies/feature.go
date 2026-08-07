package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	PolicyModel *PolicyModel
	PermManager runtime.ReloadablePermissionManager
	LiveRules   iam.RuleSet[iam.FunctionRule]
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:policies:binding:get", Handler: NewGetBindingHandler(deps.PolicyModel), Description: "Get binding info", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Schema: policyBindingGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}, Output: policyBindingOutputSchema()},
		{Name: "system:policies:binding:list", Handler: NewListBindingsHandler(deps.PolicyModel), Description: "List all bindings", Enabled: true, Intent: abstract.Read, Output: policyListBindingsOutputSchema()},
		{Name: "system:policies:rule:validate", Handler: NewValidateRuleHandler(deps.LiveRules), Description: "Validate a CEL rule expression", Enabled: true, Intent: abstract.Check, Input: runtime.Input{Schema: policyValidateInputSchema(), Payload: definition.FieldTypeObject}, Output: policyValidateOutputSchema()},
		{Name: "system:policies:rule:list", Handler: NewListRulesHandler(deps.PolicyModel), Description: "List all rules", Enabled: true, Intent: abstract.Read, Output: policyListRulesOutputSchema()},
		{Name: "system:policies:rule:get", Handler: NewGetRuleHandler(deps.PolicyModel), Description: "Get a policy rule", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Schema: policyRuleGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:create", Handler: NewCreateRuleHandler(deps.PolicyModel), Description: "Create a policy rule", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: policyRuleCreateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:update", Handler: NewUpdateRuleHandler(deps.PolicyModel), Description: "Update a policy rule", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: policyRuleUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:delete", Handler: NewDeleteRuleHandler(deps.PolicyModel), Description: "Delete a policy rule", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Schema: policyRuleDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}},
		{Name: "system:policies:reload", Handler: NewReloadPoliciesHandler(deps.PolicyModel, deps.PermManager, deps.LiveRules), Description: "Reload policies from database", Enabled: true, Intent: abstract.Read, Output: policyReloadOutputSchema()},
		{Name: "system:policies:policy:create", Handler: NewCreatePolicyHandler(deps.PolicyModel), Description: "Create a policy binding", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: policyCreateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyOutputSchema()},
		{Name: "system:policies:policy:update", Handler: NewUpdatePolicyHandler(deps.PolicyModel), Description: "Update a policy — set ruleName, enabled, or both", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: policyUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyOutputSchema()},
		{Name: "system:policies:policy:list", Handler: NewListPoliciesHandler(deps.PolicyModel), Description: "List all policy bindings", Enabled: true, Intent: abstract.Read, Output: policyListPoliciesOutputSchema()},
	}
}
