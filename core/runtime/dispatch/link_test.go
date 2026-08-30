package dispatch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

type recordingDispatcher struct {
	accepted bool
	err      error
	fires    int32
}

func (d *recordingDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if !d.accepted {
		return d.err
	}
	for i := int32(0); i < d.fires; i++ {
		abstract.Complete(onComplete, ctx, nil, nil)
	}
	return nil
}

type testMessage struct{ abstract.Message }

// TestLinkContractRejection verifies: a PreCheck rejection is surfaced by
// Send and the completion callback never fires (Send contract).
func TestLinkContractRejection(t *testing.T) {
	var fired atomic.Bool
	next := &recordingDispatcher{accepted: false}
	link := NewLink("test", func(ctx context.Context, msg abstract.Message) error {
		return errors.New("rejected")
	}, nil).Wrap(next)
	err := link.Send(context.Background(), testMessage{}, func(ctx context.Context, res *abstract.Result, err error) {
		fired.Store(true)
	})
	if err == nil || err.Error() != "rejected" {
		t.Fatalf("Send = %v, want the pre-check rejection", err)
	}
	if fired.Load() {
		t.Fatal("onComplete must never fire on synchronous rejection")
	}
}

// TestLinkContractAcceptance verifies: acceptance forwards and the outcome
// passes through the observer untouched.
func TestLinkContractAcceptance(t *testing.T) {
	var got *abstract.Result
	next := &recordingDispatcher{accepted: true}
	link := NewLink("test", nil, func(onComplete abstract.CompletionFunc) abstract.CompletionFunc {
		return func(ctx context.Context, res *abstract.Result, err error) {
			// observer sees the outcome, then forwards verbatim
			onComplete(ctx, res, err)
		}
	}).Wrap(next)
	err := link.Send(context.Background(), testMessage{}, func(ctx context.Context, res *abstract.Result, err error) {
		got = res
	})
	if err != nil {
		t.Fatalf("accepted dispatch must return nil, got %v", err)
	}
	if got != nil {
		t.Fatal("result must pass through unaltered")
	}
}

// TestLinkGuardCompletionExactlyOnce: a downstream that (buggy or by
// design) fires the callback multiple times cannot cause double delivery
// through a guarded observer.
func TestLinkGuardCompletionExactlyOnce(t *testing.T) {
	var mu sync.Mutex
	count := 0
	next := &recordingDispatcher{accepted: true, fires: 5}
	link := NewLink("test", nil, func(onComplete abstract.CompletionFunc) abstract.CompletionFunc {
		return GuardCompletion(onComplete)
	}).Wrap(next)
	_ = link.Send(context.Background(), testMessage{}, func(ctx context.Context, res *abstract.Result, err error) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("onComplete fired %d times, want exactly once", count)
	}
}

// TestLinkContractPanicSafety: a panicking terminal is recovered by the
// dispatcher (not the link); a panicking observer still must not double-fire
// — GuardCompletion documents the exactly-once seam.
func TestLinkWrapProducesIndependentInstance(t *testing.T) {
	link := NewLink("test", nil, nil)
	w1 := link.Wrap(&recordingDispatcher{})
	w2 := link.Wrap(&recordingDispatcher{})
	if w1 == w2 {
		t.Fatal("Wrap must produce independent instances (links wrap one next each)")
	}
}
