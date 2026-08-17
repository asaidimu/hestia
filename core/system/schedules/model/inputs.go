package model

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

type ScheduleListInput struct{}

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
