package policies

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Dependencies struct {
	PolicyModel      *PolicyModel
	PermManager      core.ReloadablePermissionManager
	AccessController iam.AccessController
	CompileRules     func(iam.AccessController, []PolicyRule) (iam.FunctionRuleSet, error)
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:policies:operation:get", Handler: NewGetOperationHandler(deps.PolicyModel), Description: "Get policy operation", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: policyOperationGetInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}}, Output: policyOperationOutputSchema()},
		{Name: "system:policies:rule:get", Handler: NewGetRuleHandler(deps.PolicyModel), Description: "Get policy rule", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: policyRuleGetInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:validate", Handler: NewValidateRuleHandler(), Description: "Validate CEL rule expression", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: policyValidateInputSchema(), Payload: definition.FieldTypeObject}, Output: policyValidateOutputSchema()},
		{Name: "system:policies:operation:upsert", Handler: NewUpsertOperationHandler(deps.PolicyModel), Description: "Create or update policy operation", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policyOperationUpsertInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: policyOperationOutputSchema()},
		{Name: "system:policies:operation:delete", Handler: NewDeleteOperationHandler(deps.PolicyModel), Description: "Delete policy operation", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: policyOperationDeleteInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}}},
		{Name: "system:policies:rule:upsert", Handler: NewUpsertRuleHandler(deps.PolicyModel), Description: "Create or update policy rule", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: policyRuleUpsertInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: policyRuleOutputSchema()},
		{Name: "system:policies:rule:delete", Handler: NewDeleteRuleHandler(deps.PolicyModel), Description: "Delete policy rule", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: policyRuleDeleteInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}}},
		{Name: "system:policies:rule:reload", Handler: NewReloadPoliciesHandler(deps.PolicyModel, deps.PermManager, deps.AccessController, deps.CompileRules), Description: "Reload policies from database", Enabled: true, Intent: registration.Read, Output: policyReloadOutputSchema()},
	}
}
