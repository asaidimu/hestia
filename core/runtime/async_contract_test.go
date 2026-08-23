package runtime

import (
	"context"

	"errors"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/abstract"
)

// Contract tests for the async-native Dispatcher semantics:
//
//   - Send either returns a non-nil error (synchronous rejection, callback
//     never fires, no goroutine started) or accepts the message and invokes
//     onComplete exactly once.
//   - Panics in handlers are contained and delivered as *PanicError.
//   - Callbacks fire even when the caller has long moved on (days-later
//     completion); no goroutine stays parked per in-flight message.

type chanMessage struct {
	name string
	ctx  context.Context
}

func (m *chanMessage) ID() string                             { return "" }
func (m *chanMessage) Name() string                           { return m.name }
func (m *chanMessage) Context() context.Context               { return m.ctx }
func (m *chanMessage) Input() data.Documenter                 { return nil }
func (m *chanMessage) InputChannel() <-chan data.Documenter   { return nil }
func (m *chanMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m *chanMessage) TenantID() string                       { return "" }
func (m *chanMessage) TraceID() string                        { return "" }
func (m *chanMessage) RequestID() string                      { return "" }
func (m *chanMessage) SourceIP() string                       { return "" }
func (m *chanMessage) UserAgent() string                      { return "" }
func (m *chanMessage) ResourceID() string                     { return "" }
func (m *chanMessage) SessionID() string                      { return "" }

type awaiter struct {
	t  *testing.T
	d  abstract.Dispatcher
	wg sync.WaitGroup
}

func newAwaiter(t *testing.T, d abstract.Dispatcher) *awaiter {
	a := &awaiter{t: t, d: d}
	a.wg.Add(1)
	return a
}

func TestSendExactlyOnceOnSuccess(t *testing.T) {
	local := NewLocalDispatcher()
	var calls int32
	if err := local.RegisterHandler("contract:ok", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "contract:ok", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	err := local.Send(context.Background(), &chanMessage{name: "contract:ok", ctx: context.Background()}, func(ctx context.Context, res *abstract.Result, err error) {
		atomic.AddInt32(&calls, 1)
		close(done)
	})
	if err != nil {
		t.Fatalf("Send rejected: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("completion never fired")
	}
	if calls != 1 {
		t.Fatalf("callback fired %d times, want exactly 1", calls)
	}
}

func TestSyncRejectionNeverFiresCallback(t *testing.T) {
	local := NewLocalDispatcher()
	fired := make(chan struct{})
	err := local.Send(context.Background(), &chanMessage{name: "no-such-handler", ctx: context.Background()}, func(ctx context.Context, res *abstract.Result, err error) {
		close(fired)
	})
	if err == nil {
		t.Fatal("expected synchronous rejection error")
	}
	select {
	case <-fired:
		t.Fatal("callback must not fire on sync rejection")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPanicDeliveredAsPanicError(t *testing.T) {
	local := NewLocalDispatcher()
	if err := local.RegisterHandler("contract:panic", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		panic("boom")
	}, abstract.HandlerInfo{Name: "contract:panic", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	err := local.Send(context.Background(), &chanMessage{name: "contract:panic", ctx: context.Background()}, func(ctx context.Context, res *abstract.Result, err error) {
		done <- err
	})
	if err != nil {
		t.Fatalf("unexpected rejection: %v", err)
	}
	var got error
	select {
	case got = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("completion never fired")
	}
	var perr *PanicError
	if !errors.As(got, &perr) {
		t.Fatalf("want *PanicError, got %v", got)
	}
}

func TestCallbackSurvivesCallerMovingOn(t *testing.T) {
	// Simulates a days-long handler: the dispatcher must not need the caller
	// to stay parked — the completion fires whenever execution finishes.
	local := NewLocalDispatcher()
	release := make(chan struct{})
	started := make(chan struct{})
	if err := local.RegisterHandler("contract:slow", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		close(started)
		<-release
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "contract:slow", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	fired := make(chan struct{})
	if err := local.Send(context.Background(), &chanMessage{name: "contract:slow", ctx: context.Background()}, func(ctx context.Context, res *abstract.Result, err error) {
		close(fired)
	}); err != nil {
		t.Fatal(err)
	}
	<-started // handler running; caller would normally park or leave

	// Caller "returns" — nothing to clean up, no waiter goroutine leaked.
	time.Sleep(10 * time.Millisecond)

	close(release) // days later, work completes
	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("late completion never fired")
	}
}

func TestCancelBeforeStartDeliversCtxErr(t *testing.T) {
	local := NewLocalDispatcher()
	handlerRan := false
	if err := local.RegisterHandler("contract:cancel", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		handlerRan = true
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "contract:cancel", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	if err := local.Send(ctx, &chanMessage{name: "contract:cancel", ctx: context.Background()}, func(ctx context.Context, res *abstract.Result, err error) {
		done <- err
	}); err != nil {
		t.Fatal(err)
	}
	var got error
	select {
	case got = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("completion never fired")
	}
	if !errors.Is(got, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", got)
	}
	if handlerRan {
		t.Fatal("handler must not run after cancellation before start")
	}
}

func TestNilCallbackIsFireAndForget(t *testing.T) {
	local := NewLocalDispatcher()
	ran := make(chan struct{})
	if err := local.RegisterHandler("contract:ff", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		close(ran)
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "contract:ff", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := local.Send(context.Background(), &chanMessage{name: "contract:ff", ctx: context.Background()}, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never ran")
	}
}
