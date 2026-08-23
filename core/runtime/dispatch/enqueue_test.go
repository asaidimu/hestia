package dispatch

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/abstract"
)

type stubDispatcher struct {
	sendFn func(ctx context.Context, msg abstract.Message, cb abstract.CompletionFunc) error
}

func (s *stubDispatcher) Send(ctx context.Context, msg abstract.Message, cb abstract.CompletionFunc) error {
	return s.sendFn(ctx, msg, cb)
}

func TestEnqueueReturnsImmediatelyWithID(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	completed := make(chan struct{})

	d := &stubDispatcher{sendFn: func(ctx context.Context, msg abstract.Message, cb abstract.CompletionFunc) error {
		go func() {
			close(started)
			<-release
			abstract.Complete(cb, ctx, &abstract.Result{}, nil)
			close(completed)
		}()
		return nil
	}}

	in := DispatchInput{
		Name:    "payments:mpesa:callback:create",
		Context: context.Background(),
		ID:      "fixed-id",
	}

	done := make(chan string, 1)
	go func() {
		id, err := Enqueue(context.Background(), d, in)
		if err != nil {
			t.Errorf("Enqueue: %v", err)
		}
		done <- id
	}()

	select {
	case id := <-done:
		if id != "fixed-id" {
			t.Fatalf("id = %q, want fixed-id", id)
		}
	case <-time.After(time.Second):
		t.Fatal("Enqueue did not return before handler completion")
	}

	select {
	case <-completed:
		t.Fatal("handler must not have finished yet")
	default:
	}

	close(release)
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("handler never completed")
	}
}

func TestEnqueueGeneratesIDWhenAbsent(t *testing.T) {
	var gotID string
	d := &stubDispatcher{sendFn: func(ctx context.Context, msg abstract.Message, cb abstract.CompletionFunc) error {
		gotID = msg.ID()
		abstract.Complete(cb, ctx, nil, nil)
		return nil
	}}
	id, err := Enqueue(context.Background(), d, DispatchInput{Name: "x:y:z", Context: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || id != gotID {
		t.Fatalf("id = %q, message id = %q; want equal and non-empty", id, gotID)
	}
}

func TestEnqueuePassesNilCallback(t *testing.T) {
	var sawNil bool
	d := &stubDispatcher{sendFn: func(ctx context.Context, msg abstract.Message, cb abstract.CompletionFunc) error {
		sawNil = cb == nil
		return nil
	}}
	if _, err := Enqueue(context.Background(), d, DispatchInput{Name: "x:y:z", Context: context.Background()}); err != nil {
		t.Fatal(err)
	}
	if !sawNil {
		t.Fatal("Enqueue must pass a nil completion callback")
	}
}

func TestEnqueuePropagatesSyncRejection(t *testing.T) {
	wantErr := context.DeadlineExceeded
	d := &stubDispatcher{sendFn: func(ctx context.Context, msg abstract.Message, cb abstract.CompletionFunc) error {
		return wantErr
	}}
	id, err := Enqueue(context.Background(), d, DispatchInput{Name: "x:y:z", Context: context.Background()})
	if err == nil {
		t.Fatal("expected rejection error")
	}
	if id != "" {
		t.Fatalf("id = %q on rejection, want empty", id)
	}
}
