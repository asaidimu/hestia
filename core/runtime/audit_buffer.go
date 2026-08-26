// @note #arch-20260821-010 issue resolved status=open priority=P1 tags=#arch,#concurrency : Race condition in AuditBuffer circuit breaker
//
// The Write method (line 63) reads b.failed under mutex, then calls wg.Add(1)
// outside the lock. A concurrent Close() could interleave between the check
// and the wg.Add(1).
//
// Additionally, on re-enqueue failure (line 121-125), wg.Done() is called
// without guaranteeing it was incremented for this entry. This could cause
// a panic if the entry was never added to the WaitGroup.
//
// The circuit breaker is only reset on successful persist in flushLoop,
// but if the buffer is full and the circuit opens, entries are dropped
// permanently until a successful persist occurs. Consider adding a
// timeout-based circuit breaker reset or a maximum drop count before recovery.
//
// Resolution: Hold the mutex across the entire check-and-add sequence,
// or use an atomic flag. Ensure wg.Done() is only called when wg.Add(1)
// was successfully called for the same entry.
// @note #bench-20260821-002 todo status=open priority=P1 tags=#benchmark,#performance : Audit buffer needs throughput benchmarks
//
// Audit logging is critical for compliance and security monitoring.
// Current implementation lacks benchmarks for:
//
// 1. Write throughput under high load (10000+ entries/second)
// 2. Buffer full behavior and fail-open performance
// 3. Memory usage with large entry sizes
// 4. Comparison with async vs sync persistence
//
// For IoT: Audit logs must not impact device performance.
// For HFT: Audit logging must not add latency to trades.
//
// Resolution: Add benchmarks in audit_buffer_bench_test.go:
// - BenchmarkWrite_Throughput
// - BenchmarkWrite_BufferFull
// - BenchmarkFlushLoop_Latency
// - BenchmarkMemoryUsage
package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/runtime/audit"
)

// AuditBuffer is a bounded ring buffer that decouples dispatch latency
// from audit I/O. Entries are written synchronously to the ring, then
// flushed in a background goroutine. If the buffer fills or the
// persister fails repeatedly, the circuit breaker opens and entries
// are dropped with a fallback to stderr.
type AuditBuffer struct {
	entries   chan audit.AuditEntry
	persister audit.AuditPersister
	logger    *zap.Logger

	mu         sync.Mutex
	failed     bool
	wg         sync.WaitGroup
	flusherCtx context.Context
	stop       context.CancelFunc
	stopped    chan struct{}
}

const defaultBufferSize = 4096

func NewAuditBuffer(persister audit.AuditPersister, logger *zap.Logger) *AuditBuffer {
	return NewAuditBufferSize(persister, logger, defaultBufferSize)
}

func NewAuditBufferSize(persister audit.AuditPersister, logger *zap.Logger, size int) *AuditBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	b := &AuditBuffer{
		entries:    make(chan audit.AuditEntry, size),
		persister:  persister,
		logger:     logger,
		flusherCtx: ctx,
		stop:       cancel,
		stopped:    make(chan struct{}),
	}
	go b.flushLoop()
	return b
}

// @note #review-20260821-009 issue resolved status=open priority=P1 tags=#review,#concurrency : Race condition in AuditBuffer circuit breaker
// The Write method reads b.failed under mutex, but the circuit breaker is set
// to true in the default branch without holding the mutex consistently. While
// the current code does acquire the mutex before setting b.failed, the check
// and the wg.Add(1) are not atomic — a concurrent Close() could interleave.
//
// Additionally, the circuit breaker is only reset on successful persist in
// flushLoop, but if the buffer is full and the circuit opens, entries are
// dropped permanently until a successful persist occurs. Consider adding a
// timeout-based circuit breaker reset or a maximum drop count before recovery.
func (b *AuditBuffer) Write(ctx context.Context, entry audit.AuditEntry) error {
	b.mu.Lock()
	if b.failed {
		b.mu.Unlock()
		_, _ = fmt.Fprintf(os.Stderr, "AUDIT DROP: %s %s\n", entry.EventName, entry.EventID)
		return nil
	}
	b.wg.Add(1)
	b.mu.Unlock()

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
