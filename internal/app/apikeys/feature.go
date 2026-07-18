package apikeys

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/abstract"
)

type Dependencies struct {
	APIKeyModel *APIKeyModel
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:apikeys:key:list", Handler: NewListAPIKeysHandler(deps.APIKeyModel), Description: "List API keys", Enabled: true, Intent: registration.Read, Output: apiKeyListOutputSchema()},
		{Name: "system:apikeys:key:create", Handler: NewCreateAPIKeyHandler(deps.APIKeyModel), Description: "Create an API key", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: apiKeyCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: apiKeyOutputSchema()},
		{Name: "system:apikeys:key:get", Handler: NewGetAPIKeyHandler(deps.APIKeyModel), Description: "Get API key", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: apiKeyGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id"}, Output: apiKeyOutputSchema()},
		{Name: "system:apikeys:key:update", Handler: NewUpdateAPIKeyHandler(deps.APIKeyModel), Description: "Update API key", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: apiKeyUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id", Payload: definition.FieldTypeObject}, Output: apiKeyOutputSchema()},
		{Name: "system:apikeys:key:delete", Handler: NewDeleteAPIKeyHandler(deps.APIKeyModel), Description: "Delete API key", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: apiKeyDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id"}},
		{Name: "system:apikeys:key:rotate", Handler: NewRotateAPIKeyHandler(deps.APIKeyModel), Description: "Rotate API key", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: apiKeyRotateInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id"}, Output: apiKeyOutputSchema()},
	}
}
