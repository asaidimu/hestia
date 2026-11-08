package streamsink

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTrySendAndClose(t *testing.T) {
	s := New[int](1)
	if !s.TrySend(1) {
		t.Fatal("first send should succeed")
	}
	if s.TrySend(2) {
		t.Fatal("send into a full buffer should fail")
	}
	if s.Dropped() != 1 {
		t.Fatalf("Dropped = %d, want 1", s.Dropped())
	}

	s.Close()
	s.Close() // idempotent

	if s.TrySend(3) {
		t.Fatal("send after Close should fail")
	}
	if v, ok := <-s.Channel(); !ok || v != 1 {
		t.Fatalf("buffered value = %d/%v, want 1/true", v, ok)
	}
	if _, ok := <-s.Channel(); ok {
		t.Fatal("channel should be closed")
	}
}

// TestCloseRacesWithTrySend is the regression test for the panic that plain
// close(ch) caused: a persistence callback could still be sending when the
// stream tore down. Run with -race; it must never panic.
func TestCloseRacesWithTrySend(t *testing.T) {
	for i := 0; i < 2000; i++ {
		s := New[int](4)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 64; j++ {
				s.TrySend(j)
			}
		}()
		s.Close()
		wg.Wait()
	}
}

// TestCloseRacesWithBlockingSend proves Close unblocks a Send waiting on a
// full buffer instead of deadlocking behind its read lock.
func TestCloseRacesWithBlockingSend(t *testing.T) {
	s := New[int](0) // unbuffered: Send blocks until a receiver or Close
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Send(context.Background(), 1)
	}()

	time.Sleep(10 * time.Millisecond)
	s.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Send did not return after Close")
	}
}
