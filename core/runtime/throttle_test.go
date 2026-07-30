package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
	"github.com/asaidimu/hestia/core/runtime/templateutil"
)

type actionSpy struct {
	called int
	last   abstract.Message
	mu     sync.Mutex
}

func (s *actionSpy) Send(msg abstract.Message) (*abstract.Result, error) {
	s.mu.Lock()
	s.called++
	s.last = msg
	s.mu.Unlock()
	return &abstract.Result{}, nil
}

func TestThrottleLookupNil(t *testing.T) {
	disp := NewThrottleDispatcher(nil, &actionSpy{}, zap.NewNop())
	next := &mockDispatcher{}
	wrapped := disp.Wrap(next)

	_, err := wrapped.Send(newMessage("any:op"))
	if err != nil {
		t.Fatalf("nil lookup should not block: %v", err)
	}
	if next.sendCalled != 1 {
		t.Fatal("should forward to next")
	}
}

func TestThrottleNoLimitPassesThrough(t *testing.T) {
	lookup := func(op string) *ThrottlePolicy { return nil }
	disp := NewThrottleDispatcher(lookup, &actionSpy{}, zap.NewNop())
	next := &mockDispatcher{}
	wrapped := disp.Wrap(next)

	_, err := wrapped.Send(newMessage("op:test"))
	if err != nil {
		t.Fatalf("no throttle policy should not block: %v", err)
	}
	if next.sendCalled != 1 {
		t.Fatal("should forward to next")
	}
}

func TestThrottleActionFiresAtLimit(t *testing.T) {
	spy := &actionSpy{}
	store := ratestore.New()
	disp := &ThrottleDispatcher{
		lookup: func(op string) *ThrottlePolicy {
			return &ThrottlePolicy{
				Limit:  2,
				Window: 60,
				Action: &ThrottleActionPolicy{
					Message: "test:action",
					Input:   map[string]any{"key": "value"},
				},
			}
		},
		disp:   spy,
		store:  store,
		logger: zap.NewNop(),
		next:   &mockDispatcher{},
	}
	defer store.Close()

	msg := newMessage("op:test")

	// events 1 & 2: count 1-2, both <= limit, no action
	for i := 0; i < 2; i++ {
		_, err := disp.Send(msg)
		if err != nil {
			t.Fatalf("event %d: %v", i+1, err)
		}
	}
	if spy.called != 0 {
		t.Fatalf("expected 0 actions within limit, got %d", spy.called)
	}

	// event 3: count=3 exceeds limit=2, action fires
	_, err := disp.Send(msg)
	if err != nil {
		t.Fatalf("event 3: %v", err)
	}
	if spy.called != 1 {
		t.Fatalf("expected 1 action on 3rd event, got %d", spy.called)
	}

	// event 4: count=4, action fires again
	_, err = disp.Send(msg)
	if err != nil {
		t.Fatalf("event 4: %v", err)
	}
	if spy.called != 2 {
		t.Fatalf("expected 2 actions after 4 events, got %d", spy.called)
	}
	if spy.last.Name() != "test:action" {
		t.Fatalf("expected action message 'test:action', got %q", spy.last.Name())
	}
}

func TestThrottleActionFailureIsLogged(t *testing.T) {
	spy := &actionSpy{}
	disp := NewThrottleDispatcher(func(op string) *ThrottlePolicy {
		return &ThrottlePolicy{
			Limit:  0,
			Window: 60,
			Action: &ThrottleActionPolicy{
				Message: "test:action",
			},
		}
	}, spy, zap.NewNop())
	wrapped := disp.Wrap(&mockDispatcher{})

	_, err := wrapped.Send(newMessage("op:test"))
	if err != nil {
		t.Fatalf("throttle action failure should not block original message: %v", err)
	}
}

func TestThrottleNoActionDefined(t *testing.T) {
	disp := NewThrottleDispatcher(func(op string) *ThrottlePolicy {
		return &ThrottlePolicy{
			Limit:  1,
			Window: 60,
		}
	}, &actionSpy{}, zap.NewNop())
	wrapped := disp.Wrap(&mockDispatcher{})

	_, err := wrapped.Send(newMessage("op:test"))
	if err != nil {
		t.Fatalf("even at limit, no action defined should not block: %v", err)
	}
}

