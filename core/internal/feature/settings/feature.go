package settings

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	SettingsModel *SettingsModel
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:settings:list", Handler: NewListSettingsHandler(deps.SettingsModel), Description: "List all settings", Enabled: true, Intent: abstract.Read, Output: settingsListOutputSchema()},
		{Name: "system:settings:get", Handler: NewGetSettingHandler(deps.SettingsModel), Description: "Get a setting by key", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Arguments: []abstract.ArgDef{{Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}, Output: settingOutputSchema()},
		{Name: "system:settings:set", Handler: NewSetSettingHandler(deps.SettingsModel), Description: "Create or update a setting", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: setSettingInputSchema(), Arguments: []abstract.ArgDef{{Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key", Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:settings:delete", Handler: NewDeleteSettingHandler(deps.SettingsModel), Description: "Delete a setting", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Arguments: []abstract.ArgDef{{Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}},
	}
}
