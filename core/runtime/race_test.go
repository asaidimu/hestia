package runtime

import (
	"context"
	"sync"
	"testing"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/audit"
)

// TestAuditBuffer_ConcurrentWrites exercises the audit ring under concurrent
// writers. Run with -race. The buffer must never lose the wg accounting (a
// mismatch panics WaitGroup) and must flush every entry.
func TestAuditBuffer_ConcurrentWrites(t *testing.T) {
	persister := &mockPersister{}
	const writers = 8
	const perWriter = 50
	// Size the ring larger than the total write burst so the fail-open
	// circuit breaker is not exercised (drop behavior is covered by
	// audit_hardening_test.go).
	buf := NewAuditBufferSize(persister, zap.NewNop(), writers*perWriter)
	defer buf.Close()
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if err := buf.Write(context.Background(), audit.AuditEntry{
					EventName: "test:concurrent:op:run",
					EventID:   string(rune('a'+w)) + string(rune('0'+i%10)),
				}); err != nil {
					t.Errorf("concurrent write: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	buf.Sync()

	if len(persister.entries) != writers*perWriter {
		t.Fatalf("expected %d entries, got %d", writers*perWriter, len(persister.entries))
	}
}

// TestLocalDispatcher_ConcurrentSends exercises dispatch under concurrent
// sends. Run with -race to catch races in handler registries and contexts.
func TestLocalDispatcher_ConcurrentSends(t *testing.T) {
	d := NewLocalDispatcher()
	name := "test:concurrent:h:run"
	if err := d.RegisterHandler(name, func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: name, Enabled: true}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	const sends = 200
	var wg sync.WaitGroup
	for i := 0; i < sends; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := d.Send(testMessage{name: name, ctx: context.Background()}); err != nil {
				t.Errorf("concurrent Send: %v", err)
			}
		}()
	}
	wg.Wait()
}
