package settings

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_setSettingInput    = dispatch.MustFromJSON(setSettingInputJSON)
	_settingOutput      = dispatch.MustFromJSON(settingOutputJSON)
	_settingsListOutput = dispatch.MustFromJSON(settingsListOutputJSON)
	_messageOutput      = dispatch.MustFromJSON(messageOutputJSON)
)

func setSettingInputSchema() *definition.Schema    { return _setSettingInput }
func settingOutputSchema() *definition.Schema      { return _settingOutput }
func settingsListOutputSchema() *definition.Schema { return _settingsListOutput }
func messageOutputSchema() *definition.Schema      { return _messageOutput }

var setSettingInputJSON = []byte(`{
	"version": "1.0.0",
	"name": "set_setting_input",
	"description": "Input for setting a value",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "set_setting_payload" }
		}
	},
	"schemas": {
		"set_setting_payload": {
			"name": "SetSettingPayload",
			"fields": {
				"value": { "name": "value", "type": "any" }
			}
		}
	}
}`)

var settingOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "setting_output",
	"description": "A key-value pair",
	"fields": {
		"document": {
			"name": "document",
			"type": "object",
			"schema": { "id": "setting_document" }
		}
	},
	"schemas": {
		"setting_document": {
			"name": "SettingDocument",
			"fields": {
				"key": { "name": "key", "type": "string" },
				"value": { "name": "value", "type": "any" }
			}
		}
	}
}`)

var settingsListOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "settings_list_output",
	"description": "List of settings as key-value pairs",
	"fields": {
		"documents": {
			"name": "documents",
			"type": "array",
			"schema": { "id": "setting_document" }
		}
	},
	"schemas": {
		"setting_document": {
			"name": "SettingDocument",
			"fields": {
				"key": { "name": "key", "type": "string" },
				"value": { "name": "value", "type": "any" }
			}
		}
	}
}`)

var messageOutputJSON = []byte(`{
	"version": "1.0.0",
	"name": "message_output",
	"description": "Generic message response",
	"fields": {
		"message": {
			"name": "message",
			"type": "string"
		}
	}
}`)
