// Package streamsink provides a race-free sink for streaming results.
//
// Stream services hand a Sink's channel to the transport and push documents
// from persistence/event-bus callbacks. Those callbacks can still be running
// when a stream is torn down (go-events cancels its drain context but does not
// wait for an in-flight handler), so a plain close(ch) races with a send on the
// closed channel and panics the process. Sink serializes Send/TrySend against
// Close: once Close returns, no send can reach the channel.
package streamsink

import (
	"context"
	"sync"
	"sync/atomic"
)

// Sink is a bounded channel with idempotent, race-free Close.
//
// The zero value is not usable; construct one with New.
type Sink[T any] struct {
	mu      sync.RWMutex
	ch      chan T
	closing chan struct{}
	closed  bool
	once    sync.Once
	dropped atomic.Uint64
}

// New creates a Sink with the given buffer size.
func New[T any](buffer int) *Sink[T] {
	if buffer < 0 {
		buffer = 0
	}
	return &Sink[T]{
		ch:      make(chan T, buffer),
		closing: make(chan struct{}),
	}
}

// Channel returns the receive side handed to the transport. It is closed by
// Close after every in-flight send has completed.
func (s *Sink[T]) Channel() <-chan T { return s.ch }

// TrySend never blocks. It returns false when the buffer is full or the sink
// is closed, and counts the drop.
func (s *Sink[T]) TrySend(v T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return false
	}
	select {
	case s.ch <- v:
		return true
	default:
		s.dropped.Add(1)
		return false
	}
}

// Send blocks until the value is delivered, ctx is done, or the sink closes.
// Use it where dropping would lose data (for example workflow timelines).
func (s *Sink[T]) Send(ctx context.Context, v T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return false
	}
	select {
	case s.ch <- v:
		return true
	case <-ctx.Done():
		return false
	case <-s.closing:
		return false
	}
}

// Close is idempotent. It wakes blocked senders before taking the write lock
// (a blocked Send holds the read lock, so closing `closing` first prevents a
// deadlock), then closes the channel once no sender can touch it.
func (s *Sink[T]) Close() {
	s.once.Do(func() { close(s.closing) })
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.ch)
}

// Dropped reports how many TrySend calls were rejected because the buffer was
// full or the sink was already closed.
func (s *Sink[T]) Dropped() uint64 { return s.dropped.Load() }
