package schedules_test

import (
	"context"
	"testing"

	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/schedules/model"
	"go.uber.org/zap"
)

func TestSeedSchedules(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	err := schedules.SeedSchedules(ctx, m)
	if err != nil {
		t.Fatalf("SeedSchedules: %v", err)
	}

	// Verify both system schedules were created
	docs, err := m.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 seeded schedules, got %d", len(docs))
	}

	// Verify messages
	byMessage := make(map[string]bool)
	for _, doc := range docs {
		msg, _ := doc.Get("message")
		if s, ok := msg.(string); ok {
			byMessage[s] = true
		}
	}
	if !byMessage["system:notifications:notification:cleanup"] {
		t.Error("missing notifications:cleanup schedule")
	}
	if !byMessage["system:auth:token:blocklist:prune"] {
		t.Error("missing blocklist:prune schedule")
	}

	// Verify protected flag
	for _, doc := range docs {
		v, err := doc.Get("protected")
		if err != nil {
			t.Errorf("Get protected: %v", err)
			continue
		}
		if b, ok := v.(bool); !ok || !b {
			t.Error("expected protected=true for system schedule")
		}
	}
}

func TestSeedSchedulesIdempotent(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	// Seed twice
	if err := schedules.SeedSchedules(ctx, m); err != nil {
		t.Fatalf("first SeedSchedules: %v", err)
	}
	if err := schedules.SeedSchedules(ctx, m); err != nil {
		t.Fatalf("second SeedSchedules: %v", err)
	}

	// Should still only have 2 schedules
	docs, err := m.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 seeded schedules after idempotent seed, got %d", len(docs))
	}
}

func TestSeedSchedulesPreservesExisting(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	// Create a user schedule first
	userDoc := createTestDoc(t)
	_, err := m.CreateSchedule(ctx, userDoc)
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}

	// Seed system schedules
	if err := schedules.SeedSchedules(ctx, m); err != nil {
		t.Fatalf("SeedSchedules: %v", err)
	}

	// Should have 3 schedules total (1 user + 2 system)
	docs, err := m.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 schedules (1 user + 2 system), got %d", len(docs))
	}
}

func TestDeleteProtectedSchedule(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-1"})
	m := newTestModel(t)

	// Seed system schedules (which are protected)
	if err := schedules.SeedSchedules(context.Background(), m); err != nil {
		t.Fatalf("SeedSchedules: %v", err)
	}

	// Get the ID of a system schedule
	docs, err := m.ListSchedules(context.Background())
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("no schedules found")
	}
	systemScheduleID := docs[0].ID()

	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)

	// Try to delete the protected schedule
	_, err = svc.Delete(ctx, &testMessage{}, &model.ScheduleDeleteInput{ID: systemScheduleID})
	if err == nil {
		t.Fatal("expected error when deleting protected schedule")
	}

	// Verify the schedule still exists
	existing, err := m.GetSchedule(context.Background(), systemScheduleID)
	if err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if existing == nil {
		t.Fatal("protected schedule should still exist after denied delete")
	}
}
