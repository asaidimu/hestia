// @note #cruft-20260821-011 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema functions in settings/model/inputs.go
// @see #8uuufn
//
// The schema functions (SettingKeyInputSchema, SetSettingInputSchema) are
// dead code. The generated registrations use dispatch.SchemaFromTypeWithTag
// directly.
//
// Resolution: remove the schema functions. The input types themselves are
// still used by the service methods and registrations.
package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type SettingListInput struct{}

type SettingKeyInput struct {
	Key string `input:"arguments.key"`
}

type SetSettingInput struct {
	Key   string `input:"arguments.key"`
	Value any    `input:"payload.value"`
}

func SettingKeyInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[SettingKeyInput]("input", true) }
func SetSettingInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[SetSettingInput]("input", true) }