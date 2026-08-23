package http

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/core/abstract"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime"
)

// slowLocal wires a real LocalDispatcher to a handler whose progress the test
// controls, so tests can prove the transport returned before completion.
type slowLocal struct {
	disp      *runtime.LocalDispatcher
	started   chan struct{}
	release   chan struct{}
	completed chan struct{}
}

func newSlowLocal(t *testing.T) *slowLocal {
	t.Helper()
	sl := &slowLocal{
		started:   make(chan struct{}),
		release:   make(chan struct{}),
		completed: make(chan struct{}),
	}
	sl.disp = runtime.NewLocalDispatcherWithLogger(nil)
	handler := func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		close(sl.started)
		<-sl.release
		close(sl.completed)
		return &abstract.Result{}, nil
	}
	if err := sl.disp.RegisterHandler("payments:mpesa:callback:create", handler, abstract.HandlerInfo{
		Name: "payments:mpesa:callback:create", Enabled: true,
	}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	return sl
}

func fireAndForgetMock(t *testing.T, sl *slowLocal, reg abstract.MessageRegistration) Handler {
	t.Helper()
	mt := newMockTransport()
	orch := &Interface{
		trans:        mt,
		disp:         sl.disp,
		regs:         []abstract.MessageRegistration{reg},
		bootstrapped: true,
	}
	orch.installDispatcherRegistrations()

	mt.mu.Lock()
	defer mt.mu.Unlock()
	h, ok := mt.handlers["POST /payments/mpesa/callback/create"]
	if !ok {
		t.Fatal("route not installed")
	}
	return h
}

func claimsContext() context.Context {
	return runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "u1"})
}

// TestFireAndForgetRegistrationReturnsAccepted pins the webhook contract: a
// registration flagged FireAndForget answers 202 with a correlation ID as
// soon as the dispatcher chain accepts the message — without waiting for
// handler completion.
func TestFireAndForgetRegistrationReturnsAccepted(t *testing.T) {
	sl := newSlowLocal(t)
	h := fireAndForgetMock(t, sl, abstract.MessageRegistration{
		Name:          "payments:mpesa:callback:create",
		Intent:        abstract.Create,
		FireAndForget: true,
	})

	done := make(chan Response, 1)
	go func() {
		resp, err := h(claimsContext(), Request{})
		if err != nil {
			t.Errorf("handler error: %v", err)
		}
		done <- resp
	}()

	select {
	case <-sl.started:
	case <-time.After(time.Second):
		t.Fatal("handler never started")
	}

	var resp Response
	select {
	case resp = <-done:
	case <-time.After(time.Second):
		t.Fatal("fire-and-forget request blocked until completion")
	}
	if resp.Status != statusAccepted {
		t.Fatalf("status = %d, want 202", resp.Status)
	}

	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", resp.Body)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("body[data] type = %T", body["data"])
	}
	id, _ := data["id"].(string)
	if id == "" {
		t.Fatalf("body[data][id] = %v, want non-empty correlation id", data["id"])
	}
	if status, _ := data["status"].(string); status != "accepted" {
		t.Fatalf("body[data][status] = %v, want accepted", data["status"])
	}

	// Handler must still be running: the response beat completion.
	select {
	case <-sl.completed:
		t.Fatal("handler completed before response was returned")
	default:
	}

	close(sl.release)
	select {
	case <-sl.completed:
	case <-time.After(time.Second):
		t.Fatal("handler never completed after release")
	}
}

// TestFireAndForgetUsesIdempotencyKeyAsID pins that an inbound
// Idempotency-Key becomes the accepted message's correlation ID.
func TestFireAndForgetUsesIdempotencyKeyAsID(t *testing.T) {
	sl := newSlowLocal(t)
	h := fireAndForgetMock(t, sl, abstract.MessageRegistration{
		Name:          "payments:mpesa:callback:create",
		Intent:        abstract.Create,
		FireAndForget: true,
	})

	resp, err := h(claimsContext(), Request{Headers: map[string][]string{"Idempotency-Key": {"mpesa-txn-42"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != statusAccepted {
		t.Fatalf("status = %d, want 202", resp.Status)
	}
	body := resp.Body.(map[string]any)
	data := body["data"].(map[string]any)
	if id, _ := data["id"].(string); id != "mpesa-txn-42" {
		t.Fatalf("id = %q, want mpesa-txn-42", id)
	}

	close(sl.release)
	<-sl.completed
}

// TestFireAndForgetValidationStaysSynchronous pins that input validation for
// fire-and-forget operations fails the request immediately instead of acking
// garbage input.
func TestFireAndForgetValidationStaysSynchronous(t *testing.T) {
	sl := newSlowLocal(t)

	type callbackInput struct {
		Ref string `input:"payload.ref" anansi:"required=true"`
	}
	h := fireAndForgetMock(t, sl, abstract.MessageRegistration{
		Name:          "payments:mpesa:callback:create",
		Intent:        abstract.Create,
		FireAndForget: true,
		Input: runtime.Input{
			Schema: dispatch.SchemaFromTypeWithTag[callbackInput]("input"),
		},
	})

	_, err := h(claimsContext(), Request{}) // empty payload → missing required "ref"
	if err == nil {
		t.Fatal("expected synchronous validation error")
	}
	var sysErr *common.SystemError
	if !errors.As(err, &sysErr) || sysErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}

	// The rejected request must never have reached the dispatcher.
	select {
	case <-sl.started:
		t.Fatal("handler ran for an invalid payload")
	default:
	}
}
