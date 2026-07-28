package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	ScheduleModel *ScheduleModel
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{
			Name:        "system:schedules:schedule:create",
			Handler:     NewCreateScheduleHandler(deps.ScheduleModel),
			Description: "Schedule a message for future delivery",
			Enabled:     true,
			Intent:      abstract.Create,
			Input:       runtime.Input{Schema: scheduleCreateInputSchema()},
			Output:      messageOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:list",
			Handler:     NewListSchedulesHandler(deps.ScheduleModel),
			Description: "List all scheduled messages",
			Enabled:     true,
			Intent:      abstract.Read,
			Output:      schedulesListOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:get",
			Handler:     NewGetScheduleHandler(deps.ScheduleModel),
			Description: "Get a single scheduled message by ID",
			Enabled:     true,
			Intent:      abstract.Read,
			Input:       runtime.Input{Arguments: []abstract.ArgDef{{Name: "id", Type: definition.FieldTypeString}}, ResourceIDField: "id"},
			Output:      scheduleOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:delete",
			Handler:     NewDeleteScheduleHandler(deps.ScheduleModel),
			Description: "Delete a scheduled message",
			Enabled:     true,
			Intent:      abstract.Delete,
			Input:       runtime.Input{Arguments: []abstract.ArgDef{{Name: "id", Type: definition.FieldTypeString}}, ResourceIDField: "id"},
			Output:      messageOutputSchema(),
		},
	}
}
