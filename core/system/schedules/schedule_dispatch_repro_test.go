package schedules_test

import (
        "context"
        "sync/atomic"
        "testing"
        "time"

        "github.com/asaidimu/go-anansi/v8/core/data"

        "github.com/asaidimu/hestia/core/abstract"
        runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
        "github.com/asaidimu/hestia/core/runtime/scheduler"
        schedulespkg "github.com/asaidimu/hestia/core/system/schedules"
        "github.com/asaidimu/hestia/core/system/schedules/model"
        dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
        "github.com/asaidimu/hestia/core/runtime"

        "go.uber.org/zap"
)

// TestScheduleDispatchesNotification verifies the full round-trip:
//  1. A schedule targeting system:notifications:notification:create is persisted.
//  2. LiveSchedule.Init loads it and registers it with the Scheduler.
//  3. The cron fires and dispatches the message through the LocalDispatcher.
//  4. The notification handler is actually invoked with the correct payload.
//
// This reproduces the user's scenario: a "* * * * *" schedule that creates
// a notification with subject, body, type, and user_id.
// TestScheduleDispatchesNotificationFast is the CI-friendly version of
// TestScheduleDispatchesNotification. Uses @every 1s instead of * * * * *
// so it completes in ~2s instead of up to 70s.
func TestScheduleDispatchesNotificationFast(t *testing.T) {
        testScheduleDispatchesNotification(t, "@every 1s", 5*time.Second)
}

// TestScheduleDispatchesNotification is the exact repro using the user's
// cron expression. It waits up to 70s for the next minute boundary.
// Run with: go test -run TestScheduleDispatchesNotification$ -v -timeout 120s
func TestScheduleDispatchesNotification(t *testing.T) {
        testScheduleDispatchesNotification(t, "* * * * *", 70*time.Second)
}

func testScheduleDispatchesNotification(t *testing.T, cronExpr string, wait time.Duration) {
        ctx := context.Background()
        log := zap.NewNop()

        schedModel := newTestModel(t)

        disp := runtime.NewLocalDispatcher()

        // Track how many times the handler was invoked and capture the payload.
        var callCount atomic.Int64
        var lastPayload atomic.Value // stores map[string]any

        err := disp.RegisterHandler("system:notifications:notification:create",
                dispatch.Handle(func(ctx context.Context, msg abstract.Message, input *notifCreateDTO) (*abstract.Result, error) {
                        callCount.Add(1)
                        lastPayload.Store(map[string]any{
                                "user_id": input.UserID,
                                "subject": input.Subject,
                                "body":   input.Body,
                                "type":   input.Type,
                        })
                        return &abstract.Result{}, nil
                }),
                abstract.HandlerInfo{Description: "test notification create", Enabled: true},
        )
        if err != nil {
                t.Fatalf("register handler: %v", err)
        }

        // Create the schedule document matching the user's data exactly.
        scheduleDoc := data.MustNewDocument(map[string]any{
                "user_id":  "01a0195b88267850bee3fce1af080afe",
                "message":  "system:notifications:notification:create",
                "input": map[string]any{
                        "body":    "Step away from the screen. Stretch, hydrate, rest your eyes.",
                        "subject": "Time for a break!",
                        "type":    "reminder",
                        "user_id": "01a0195b88267850bee3fce1af080afe",
                },
                "cron":     cronExpr,
                "disabled": false,
        })

        saved, err := schedModel.CreateSchedule(ctx, scheduleDoc)
        if err != nil {
                t.Fatalf("create schedule: %v", err)
        }
        t.Logf("created schedule %s with cron %q", saved.ID(), cronExpr)

        schedCtx, schedCancel := context.WithCancel(context.Background())
        defer schedCancel()

        sched := scheduler.New(schedCtx, log)
        live := schedulespkg.NewLiveSchedule(schedModel, sched, disp, log)

        if err := live.Init(ctx); err != nil {
                t.Fatalf("LiveSchedule.Init: %v", err)
        }

        jobs := sched.List()
        found := false
        for _, j := range jobs {
                if j.Name == "schedule:"+saved.ID() {
                        found = true
                        t.Logf("job registered: name=%s expr=%s next=%s", j.Name, j.Expr, j.Next)
                        break
                }
        }
        if !found {
                t.Fatal("schedule was not registered with the cron scheduler after Init")
        }

        sched.Start()
        defer sched.Stop()

        // Wait for the cron to fire.
        deadline := time.Now().Add(wait)
        for time.Now().Before(deadline) {
                if callCount.Load() > 0 {
                        break
                }
                time.Sleep(500 * time.Millisecond)
        }

        calls := callCount.Load()
        t.Logf("handler was invoked %d time(s)", calls)
        if calls == 0 {
                t.Fatal("notification handler was never invoked — schedule did not fire")
        }

        v := lastPayload.Load()
        if v == nil {
                t.Fatal("no captured payload")
        }
        payload, ok := v.(map[string]any)
        if !ok {
                t.Fatalf("payload is not a map: %T", v)
        }
        if payload["subject"] != "Time for a break!" {
                t.Errorf("subject = %v, want %q", payload["subject"], "Time for a break!")
        }
        if payload["user_id"] != "01a0195b88267850bee3fce1af080afe" {
                t.Errorf("user_id = %v, want %q", payload["user_id"], "01a0195b88267850bee3fce1af080afe")
        }
        t.Logf("payload verified: %+v", payload)
}

