package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// CapabilityNameInput is the input for system:core:capability:set.
type CapabilityNameInput struct {
	Name    string `input:"arguments.name"`
	Enabled bool   `input:"payload.enabled"`
}

func CapabilityNameInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[CapabilityNameInput]("input", true)
}
