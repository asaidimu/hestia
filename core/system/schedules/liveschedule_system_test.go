package schedules_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	schedulespkg "github.com/asaidimu/hestia/core/system/schedules"
	"go.uber.org/zap"
)

// TestProtectedSystemScheduleRunsAsSystem covers the seeded schedules: they
// have no creator by design (user_id="") and must fire under the SYSTEM
// identity. Before this fix every hourly notification-cleanup tick was logged
// as "creator is missing or no longer active" and skipped.
func TestProtectedSystemScheduleRunsAsSystem(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	log := zap.NewNop()

	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()

	sched := scheduler.New(schedCtx, log)
	disp := runtime.NewLocalDispatcher()

	var calls atomic.Int64
	var system atomic.Bool
	var hasClaims atomic.Bool
	err := disp.RegisterHandler("system:test:handler",
		dispatch.Handle(func(handlerCtx context.Context, msg abstract.Message, input *schedValidateDTO) (*abstract.Result, error) {
			calls.Add(1)
			system.Store(runtime.IsSystemIdentity(handlerCtx))
			_, ok := runtimecontext.ClaimsFromContext(handlerCtx)
			hasClaims.Store(ok)
			return &abstract.Result{}, nil
		}),
		abstract.HandlerInfo{Description: "system schedule test", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}

	protected := true
	saved, err := m.CreateSchedule(ctx, data.MustNewDocument(map[string]any{
		"user_id":   "",
		"message":   "system:test:handler",
		"input":     map[string]any{"token": "t"},
		"cron":      "@every 1s",
		"protected": protected,
	}))
	if err != nil {
		t.Fatalf("create system schedule: %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved schedule")
	}

	live := schedulespkg.NewLiveSchedule(m, sched, disp, log)
	if err := live.Init(ctx); err != nil {
		t.Fatalf("LiveSchedule.Init: %v", err)
	}
	sched.Start()
	defer sched.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && calls.Load() == 0 {
		time.Sleep(100 * time.Millisecond)
	}

	if calls.Load() == 0 {
		t.Fatal("protected system schedule never fired")
	}
	if !system.Load() {
		t.Error("protected system schedule did not dispatch under the SYSTEM identity")
	}
	if hasClaims.Load() {
		t.Error("system schedule must not carry end-user claims")
	}
}

// TestCreatorlessUnprotectedScheduleIsSkipped is the fail-closed half: a row
// with no creator that is not marked protected is corrupt or inserted out of
// band and must never escalate to SYSTEM.
func TestCreatorlessUnprotectedScheduleIsSkipped(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	log := zap.NewNop()

	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()

	sched := scheduler.New(schedCtx, log)
	disp := runtime.NewLocalDispatcher()

	var calls atomic.Int64
	err := disp.RegisterHandler("system:test:handler",
		dispatch.Handle(func(handlerCtx context.Context, msg abstract.Message, input *schedValidateDTO) (*abstract.Result, error) {
			calls.Add(1)
			return &abstract.Result{}, nil
		}),
		abstract.HandlerInfo{Description: "creatorless test", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}

	_, err = m.CreateSchedule(ctx, data.MustNewDocument(map[string]any{
		"user_id": "",
		"message": "system:test:handler",
		"input":   map[string]any{"token": "t"},
		"cron":    "@every 1s",
	}))
	if err != nil {
		t.Fatalf("create creatorless schedule: %v", err)
	}

	live := schedulespkg.NewLiveSchedule(m, sched, disp, log)
	if err := live.Init(ctx); err != nil {
		t.Fatalf("LiveSchedule.Init: %v", err)
	}
	sched.Start()
	defer sched.Stop()

	time.Sleep(2500 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("creatorless unprotected schedule fired %d time(s); it must be skipped", calls.Load())
	}
}

// TestUserScheduleKeepsCreatorIdentity is a regression guard for S-1: normal
// user schedules still fire as the creator, never as SYSTEM.
func TestUserScheduleKeepsCreatorIdentity(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	log := zap.NewNop()

	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()

	sched := scheduler.New(schedCtx, log)
	disp := runtime.NewLocalDispatcher()

	var calls atomic.Int64
	var gotUser atomic.Value
	var system atomic.Bool
	err := disp.RegisterHandler("system:test:handler",
		dispatch.Handle(func(handlerCtx context.Context, msg abstract.Message, input *schedValidateDTO) (*abstract.Result, error) {
			calls.Add(1)
			system.Store(runtime.IsSystemIdentity(handlerCtx))
			if claims, ok := runtimecontext.ClaimsFromContext(handlerCtx); ok {
				gotUser.Store(claims.UserID)
			}
			return &abstract.Result{}, nil
		}),
		abstract.HandlerInfo{Description: "user schedule test", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}

	_, err = m.CreateSchedule(ctx, data.MustNewDocument(map[string]any{
		"user_id": "user-1",
		"message": "system:test:handler",
		"input":   map[string]any{"token": "t"},
		"cron":    "@every 1s",
	}))
	if err != nil {
		t.Fatalf("create user schedule: %v", err)
	}

	live := schedulespkg.NewLiveSchedule(m, sched, disp, log)
	if err := live.Init(ctx); err != nil {
		t.Fatalf("LiveSchedule.Init: %v", err)
	}
	sched.Start()
	defer sched.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && calls.Load() == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	if calls.Load() == 0 {
		t.Fatal("user schedule never fired")
	}
	if system.Load() {
		t.Error("user schedule must not dispatch as SYSTEM")
	}
	if got := gotUser.Load(); got != "user-1" {
		t.Errorf("creator identity = %v, want user-1", got)
	}
}
