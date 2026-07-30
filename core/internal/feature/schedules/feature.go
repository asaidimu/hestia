package schedules

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	ScheduleModel *ScheduleModel
	LiveSchedule  *LiveSchedule
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	h := NewScheduleHandlers(deps.ScheduleModel, deps.LiveSchedule)

	return []abstract.MessageRegistration{
		{
			Name:        "system:schedules:schedule:create",
			Handler:     h.Create,
			Description: "Create a cron-triggered schedule",
			Enabled:     true,
			Intent:      abstract.Create,
			Input:       runtime.Input{Schema: scheduleCreateInputSchema()},
			Output:      messageOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:list",
			Handler:     h.List,
			Description: "List all schedules",
			Enabled:     true,
			Intent:      abstract.Read,
			Output:      schedulesListOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:get",
			Handler:     h.Get,
			Description: "Get a single schedule by ID",
			Enabled:     true,
			Intent:      abstract.Read,
			Input:       runtime.Input{Arguments: []abstract.ArgDef{{Name: "id", Type: definition.FieldTypeString}}, ResourceIDField: "id"},
			Output:      scheduleOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:update",
			Handler:     h.Update,
			Description: "Update a schedule",
			Enabled:     true,
			Intent:      abstract.Update,
			Input:       runtime.Input{Schema: scheduleUpdateInputSchema()},
			Output:      messageOutputSchema(),
		},
		{
			Name:        "system:schedules:schedule:delete",
			Handler:     h.Delete,
			Description: "Delete a schedule",
			Enabled:     true,
			Intent:      abstract.Delete,
			Input:       runtime.Input{Arguments: []abstract.ArgDef{{Name: "id", Type: definition.FieldTypeString}}, ResourceIDField: "id"},
			Output:      messageOutputSchema(),
		},
	}
}
