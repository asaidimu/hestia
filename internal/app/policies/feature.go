package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/abstract"
)

type Dependencies struct {
	PolicyModel *PolicyModel
	PermManager core.ReloadablePermissionManager
	LiveRules   iam.RuleSet[iam.FunctionRule]
	LiveOps     collection.LiveCollection[*OperationPolicy]
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		 {Name: "system:policies:operation:get", Handler: NewGetOperationHandler(deps.PolicyModel), Description: "Get policy operation", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: policyOperationGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}, Output: policyOperationOutputSchema()},
		{Name: "system:policies:rule:get", Handler: NewGetRuleHandler(deps.PolicyModel), Description: "Get policy rule", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: policyRuleGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:validate", Handler: NewValidateRuleHandler(), Description: "Validate CEL rule expression", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: policyValidateInputSchema(), Payload: definition.FieldTypeObject}, Output: policyValidateOutputSchema()},
		{Name: "system:policies:operation:upsert", Handler: NewUpsertOperationHandler(deps.PolicyModel, deps.LiveOps), Description: "Create or update policy operation", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policyOperationUpsertInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyOperationOutputSchema()},
		{Name: "system:policies:operation:delete", Handler: NewDeleteOperationHandler(deps.PolicyModel, deps.LiveOps), Description: "Delete policy operation", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: policyOperationDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}},
		{Name: "system:policies:rule:upsert", Handler: NewUpsertRuleHandler(deps.PolicyModel, deps.LiveRules), Description: "Create or update policy rule", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policyRuleUpsertInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name", Payload: definition.FieldTypeObject}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:delete", Handler: NewDeleteRuleHandler(deps.PolicyModel, deps.LiveRules), Description: "Delete policy rule", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: policyRuleDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "name", Type: definition.FieldTypeString}}, ResourceIDField: "name"}},
		{Name: "system:policies:rule:reload", Handler: NewReloadPoliciesHandler(deps.PolicyModel, deps.PermManager, deps.LiveRules), Description: "Reload policies from database", Enabled: true, Intent: registration.Read, Output: policyReloadOutputSchema()},
	}
}
