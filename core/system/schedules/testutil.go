package schedules

import (
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/schedules/model"
)

// NewSchedulesServiceForTest creates a SchedulesService with explicit dependencies for testing.
func NewSchedulesServiceForTest(m *model.SystemScheduledMessagess, live *LiveSchedule) *SchedulesService {
	return &SchedulesService{model: m, live: live}
}

// NewSchedulesServiceWithRegistrationsForTest creates a SchedulesService bound
// to an explicit message catalog, so target/schema validation is exercised.
func NewSchedulesServiceWithRegistrationsForTest(m *model.SystemScheduledMessagess, live *LiveSchedule, regs []abstract.MessageRegistration) *SchedulesService {
	return &SchedulesService{model: m, live: live, registrations: &regs}
}