// TestScheduleNotLoadedWhenInsertedAfterInit reproduces the user's likely
// root cause: inserting a schedule document directly into the DB after
// LiveSchedule.Init has already run. The scheduler should NOT fire for
// that document because LiveSchedule never registered it.
func TestScheduleNotLoadedWhenInsertedAfterInit(t *testing.T) {
        ctx := context.Background()
        log := zap.NewNop()

        schedModel := newTestModel(t)
        schedCtx, schedCancel := context.WithCancel(context.Background())
        defer schedCancel()

        sched := scheduler.New(schedCtx, log)
        disp := runtime.NewLocalDispatcher()
        live := schedulespkg.NewLiveSchedule(schedModel, sched, disp, log)

        if err := live.Init(ctx); err != nil {
                t.Fatalf("LiveSchedule.Init: %v", err)
        }
        sched.Start()
        defer sched.Stop()

        if len(sched.List()) != 0 {
                t.Fatalf("expected 0 jobs after init, got %d", len(sched.List()))
        }

        // Insert a schedule directly into the DB (simulating a direct DB write).
        sneakyDoc := data.MustNewDocument(map[string]any{
                "user_id":  "user-sneaky",
                "message":  "system:test:handler",
                "input":    map[string]any{"token": "abc"},
                "cron":     "* * * * *",
                "disabled": false,
        })
        saved, err := schedModel.CreateSchedule(ctx, sneakyDoc)
        if err != nil {
                t.Fatalf("direct DB insert: %v", err)
        }
        t.Logf("inserted schedule %s directly into DB (bypassing LiveSchedule.Register)", saved.ID())

        for _, j := range sched.List() {
                if j.Name == "schedule:"+saved.ID() {
                        t.Error("schedule was registered, but it should NOT have been (inserted after Init)")
                        return
                }
        }
        t.Log("confirmed: schedule in DB but not registered with cron engine")
}

// TestScheduleRegisteredViaServiceCreate verifies that creating a schedule
// through the service (the API path) DOES immediately register it with the
// live scheduler, even after Init has already run.
func TestScheduleRegisteredViaServiceCreate(t *testing.T) {
        ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "user-1"})
        log := zap.NewNop()

        schedModel := newTestModel(t)
        schedCtx, schedCancel := context.WithCancel(context.Background())
        defer schedCancel()

        sched := scheduler.New(schedCtx, log)
        disp := runtime.NewLocalDispatcher()
        live := schedulespkg.NewLiveSchedule(schedModel, sched, disp, log)

        if err := live.Init(ctx); err != nil {
                t.Fatalf("LiveSchedule.Init: %v", err)
        }
        sched.Start()
        defer sched.Stop()

        svc := schedulespkg.NewSchedulesServiceForTest(schedModel, live)
        result, err := svc.Create(ctx, &testMessage{}, &model.ScheduleCreateInput{
                Message: "system:test:handler",
                Input:   map[string]any{"token": "xyz"},
                Cron:    "@every 1h",
        })
        if err != nil {
                t.Fatalf("service create: %v", err)
        }
        if result == nil || result.ID == "" {
                t.Fatal("expected created schedule with ID")
        }

        found := false
        for _, j := range sched.List() {
                if j.Name == "schedule:"+result.ID {
                        found = true
                        t.Logf("schedule %s registered at runtime via service", result.ID)
                        break
                }
        }
        if !found {
                t.Error("schedule created via service was NOT registered with the scheduler")
        }
}

// notifCreateDTO mirrors the fields that LiveSchedule.dispatch wraps in
// {"payload": ...}. The input tags must match what the real
// NotificationCreateInput uses (payload.subject, payload.user_id, etc.)
// so that dispatch.Handle binds the envelope correctly.
type notifCreateDTO struct {
        UserID string  `anansi:"user_id,required=true" input:"payload.user_id"`
        Subject string  `anansi:"subject,required=true" input:"payload.subject"`
        Body   string  `anansi:"body,required=false" input:"payload.body"`
        Type   string  `anansi:"type,required=false" input:"payload.type"`
}
