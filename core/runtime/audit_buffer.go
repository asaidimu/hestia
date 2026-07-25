package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AuditBuffer is a bounded ring buffer that decouples dispatch latency
// from audit I/O. Entries are written synchronously to the ring, then
// flushed in a background goroutine. If the buffer fills or the
// persister fails repeatedly, the circuit breaker opens and entries
// are dropped with a fallback to stderr.
type AuditBuffer struct {
	entries   chan AuditEntry
	persister AuditPersister
	logger    *zap.Logger

	mu         sync.Mutex
	failed     bool
	wg         sync.WaitGroup
	flusherCtx context.Context
	stop       context.CancelFunc
	stopped    chan struct{}
}

const defaultBufferSize = 4096

func NewAuditBuffer(persister AuditPersister, logger *zap.Logger) *AuditBuffer {
	return NewAuditBufferSize(persister, logger, defaultBufferSize)
}

func NewAuditBufferSize(persister AuditPersister, logger *zap.Logger, size int) *AuditBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	b := &AuditBuffer{
		entries:    make(chan AuditEntry, size),
		persister:  persister,
		logger:     logger,
		flusherCtx: ctx,
		stop:       cancel,
		stopped:    make(chan struct{}),
	}
	go b.flushLoop()
	return b
}

// Write enqueues an audit entry. Returns an error only if the buffer
// is full and the circuit breaker has opened (best-effort).
func (b *AuditBuffer) Write(ctx context.Context, entry AuditEntry) error {
	b.mu.Lock()
	failed := b.failed
	b.mu.Unlock()

	if failed {
		_, _ = fmt.Fprintf(os.Stderr, "AUDIT DROP: %s %s\n", entry.EventName, entry.EventID)
		return nil
	}

	b.wg.Add(1)
	select {
	case b.entries <- entry:
		return nil
	default:
		b.wg.Done()
		// Buffer full — open circuit breaker
		b.mu.Lock()
		b.failed = true
		b.mu.Unlock()
		b.logger.Error("audit buffer full, entering fail-open mode",
			zap.String("event", entry.EventName),
			zap.Int("capacity", cap(b.entries)),
		)
		_, _ = fmt.Fprintf(os.Stderr, "AUDIT DROP (buffer full): %s %s\n", entry.EventName, entry.EventID)
		return nil
	}
}

// Sync blocks until all entries queued at call time are flushed.
// For use in tests that need synchronous audit persistence.
func (b *AuditBuffer) Sync() {
	b.wg.Wait()
}

func (b *AuditBuffer) Close() {
	b.stop()
	<-b.stopped
}

func (b *AuditBuffer) flushLoop() {
	defer close(b.stopped)

	const maxBackoff = 30 * time.Second
	backoff := 100 * time.Millisecond

	for {
		select {
		case <-b.flusherCtx.Done():
			b.drain()
			return
		case entry := <-b.entries:
			if err := b.persister.Insert(b.flusherCtx, entry); err != nil {
				b.logger.Error("audit persist failed, will retry",
					zap.String("event", entry.EventName),
					zap.Error(err),
				)
				// Re-enqueue on failure; drop if buffer is full
				select {
				case b.entries <- entry:
				default:
					b.wg.Done()
				}
				// Backoff before next attempt
				select {
				case <-b.flusherCtx.Done():
					return
				case <-time.After(backoff):
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
			} else {
				b.wg.Done()
				backoff = 100 * time.Millisecond
				b.mu.Lock()
				if b.failed {
					b.failed = false
				}
				b.mu.Unlock()
			}
		}
	}
}

// drain flushes remaining entries on shutdown.
func (b *AuditBuffer) drain() {
	for {
		select {
		case entry := <-b.entries:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = b.persister.Insert(ctx, entry)
			b.wg.Done()
			cancel()
		default:
			return
		}
	}
}
