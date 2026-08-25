package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
)

var assertAnError = errors.New("assertion error")

type mockMessage struct {
	name     string
	ctx      context.Context
	sourceIP string
	tenantID string
}

func (m *mockMessage) ID() string                             { return "test-id" }
func (m *mockMessage) Name() string                           { return m.name }
func (m *mockMessage) Context() context.Context               { return m.ctx }
func (m *mockMessage) Input() data.Documenter                 { return nil }
func (m *mockMessage) InputChannel() <-chan data.Documenter   { return nil }
func (m *mockMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m *mockMessage) TenantID() string                       { return m.tenantID }
func (m *mockMessage) TraceID() string                        { return "" }
func (m *mockMessage) RequestID() string                      { return "" }
func (m *mockMessage) SourceIP() string                       { return m.sourceIP }
func (m *mockMessage) UserAgent() string                      { return "" }
func (m *mockMessage) ResourceID() string                     { return "" }
func (m *mockMessage) SessionID() string                      { return "" }

type mockDispatcher struct {
	sendCalled int
	lastMsg    abstract.Message
	SendFunc   func(msg abstract.Message) (*abstract.Result, error)
	mu         sync.Mutex
}

func (d *mockDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	d.mu.Lock()
	d.sendCalled++
	d.lastMsg = msg
	d.mu.Unlock()
	var res *abstract.Result
	var err error
	if d.SendFunc != nil {
		res, err = d.SendFunc(msg)
	} else {
		res = &abstract.Result{}
	}
	abstract.Complete(onComplete, ctx, res, err)
	return nil
}

func newMessage(name string) *mockMessage {
	return &mockMessage{name: name, ctx: context.Background()}
}

func msgWithUser(name, userID string) *mockMessage {
	claims := &abstract.Claims{UserID: userID, Scopes: []string{"user"}}
	ctx := runtimecontext.ContextWithClaims(context.Background(), claims)
	return &mockMessage{name: name, ctx: ctx}
}

func msgWithIP(name, ip string) *mockMessage {
	return &mockMessage{name: name, ctx: context.Background(), sourceIP: ip}
}

func msgWithTenant(name, tenantID string) *mockMessage {
	ctx := runtimecontext.ContextWithTenantID(context.Background(), tenantID)
	return &mockMessage{name: name, ctx: ctx, tenantID: tenantID}
}

func TestMatchOperationExact(t *testing.T) {
	if !matchOperation("system:auth:session:create", "system:auth:session:create") {
		t.Error("exact match failed")
	}
	if matchOperation("system:auth:session:create", "system:auth:session:delete") {
		t.Error("exact match should not match different operation")
	}
}

func TestMatchOperationWildcard(t *testing.T) {
	if !matchOperation("system:auth:*", "system:auth:session:create") {
		t.Error("wildcard suffix match failed")
	}
	if matchOperation("system:auth:*", "system:other:thing") {
		t.Error("wildcard should not match different prefix")
	}
}

func TestMatchOperationCatchAll(t *testing.T) {
	if !matchOperation("*", "anything:at:all") {
		t.Error("catch-all should match everything")
	}
	if !matchOperation("", "anything:at:all") {
		t.Error("empty pattern should match everything")
	}
}

func TestBuildKeyUser(t *testing.T) {
	msg := msgWithUser("op:test", "user-42")
	key := buildRateLimitKey("user", msg)
	want := "rate:user:op:test:user-42"
	if key != want {
		t.Errorf("got %q, want %q", key, want)
	}
}

func TestBuildKeyIP(t *testing.T) {
	msg := msgWithIP("op:test", "203.0.113.42")
	key := buildRateLimitKey("ip", msg)
	want := "rate:ip:op:test:203.0.113.42"
	if key != want {
		t.Errorf("got %q, want %q", key, want)
	}
}

func TestBuildKeyTenant(t *testing.T) {
	msg := msgWithTenant("op:test", "tenant-xyz")
	key := buildRateLimitKey("tenant", msg)
	want := "rate:tenant:op:test:tenant-xyz"
	if key != want {
		t.Errorf("got %q, want %q", key, want)
	}
}

func TestBuildKeyGlobal(t *testing.T) {
	msg := newMessage("op:test")
	key := buildRateLimitKey("global", msg)
	want := "rate:global:op:test"
	if key != want {
		t.Errorf("got %q, want %q", key, want)
	}
}

func TestBuildKeyEmptyDimensionFallsBackToGlobal(t *testing.T) {
	msg := newMessage("op:test")
	key := buildRateLimitKey("user", msg)
	want := "rate:global:op:test"
	if key != want {
		t.Errorf("empty userID should fall back to global, got %q, want %q", key, want)
	}
}

func TestRateLimitLookupNilReturnsNil(t *testing.T) {
	disp := NewRateLimitDispatcher(nil, nil)
	next := &mockDispatcher{}
	wrapped := disp.Wrap(next)

	_, err := testAwait(wrapped, newMessage("anything"))
	if err != nil {
		t.Fatalf("nil lookup should not block: %v", err)
	}
	if next.sendCalled != 1 {
		t.Fatal("nil lookup should forward to next")
	}
}