func TestThrottleWindowResetsCount(t *testing.T) {
	restore := ratestore.SetNow(func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	defer restore()

	spy := &actionSpy{}
	disp := NewThrottleDispatcher(func(op string) *ThrottlePolicy {
		return &ThrottlePolicy{
			Limit:  2,
			Window: 10,
			Action: &ThrottleActionPolicy{
				Message: "test:action",
				Input:   map[string]any{"key": "{{ .operation }}"},
			},
		}
	}, spy, zap.NewNop())
	wrapped := disp.Wrap(&mockDispatcher{})

	// 3 events in window: 3rd fires action
	for i := 0; i < 3; i++ {
		wrapped.Send(newMessage("op:test"))
	}
	if spy.called != 1 {
		t.Fatalf("expected 1 action, got %d", spy.called)
	}

	// advance past window
	restore2 := ratestore.SetNow(func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 11, 0, time.UTC)
	})
	defer restore2()

	// 1st event in new window: count resets, no action
	wrapped.Send(newMessage("op:test"))
	if spy.called != 1 {
		t.Fatalf("after window reset, no action expected, got %d", spy.called)
	}

	// 2 more events: triggers action again
	wrapped.Send(newMessage("op:test"))
	wrapped.Send(newMessage("op:test"))
	if spy.called != 2 {
		t.Fatalf("should fire once more after reset, got %d", spy.called)
	}
}

func TestThrottleResolveTemplateClaims(t *testing.T) {
	data := ThrottleTemplateData{
		Claims:    map[string]any{"user_id": "u-42", "email": "test@example.com"},
		SourceIP:  "10.0.0.1",
		Operation: "sys:test:op",
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Test basic template resolution
	m := templateutil.ResolveMap(map[string]any{"id": "{{ .claims.user_id }}"}, data.toMap())
	if m["id"] != "u-42" {
		t.Errorf("expected 'u-42', got %q", m["id"])
	}

	// Test nested template resolution
	tmpl := map[string]any{
		"arguments.id":   "{{ .claims.user_id }}",
		"payload.reason": "rate limit exceeded from {{ .sourceIP }}",
		"metadata.op":    "{{ .operation }}",
		"static":         42,
	}

	resolvedInput := templateutil.ResolveMap(tmpl, data.toMap())

	if resolvedInput["arguments.id"] != "u-42" {
		t.Errorf("claims.user_id: got %q", resolvedInput["arguments.id"])
	}
	if resolvedInput["payload.reason"] != "rate limit exceeded from 10.0.0.1" {
		t.Errorf("source_ip template: got %q", resolvedInput["payload.reason"])
	}
	if resolvedInput["metadata.op"] != "sys:test:op" {
		t.Errorf("operation template: got %q", resolvedInput["metadata.op"])
	}
	if resolvedInput["static"] != 42 {
		t.Errorf("static value: got %v", resolvedInput["static"])
	}
}

func TestThrottleActionMessageName(t *testing.T) {
	spy := &actionSpy{}
	disp := NewThrottleDispatcher(func(op string) *ThrottlePolicy {
		return &ThrottlePolicy{
			Limit:  1,
			Window: 60,
			Action: &ThrottleActionPolicy{
				Message: "custom:action",
				Input:   map[string]any{"key": "value", "uid": "{{ .claims.user_id }}"},
			},
		}
	}, spy, zap.NewNop())
	wrapped := disp.Wrap(&mockDispatcher{})

	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "u-42"})
	msg := &mockMessage{name: "op:test", ctx: ctx}

	// first send: count=1, 1>1=false, no action
	_, err := wrapped.Send(msg)
	if err != nil {
		t.Fatalf("first send: %v", err)
	}
	// second send: count=2, 2>1=true, action fires
	_, err = wrapped.Send(msg)
	if err != nil {
		t.Fatalf("second send: %v", err)
	}

	if spy.called != 1 {
		t.Fatalf("action should be dispatched once, got %d calls", spy.called)
	}
	if spy.last.Name() != "custom:action" {
		t.Fatalf("expected 'custom:action', got %q", spy.last.Name())
	}
	if key, _ := spy.last.Input().GetString("key"); key != "value" {
		t.Fatalf("expected input key 'value', got %q", key)
	}
	if uid, _ := spy.last.Input().GetString("uid"); uid != "u-42" {
		t.Fatalf("expected uid 'u-42', got %q", uid)
	}
}

func TestThrottleBuildTemplateData(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "u-99"})
	msg := &mockMessage{name: "sys:action", ctx: ctx, sourceIP: "1.2.3.4"}

	data := buildThrottleTemplateData(ctx, msg)

	if data.Operation != "sys:action" {
		t.Errorf("operation: got %q", data.Operation)
	}
	if data.SourceIP != "1.2.3.4" {
		t.Errorf("sourceIP: got %q", data.SourceIP)
	}
	if data.Claims["user_id"] != "u-99" {
		t.Errorf("claims.user_id: got %v", data.Claims["user_id"])
	}
}
