package schedules

import (
	"github.com/asaidimu/hestia/core/system/schedules/model"
)

// NewSchedulesServiceForTest creates a SchedulesService with explicit dependencies for testing.
func NewSchedulesServiceForTest(m *model.ScheduleModel, live *LiveSchedule) *SchedulesService {
	return &SchedulesService{model: m, live: live}
}
