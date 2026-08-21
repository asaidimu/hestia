// @note #cruft-20260821-015 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema function in operations/inputs.go
// @see #8uuufn
//
// The schema function CapabilityNameInputSchema() is dead code. The generated
// registrations use dispatch.SchemaFromTypeWithTag directly.
//
// Resolution: remove the schema function. The CapabilityNameInput type itself
// is still used by the service methods and registrations.
package operations

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
