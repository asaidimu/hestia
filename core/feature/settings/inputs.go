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
