package schedules

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/hestia/core/system/schedules/model"
)

type systemSchedule struct {
	Message string
	Cron    string
}

// SeedSchedules seeds the initial set of system schedules.
// Idempotent — existing records are left unchanged.
func SeedSchedules(ctx context.Context, m *model.SystemScheduledMessagess) error {
	existing, err := m.ListSchedules(ctx)
	if err != nil {
		return common.SystemErrorFrom(err).WithOperation("SeedSchedules").WithMessage("list existing schedules failed")
	}

	byMessage := make(map[string]bool, len(existing))
	for _, doc := range existing {
		if msg, err := doc.Get("message"); err == nil {
			if s, ok := msg.(string); ok {
				byMessage[s] = true
			}
		}
	}

	seeds := []systemSchedule{
		{Message: "system:notifications:notification:cleanup", Cron: "@every 1h"},
		{Message: "system:auth:token:blocklist:prune", Cron: "@every 1h"},
	}

	for _, s := range seeds {
		if byMessage[s.Message] {
			continue
		}
		protected := true
		doc := &model.SystemScheduledMessages{
			UserID:    "",
			Message:   s.Message,
			Cron:      s.Cron,
			Protected: &protected,
		}
		if _, err := m.Create(ctx, doc); err != nil {
			return common.SystemErrorFrom(err).WithOperation("SeedSchedules").WithPath(s.Message).WithMessage("seed schedule failed")
		}
	}

	return nil
}
