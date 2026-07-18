package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/abstract"
)

type Dependencies struct {
	PolicyModel  *PolicyModel
	PermManager  core.ReloadablePermissionManager
	LiveRules    iam.RuleSet[iam.FunctionRule]
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:policies:operation:get", Handler: NewGetOperationHandler(deps.PolicyModel), Description: "Get operation info", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: policyOperationGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}, Output: policyOperationOutputSchema()},
		{Name: "system:policies:operation:list", Handler: NewListOperationsHandler(deps.PolicyModel), Description: "List all operations", Enabled: true, Intent: registration.Read, Output: policyListOperationsOutputSchema()},
		{Name: "system:policies:rule:get", Handler: NewGetRuleHandler(deps.PolicyModel), Description: "Get a policy rule", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: policyRuleGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:list", Handler: NewListRulesHandler(deps.PolicyModel), Description: "List all rules", Enabled: true, Intent: registration.Read, Output: policyListRulesOutputSchema()},
		{Name: "system:policies:rule:create", Handler: NewCreateRuleHandler(deps.PolicyModel), Description: "Create a policy rule", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: policyRuleCreateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:update", Handler: NewUpdateRuleHandler(deps.PolicyModel), Description: "Update a policy rule", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policyRuleUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:delete", Handler: NewDeleteRuleHandler(deps.PolicyModel), Description: "Delete a policy rule", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: policyRuleDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}},
		{Name: "system:policies:rule:validate", Handler: NewValidateRuleHandler(), Description: "Validate a CEL rule expression", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: policyValidateInputSchema(), Payload: definition.FieldTypeObject}, Output: policyValidateOutputSchema()},
		{Name: "system:policies:rule:reload", Handler: NewReloadPoliciesHandler(deps.PolicyModel, deps.PermManager, deps.LiveRules), Description: "Reload policies from database", Enabled: true, Intent: registration.Read, Output: policyReloadOutputSchema()},
		{Name: "system:policies:policy:create", Handler: NewCreatePolicyHandler(deps.PolicyModel), Description: "Create a policy binding", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: policyCreateInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyOutputSchema()},
		{Name: "system:policies:policy:update-rule", Handler: NewUpdatePolicyRuleHandler(deps.PolicyModel), Description: "Change the rule bound to an operation", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policyUpdateRuleInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyOutputSchema()},
		{Name: "system:policies:policy:set-enabled", Handler: NewSetPolicyEnabledHandler(deps.PolicyModel), Description: "Enable or disable a policy", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policySetEnabledInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyOutputSchema()},
		{Name: "system:policies:policy:list", Handler: NewListPoliciesHandler(deps.PolicyModel), Description: "List all policy bindings", Enabled: true, Intent: registration.Read, Output: policyListPoliciesOutputSchema()},
	}
}
