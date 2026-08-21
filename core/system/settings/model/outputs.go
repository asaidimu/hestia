// @note #cruft-20260821-012 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema functions in settings/model/outputs.go
// @see #8uuufn
//
// The schema functions (settingOutputSchema, settingsListOutputSchema,
// messageOutputSchema) are dead code. The generated registrations use
// dispatch.SchemaFromType directly.
//
// Resolution: remove the schema functions. The output types themselves are
// still used by the service methods and registrations.
package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// SettingDocumentView is the wire shape of a single setting.
type SettingDocumentView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Key                    string `anansi:"key"`
	Value                  any    `anansi:"value"`
}

// SettingOutput declares the single-setting schema.
type SettingOutput struct {
	Document SettingDocumentView `anansi:"document"`
}

func settingOutputSchema() *definition.Schema { return dispatch.SchemaFromType[SettingOutput]() }

// SettingsListOutput declares the settings list schema.
type SettingsListOutput struct {
	Documents []SettingDocumentView `anansi:"documents"`
}

func settingsListOutputSchema() *definition.Schema { return dispatch.SchemaFromType[SettingsListOutput]() }

// MessageOutput declares a simple status message response.
type MessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}

func messageOutputSchema() *definition.Schema { return dispatch.SchemaFromType[MessageOutput]() }