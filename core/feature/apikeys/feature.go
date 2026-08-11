package apikeys

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/apikeys/model"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	APIKeyModel *model.SystemAPIKeys
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:apikeys:key:list", Handler: NewListAPIKeysHandler(deps.APIKeyModel), Description: "List API keys", Enabled: true, Intent: abstract.Read, Output: model.APIKeyListOutputSchema()},
		{Name: "system:apikeys:key:create", Handler: NewCreateAPIKeyHandler(deps.APIKeyModel), Description: "Create an API key", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: model.APIKeyCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: model.APIKeyOutputSchema()},
		{Name: "system:apikeys:key:get", Handler: NewGetAPIKeyHandler(deps.APIKeyModel), Description: "Get API key", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Schema: model.APIKeyGetInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id"}, Output: model.APIKeyOutputSchema()},
		{Name: "system:apikeys:key:update", Handler: NewUpdateAPIKeyHandler(deps.APIKeyModel), Description: "Update API key", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: model.APIKeyUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id", Payload: definition.FieldTypeObject}, Output: model.APIKeyOutputSchema()},
		{Name: "system:apikeys:key:delete", Handler: NewDeleteAPIKeyHandler(deps.APIKeyModel), Description: "Delete API key", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Schema: model.APIKeyDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id"}},
		{Name: "system:apikeys:key:rotate", Handler: NewRotateAPIKeyHandler(deps.APIKeyModel), Description: "Rotate API key", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: model.APIKeyRotateInputSchema(), Arguments: []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}}, ResourceIDField: "key_id"}, Output: model.APIKeyOutputSchema()},
	}
}
