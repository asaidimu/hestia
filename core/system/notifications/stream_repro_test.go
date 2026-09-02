package notifications

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime"
	notifmodel "github.com/asaidimu/hestia/core/system/notifications/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
)

// TestNotificationStreamClosesImmediately proves that the notification
// stream's DocumentChannel is closed right after dispatch.Dispatch returns.
// The cleanup goroutine treats the inputCh start signal as a stop signal.
func TestNotificationStreamClosesImmediately(t *testing.T) {
	notifmodel.DangerouslyResetSystemNotificationssModel()
	p := testutil.NewPersistence(t)
	m, err := notifmodel.InitSystemNotificationssModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("init model: %v", err)
	}

	disp := runtime.NewLocalDispatcher()
	svc := &NotificationsService{model: m, persist: p}

	err = disp.RegisterHandler("system:notifications:notification:stream",
		dispatch.Handle[notifmodel.NotificationStreamInput](svc.Stream),
		abstract.HandlerInfo{Description: "notification stream", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}

	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{
		UserID: "user-test",
	})

	streamResult, err := dispatch.Dispatch(ctx, disp, dispatch.DispatchInput{
		Name:    "system:notifications:notification:stream",
		Context: ctx,
		Intent:  abstract.Stream,
		Document: data.MustNewDocument(map[string]any{}, ctx),
	})
	if err != nil {
		t.Fatalf("dispatch stream: %v", err)
	}
	if streamResult == nil || streamResult.DocumentChannel == nil {
		t.Fatal("stream result has no DocumentChannel")
	}
	defer streamResult.Release()

	// Give the cleanup goroutine time to race.
	time.Sleep(200 * time.Millisecond)

	select {
	case _, ok := <-streamResult.DocumentChannel:
		if !ok {
			t.Error("DocumentChannel is closed 200ms after Dispatch returned.")
			t.Error("BUG: NotificationsService.Stream cleanup goroutine closes docCh on inputCh signal (start),")
			t.Error("instead of waiting for ctx.Done() (client disconnect) like AuditService.Stream does.")
		}
	default:
		t.Log("DocumentChannel still open after 200ms (unexpected — stream may work under race conditions)")
	}
}

// TestNotificationStreamNeverDelivers creates a stream, then creates a
// notification for the same user, and verifies the notification does NOT
// arrive on the stream (because the stream closed too early).
func TestNotificationStreamNeverDelivers(t *testing.T) {
	notifmodel.DangerouslyResetSystemNotificationssModel()
	p := testutil.NewPersistence(t)
	m, err := notifmodel.InitSystemNotificationssModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("init model: %v", err)
	}

	disp := runtime.NewLocalDispatcher()
	svc := &NotificationsService{model: m, persist: p}

	err = disp.RegisterHandler("system:notifications:notification:stream",
		dispatch.Handle[notifmodel.NotificationStreamInput](svc.Stream),
		abstract.HandlerInfo{Description: "notification stream", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register stream handler: %v", err)
	}

	err = disp.RegisterHandler("system:notifications:notification:create",
		dispatch.HandleDocument[notifmodel.NotificationCreate, *notifmodel.SystemNotifications](svc.CreateNotification),
		abstract.HandlerInfo{Description: "notification create", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register create handler: %v", err)
	}

	const targetUser = "user-42"

	// Open stream.
	streamCtx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{
		UserID: targetUser,
	})
	streamResult, err := dispatch.Dispatch(streamCtx, disp, dispatch.DispatchInput{
		Name:    "system:notifications:notification:stream",
		Context: streamCtx,
		Intent:  abstract.Stream,
		Document: data.MustNewDocument(map[string]any{}, streamCtx),
	})
	if err != nil {
		t.Fatalf("dispatch stream: %v", err)
	}
	if streamResult == nil || streamResult.DocumentChannel == nil {
		t.Fatal("stream result has no DocumentChannel")
	}
	defer streamResult.Release()

	// Start reading from the stream.
	var received atomic.Int64
	go func() {
		for range streamResult.DocumentChannel {
			received.Add(1)
		}
	}()

	// Small delay to let the stream settle.
	time.Sleep(100 * time.Millisecond)

	// Create a notification for the same user (what a schedule would do).
	createCtx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{
		UserID: targetUser,
	})
	createInput := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"user_id": targetUser,
			"subject": "Time for a break!",
			"body":    "Stretch!",
			"type":    "reminder",
		},
	}, createCtx)

	createResult, err := dispatch.Dispatch(createCtx, disp, dispatch.DispatchInput{
		Name:    "system:notifications:notification:create",
		Context: createCtx,
		Intent:  abstract.Create,
		Document: createInput,
	})
	if err != nil {
		t.Fatalf("dispatch create: %v", err)
	}
	if createResult != nil {
		createResult.Release()
	}
	t.Log("notification created")

	// Wait for the stream reader to finish.
	time.Sleep(500 * time.Millisecond)

	if received.Load() == 0 {
		t.Error("notification did NOT arrive on the stream.")
		t.Error("ROOT CAUSE: NotificationsService.Stream closes docCh immediately on the inputCh start signal.")
		t.Error("FIX: Follow the AuditService.Stream pattern — subscribe AFTER the start signal,")
		t.Error("     then wait for ctx.Done() before closing docCh.")
	} else {
		t.Logf("notification arrived on stream (%d)", received.Load())
	}
}
