package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	notifmodel "github.com/asaidimu/hestia/core/system/notifications/model"
)

// newStreamTestService wires a dispatcher with the notification stream and
// create handlers, backed by a fresh in-memory persistence.
func newStreamTestService(t *testing.T) *runtime.LocalDispatcher {
	t.Helper()
	notifmodel.DangerouslyResetSystemNotificationssModel()
	p := testutil.NewPersistence(t)
	m, err := notifmodel.InitSystemNotificationssModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("init model: %v", err)
	}

	disp := runtime.NewLocalDispatcher()
	svc := &NotificationsService{model: m, persist: p}

	if err := disp.RegisterHandler("system:notifications:notification:stream",
		dispatch.Handle[notifmodel.NotificationStreamInput](svc.Stream),
		abstract.HandlerInfo{Description: "notification stream", Enabled: true},
	); err != nil {
		t.Fatalf("register stream handler: %v", err)
	}
	if err := disp.RegisterHandler("system:notifications:notification:create",
		dispatch.HandleDocument[notifmodel.NotificationCreate, *notifmodel.SystemNotifications](svc.CreateNotification),
		abstract.HandlerInfo{Description: "notification create", Enabled: true},
	); err != nil {
		t.Fatalf("register create handler: %v", err)
	}
	return disp
}

func openNotificationStream(t *testing.T, ctx context.Context, disp *runtime.LocalDispatcher) *abstract.Result {
	t.Helper()
	result, err := dispatch.Dispatch(ctx, disp, dispatch.DispatchInput{
		Name:     "system:notifications:notification:stream",
		Context:  ctx,
		Intent:   abstract.Stream,
		Document: data.MustNewDocument(map[string]any{}, ctx),
	})
	if err != nil {
		t.Fatalf("dispatch stream: %v", err)
	}
	if result == nil || result.DocumentChannel == nil {
		t.Fatal("stream result has no DocumentChannel")
	}
	t.Cleanup(result.Release)
	return result
}

// TestNotificationStreamStaysOpenAfterDispatch is the regression test for the
// cleanup goroutine treating the dispatch readiness signal as a stop signal.
// The stream must stay open until the client's context is cancelled.
func TestNotificationStreamStaysOpenAfterDispatch(t *testing.T) {
	disp := newStreamTestService(t)

	ctx, cancel := context.WithCancel(runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{
		UserID: "user-test",
	}))
	defer cancel()

	result := openNotificationStream(t, ctx, disp)

	select {
	case _, ok := <-result.DocumentChannel:
		if !ok {
			t.Fatal("DocumentChannel closed right after Dispatch returned; stream must stay open")
		}
		t.Fatal("unexpected document on an idle stream")
	case <-time.After(200 * time.Millisecond):
		// Still open, as required.
	}

	cancel()

	select {
	case _, ok := <-result.DocumentChannel:
		if ok {
			t.Fatal("DocumentChannel delivered a value instead of closing after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DocumentChannel did not close after cancellation")
	}
}

// TestNotificationStreamDeliversCreatedNotification proves the end-to-end path:
// a notification created for the streaming user arrives on the stream.
func TestNotificationStreamDeliversCreatedNotification(t *testing.T) {
	disp := newStreamTestService(t)

	const targetUser = "user-42"
	ctx, cancel := context.WithCancel(runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{
		UserID: targetUser,
	}))
	defer cancel()

	result := openNotificationStream(t, ctx, disp)

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
		Name:     "system:notifications:notification:create",
		Context:  createCtx,
		Intent:   abstract.Create,
		Document: createInput,
	})
	if err != nil {
		t.Fatalf("dispatch create: %v", err)
	}
	if createResult != nil {
		createResult.Release()
	}

	select {
	case doc, ok := <-result.DocumentChannel:
		if !ok {
			t.Fatal("DocumentChannel closed before delivering the notification")
		}
		if doc == nil {
			t.Fatal("received a nil notification document")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("created notification never arrived on the stream")
	}
}
