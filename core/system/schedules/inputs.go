// @note #cruft-20260821-021 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema functions in schedules/inputs.go
// @see #8uuufn
//
// The schema functions (ScheduleCreateInputSchema, ScheduleGetInputSchema,
// ScheduleUpdateInputSchema, ScheduleDeleteInputSchema) are dead code. The
// generated registrations use dispatch.SchemaFromTypeWithTag directly.
//
// Resolution: remove the schema functions. The input types themselves are
// still used by the service methods and registrations.
package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type ScheduleCreateInput struct {
	UserID   string         `input:"payload.user_id"`
	Message  string         `input:"payload.message"`
	Input    map[string]any `input:"payload.input"`
	Cron     string         `input:"payload.cron"`
	Disabled bool           `input:"payload.disabled"`
}

type ScheduleGetInput struct {
	ID string `input:"arguments.id"`
}

type ScheduleUpdateInput struct {
	ID       string         `input:"arguments.id"`
	Message  string         `input:"payload.message"`
	Input    map[string]any `input:"payload.input"`
	Cron     string         `input:"payload.cron"`
	Disabled bool           `input:"payload.disabled"`
}

type ScheduleDeleteInput struct {
	ID string `input:"arguments.id"`
}

func ScheduleCreateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[ScheduleCreateInput]("input", true) }
func ScheduleGetInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[ScheduleGetInput]("input", true) }
func ScheduleUpdateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[ScheduleUpdateInput]("input", true) }
func ScheduleDeleteInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[ScheduleDeleteInput]("input", true) }
