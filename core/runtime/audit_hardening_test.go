package runtime

import (
	"context"
	"log/slog"
	"testing"

	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/audit"
)

// TestDeniedRequestIsNotAudited pins the known audit gap: because the default
// chain places "secure" OUTSIDE "audit", an authorization denial returns at
// the secure link and never reaches the audit link. Denied requests therefore
// produce NO audit record. This is intentional today (denials are cheap, the
// audit link is the innermost), but it is a compliance gap that a regression
// test must keep visible.
func TestDeniedRequestIsNotAudited(t *testing.T) {
	persister := &mockPersister{}

	base := NewLocalDispatcher()
	if err := base.RegisterHandler("admin:only", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "admin:only", Enabled: true}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	auditDisp := NewAuditDispatcher(base, persister)

	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("admin:only", "administrator", "")
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"administrator": compileRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})

	// Mirror the real arrangement: secure is OUTER, audit is INNER.
	secure := NewSecureDispatcher(auditDisp, permMgr, ac)

	_, err := secure.Send(testMessage{ctx: anonymousContext(), name: "admin:only"})
	if err == nil {
		t.Fatal("expected authz denial for anonymous on admin-scoped message")
	}
	auditDisp.Sync()
	if len(persister.entries) != 0 {
		t.Fatalf("denied request must not be audited (secure is outer to audit); got %d entries", len(persister.entries))
	}

	// Control: an allowed request DOES reach the audit link.
	_, err = secure.Send(testMessage{ctx: adminContext(), name: "admin:only"})
	if err != nil {
		t.Fatalf("admin should be allowed: %v", err)
	}
	auditDisp.Sync()
	if len(persister.entries) != 1 {
		t.Fatalf("expected 1 audit entry for allowed request, got %d", len(persister.entries))
	}
	if persister.entries[0].Status != audit.AuditStatusSuccess {
		t.Fatalf("expected success status, got %q", persister.entries[0].Status)
	}
}

// blockingPersister blocks every Insert so the flusher cannot drain the ring.
type blockingPersister struct {
	release chan struct{}
}

func (p *blockingPersister) Insert(_ context.Context, _ audit.AuditEntry) error {
	<-p.release
	return nil
}

// TestAuditBufferFailOpenDropsWhenBroken verifies the circuit breaker: once the
// buffer is in fail-open mode, Write returns nil (best-effort) and drops the
// entry instead of blocking or erroring the request path.
func TestAuditBufferFailOpenDropsWhenBroken(t *testing.T) {
	persister := &blockingPersister{release: make(chan struct{})}
	defer close(persister.release)

	buf := NewAuditBufferSize(persister, zap.NewNop(), 4)
	t.Cleanup(buf.Close)

	// Simulate the breaker having opened (buffer full / persister failing).
	buf.mu.Lock()
	buf.failed = true
	buf.mu.Unlock()

	if err := buf.Write(context.Background(), audit.AuditEntry{EventName: "e", EventID: "1"}); err != nil {
		t.Fatalf("write in fail-open mode must not error, got: %v", err)
	}
	if len(buf.entries) != 0 {
		t.Fatalf("fail-open mode must drop the entry; buffer holds %d", len(buf.entries))
	}
}

// TestAuditBufferDrainsOnClose verifies the audit buffer flushes queued entries
// on shutdown so Close does not lose the tail of the audit log.
func TestAuditBufferDrainsOnClose(t *testing.T) {
	persister := &mockPersister{}
	buf := NewAuditBufferSize(persister, zap.NewNop(), 16)

	for i := 0; i < 5; i++ {
		if err := buf.Write(context.Background(), audit.AuditEntry{EventName: "e", EventID: string(rune('a' + i))}); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	buf.Close()
	if len(persister.entries) != 5 {
		t.Fatalf("expected 5 entries drained on close, got %d", len(persister.entries))
	}
}

// TestAuditBufferSyncedFlush ensures Sync blocks until queued entries are
// persisted — the guarantee tests rely on to make audit assertions.
func TestAuditBufferSyncedFlush(t *testing.T) {
	persister := &mockPersister{}
	buf := NewAuditBufferSize(persister, zap.NewNop(), 16)
	t.Cleanup(buf.Close)

	for i := 0; i < 3; i++ {
		if err := buf.Write(context.Background(), audit.AuditEntry{EventName: "e", EventID: string(rune('a' + i))}); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	buf.Sync()
	if len(persister.entries) != 3 {
		t.Fatalf("expected 3 entries after Sync, got %d", len(persister.entries))
	}
}
