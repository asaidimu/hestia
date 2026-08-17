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

func benchSecureDispatcher() *SecureDispatcher {
	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("bench:svc:op:run", "administrator", "")
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"administrator": compileRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})
	return NewSecureDispatcher(noopDispatcher{}, permMgr, ac)
}

func BenchmarkLocalDispatcherSend(b *testing.B) {
	d := NewLocalDispatcher()
	if err := d.RegisterHandler("bench:svc:op:run", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "bench:svc:op:run", Enabled: true}); err != nil {
		b.Fatalf("RegisterHandler: %v", err)
	}

	msg := testMessage{name: "bench:svc:op:run", ctx: adminContext()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.Send(msg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSecureDispatcher_Allowed(b *testing.B) {
	d := benchSecureDispatcher()
	msg := testMessage{name: "bench:svc:op:run", ctx: adminContext()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.Send(msg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSecureDispatcher_Denied(b *testing.B) {
	d := benchSecureDispatcher()
	msg := testMessage{name: "bench:svc:op:run", ctx: anonymousContext()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.Send(msg); err == nil {
			b.Fatal("expected denial")
		}
	}
}

func BenchmarkSecureDispatcher_SystemBypass(b *testing.B) {
	d := benchSecureDispatcher()
	msg := testMessage{name: "bench:svc:op:run", ctx: systemContext()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.Send(msg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullChain_AuthzAllowed(b *testing.B) {
	// Mirrors the production order: bootstrap -> secure -> ... -> audit.
	base := NewLocalDispatcher()
	if err := base.RegisterHandler("bench:svc:op:run", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "bench:svc:op:run", Enabled: true}); err != nil {
		b.Fatalf("RegisterHandler: %v", err)
	}

	auditDisp := NewAuditDispatcher(base, &mockPersister{})
	defer auditDisp.Close()
	secure := benchSecureDispatcher()
	bootstrap := NewBootstrapDispatcher(noopDispatcher{}, staticBootstrapRegistry{safe: map[string]bool{}}, func() bool { return true })

	disp := NewDispatcherChain(
		LinkEntry{Name: "bootstrap", Link: bootstrap},
		LinkEntry{Name: "secure", Link: secure},
		LinkEntry{Name: "audit", Link: auditDisp},
	).Build(base)

	msg := testMessage{name: "bench:svc:op:run", ctx: adminContext()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := disp.Send(msg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBootstrapDispatcher_Gated(b *testing.B) {
	disp := NewBootstrapDispatcher(noopDispatcher{}, staticBootstrapRegistry{safe: map[string]bool{}}, func() bool { return false })
	msg := testMessage{name: "bench:svc:op:run", ctx: systemContext()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := disp.Send(msg); err == nil {
			b.Fatal("expected pre-bootstrap gate")
		}
	}
}

// BenchmarkAuditBuffer_Write measures enqueue cost on the audit hot path.
// The buffer is drained in fixed batches so the measurement stays in normal
// (non-fail-open) mode: production audits must never hit the drop path.
func BenchmarkAuditBuffer_Write(b *testing.B) {
	const batch = 512
	buf := NewAuditBufferSize(&mockPersister{}, zap.NewNop(), batch*2)
	defer buf.Close()
	entry := audit.AuditEntry{EventName: "bench:svc:op:run", EventID: "bench"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := buf.Write(ctx, entry); err != nil {
			b.Fatal(err)
		}
		if i%batch == batch-1 {
			buf.Sync()
		}
	}
}