func TestRateLimitLookupNoMatchPassesThrough(t *testing.T) {
	lookup := func(op string) *RateLimitPolicy {
		return nil
	}
	disp := NewRateLimitDispatcher(lookup, nil)
	next := &mockDispatcher{}
	wrapped := disp.Wrap(next)

	_, err := testAwait(wrapped, newMessage("op:test"))
	if err != nil {
		t.Fatalf("no matching policy should not block: %v", err)
	}
	if next.sendCalled != 1 {
		t.Fatal("should forward to next for unmatched operation")
	}
}

func TestRateLimitLookupEnforcesLimit(t *testing.T) {
	lookup := func(op string) *RateLimitPolicy {
		return &RateLimitPolicy{
			Enabled:  true,
			Identity: "global",
			Capacity: 1,
			Refill:   1,
			Period:   60,
		}
	}
	disp := NewRateLimitDispatcher(lookup, nil)
	wrapped := disp.Wrap(&mockDispatcher{})

	_, err := testAwait(wrapped, newMessage("op:test"))
	if err != nil {
		t.Fatalf("first request should be allowed: %v", err)
	}

	_, err = testAwait(wrapped, newMessage("op:test"))
	if err == nil {
		t.Fatal("second request should be rate-limited")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestRateLimitLookupPerUser(t *testing.T) {
	lookup := func(op string) *RateLimitPolicy {
		return &RateLimitPolicy{
			Enabled:  true,
			Identity: "user",
			Capacity: 1,
			Refill:   1,
			Period:   60,
		}
	}
	disp := NewRateLimitDispatcher(lookup, nil)
	wrapped := disp.Wrap(&mockDispatcher{})

	alice := msgWithUser("op:test", "alice")
	bob := msgWithUser("op:test", "bob")

	_, err := testAwait(wrapped, alice)
	if err != nil {
		t.Fatalf("alice first request: %v", err)
	}
	_, err = testAwait(wrapped, alice)
	if err == nil {
		t.Fatal("alice second request should be rate-limited")
	}

	_, err = testAwait(wrapped, bob)
	if err != nil {
		t.Fatalf("bob first request should be allowed (different key): %v", err)
	}
}

func TestRateLimitLookupDifferentOpsDifferentLimits(t *testing.T) {
	lookup := func(op string) *RateLimitPolicy {
		switch op {
		case "limited:op":
			return &RateLimitPolicy{Enabled: true, Identity: "global", Capacity: 1, Refill: 1, Period: 60}
		default:
			return nil
		}
	}
	next := &mockDispatcher{}
	disp := NewRateLimitDispatcher(lookup, nil)
	wrapped := disp.Wrap(next)

	_, err := testAwait(wrapped, newMessage("other:op"))
	if err != nil {
		t.Fatalf("unmatched operation should not be rate-limited: %v", err)
	}
	if next.sendCalled != 1 {
		t.Fatal("should forward to next for unmatched operation")
	}
}

func TestTokenBucketAllowsBurst(t *testing.T) {
	store := ratestore.New()
	defer store.Close()

	key := "test:burst"
	remaining, ok, err := store.CheckAndConsume(context.Background(), key, 5, 5, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("first request should be allowed")
	}
	if remaining != 4 {
		t.Fatalf("expected 4 remaining, got %d", remaining)
	}

	for i := 0; i < 4; i++ {
		_, ok, err := store.CheckAndConsume(context.Background(), key, 5, 5, time.Minute)
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("request %d should be allowed (burst)", i)
		}
	}

	_, ok, err = store.CheckAndConsume(context.Background(), key, 5, 5, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("6th request should exceed burst capacity")
	}
}

func TestTokenBucketRefills(t *testing.T) {
	restore := ratestore.SetNow(func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	defer restore()

	store := ratestore.New()
	defer store.Close()

	key := "test:refill"

	remaining, ok, err := store.CheckAndConsume(context.Background(), key, 2, 2, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || remaining != 1 {
		t.Fatalf("expected 1 remaining, got %d", remaining)
	}

	remaining, ok, err = store.CheckAndConsume(context.Background(), key, 2, 2, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || remaining != 0 {
		t.Fatalf("expected 0 remaining, got %d", remaining)
	}

	_, ok, err = store.CheckAndConsume(context.Background(), key, 2, 2, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("should be rate-limited after consuming all tokens")
	}

	restore2 := ratestore.SetNow(func() time.Time {
		return time.Date(2025, 1, 1, 0, 1, 0, 0, time.UTC)
	})
	defer restore2()

	remaining, ok, err = store.CheckAndConsume(context.Background(), key, 2, 2, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("should be allowed after refill period")
	}
	if remaining != 1 {
		t.Fatalf("expected 1 remaining after refill, got %d", remaining)
	}
}
