// @note #cruft-20260821-006 issue status=open priority=P1 tags=#cruft,#duplicate : Duplicate input types in settings/inputs.go and settings/model/inputs.go
// @see #8uuufn
//
// This file defines SettingKeyInput and SetSettingInput, which are duplicated
// in settings/model/inputs.go. The generated registrations use the model/
// versions. Additionally, SettingListInput is defined only in model/inputs.go.
//
// The schema functions (SettingKeyInputSchema, SetSettingInputSchema) are
// dead code — the registrations use dispatch.SchemaFromTypeWithTag directly.
//
// Resolution: delete this file entirely; all consumers should import from
// settings/model.
package settings

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type SettingKeyInput struct {
	Key string `input:"arguments.key"`
}

type SetSettingInput struct {
	Key   string `input:"arguments.key"`
	Value any    `input:"payload.value"`
}

func SettingKeyInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[SettingKeyInput]("input", true) }
func SetSettingInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[SetSettingInput]("input", true) }
