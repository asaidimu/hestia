package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/abstract"
)

type stubDispatcher struct {
	sendErr error
	cbErr   error
	cbRes   *abstract.Result
}

func (d *stubDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if d.sendErr != nil {
		return d.sendErr
	}
	abstract.Complete(onComplete, ctx, d.cbRes, d.cbErr)
	return nil
}

type captureMessage struct{ abstract.Message }

func TestRestartLinkHookFiresOnCompletionError(t *testing.T) {
	sentinel := fmt.Errorf("wrapped: %w", ErrRestartRequired)
	var mu sync.Mutex
	order := []string{}
	hookErr := make(chan error, 1)
	next := &stubDispatcher{cbErr: sentinel}
	link := NewRestartLink(func(err error) {
		mu.Lock()
		order = append(order, "hook")
		mu.Unlock()
		hookErr <- err
	}).Wrap(next)

	cbFired := make(chan struct{}, 1)
	err := link.Send(context.Background(), captureMessage{}, func(ctx context.Context, res *abstract.Result, err error) {
		mu.Lock()
		order = append(order, "onComplete")
		mu.Unlock()
		cbFired <- struct{}{}
	})
	if err != nil {
		t.Fatalf("Send returned error on completion path: %v", err)
	}
	select {
	case <-cbFired:
	case <-time.After(time.Second):
		t.Fatal("onComplete never fired")
	}
	select {
	case got := <-hookErr:
		if !errors.Is(got, ErrRestartRequired) {
			t.Fatalf("hook got %v, want ErrRestartRequired", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("restart hook never fired")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "onComplete" || order[1] != "hook" {
		t.Fatalf("ordering = %v, want [onComplete hook] (response before hook)", order)
	}
}

func TestRestartLinkHookFiresOnSyncRejection(t *testing.T) {
	sentinel := fmt.Errorf("bootstrap: %w", ErrRestartRequired)
	hookCh := make(chan error, 1)
	next := &stubDispatcher{sendErr: sentinel}
	link := NewRestartLink(func(err error) { hookCh <- err }).Wrap(next)

	cbFired := false
	err := link.Send(context.Background(), captureMessage{}, func(ctx context.Context, res *abstract.Result, err error) {
		cbFired = true
	})
	if !errors.Is(err, ErrRestartRequired) {
		t.Fatalf("Send = %v, want wrapped ErrRestartRequired", err)
	}
	if cbFired {
		t.Fatal("onComplete must not fire on synchronous rejection (Send contract)")
	}
	select {
	case got := <-hookCh:
		if !errors.Is(got, ErrRestartRequired) {
			t.Fatalf("hook got %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("restart hook never fired on sync rejection")
	}
}

func TestRestartLinkIgnoresOtherErrors(t *testing.T) {
	hookCh := make(chan error, 1)
	next := &stubDispatcher{cbErr: errors.New("boom")}
	link := NewRestartLink(func(err error) { hookCh <- err }).Wrap(next)
	_ = link.Send(context.Background(), captureMessage{}, func(ctx context.Context, res *abstract.Result, err error) {})
	select {
	case got := <-hookCh:
		t.Fatalf("hook fired for non-restart error: %v", got)
	case <-time.After(500 * time.Millisecond):
	}
}

func TestRestartLinkNilHookPassesThrough(t *testing.T) {
	sentinel := fmt.Errorf("wrapped: %w", ErrRestartRequired)
	next := &stubDispatcher{sendErr: sentinel}
	link := NewRestartLink(nil).Wrap(next)
	fired := false
	err := link.Send(context.Background(), captureMessage{}, func(ctx context.Context, res *abstract.Result, err error) {
		fired = true
	})
	if !errors.Is(err, ErrRestartRequired) {
		t.Fatalf("nil hook must pass through unchanged, got %v", err)
	}
	if fired {
		t.Fatal("onComplete must not fire on synchronous rejection")
	}
}
