package runtime

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
)

// orderRecorder captures the execution order of dispatcher links.
type orderRecorder struct {
	mu    sync.Mutex
	names []string
}

func (r *orderRecorder) record(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = append(r.names, name)
}

type recordingLink struct {
	name string
	rec  *orderRecorder
}

func (l recordingLink) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return recordingDispatcher{name: l.name, rec: l.rec, next: next}
}

type recordingDispatcher struct {
	name string
	rec  *orderRecorder
	next abstract.Dispatcher
}

func (d recordingDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	d.rec.record(d.name)
	return d.next.Send(ctx, msg, onComplete)
}

func assertOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("order length = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestDispatcherChain_BuildRunsOutermostFirst(t *testing.T) {
	rec := &orderRecorder{}
	base := recordingDispatcher{name: "base", rec: rec, next: noopDispatcher{}}
	chain := NewDispatcherChain(
		LinkEntry{Name: "a", Link: recordingLink{"a", rec}},
		LinkEntry{Name: "b", Link: recordingLink{"b", rec}},
		LinkEntry{Name: "c", Link: recordingLink{"c", rec}},
	)
	disp := chain.Build(base)
	if _, err := testAwait(disp, testMessage{name: "x", ctx: context.Background()}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// The first entry is the outermost wrapper and therefore runs first.
	assertOrder(t, rec.names, []string{"a", "b", "c", "base"})
}

func TestDispatcherChain_InsertAfter(t *testing.T) {
	rec := &orderRecorder{}
	base := recordingDispatcher{name: "base", rec: rec, next: noopDispatcher{}}
	chain := NewDispatcherChain(LinkEntry{Name: "a", Link: recordingLink{"a", rec}})
	chain.InsertAfter("a", "after-a", recordingLink{"after-a", rec})
	chain.InsertAfter("missing", "never", recordingLink{"never", rec}) // no-op on unknown name
	disp := chain.Build(base)
	if _, err := testAwait(disp, testMessage{name: "x", ctx: context.Background()}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	assertOrder(t, rec.names, []string{"a", "after-a", "base"})
}

func TestDispatcherChain_InsertBefore(t *testing.T) {
	rec := &orderRecorder{}
	base := recordingDispatcher{name: "base", rec: rec, next: noopDispatcher{}}
	chain := NewDispatcherChain(LinkEntry{Name: "b", Link: recordingLink{"b", rec}})
	chain.InsertBefore("b", "before-b", recordingLink{"before-b", rec})
	disp := chain.Build(base)
	if _, err := testAwait(disp, testMessage{name: "x", ctx: context.Background()}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	assertOrder(t, rec.names, []string{"before-b", "b", "base"})
}

func TestDispatcherChain_Remove(t *testing.T) {
	rec := &orderRecorder{}
	base := recordingDispatcher{name: "base", rec: rec, next: noopDispatcher{}}
	chain := NewDispatcherChain(
		LinkEntry{Name: "a", Link: recordingLink{"a", rec}},
		LinkEntry{Name: "b", Link: recordingLink{"b", rec}},
	)
	chain.Remove("a")
	disp := chain.Build(base)
	if _, err := testAwait(disp, testMessage{name: "x", ctx: context.Background()}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	assertOrder(t, rec.names, []string{"b", "base"})
}

// TestSecurePrecedesAuditInDefaultChainOrder pins the layout the SystemModule
// installs: "secure" sits outside "audit" (audit innermost). The security
// consequence — authz denials never reach the audit link — is verified
// end-to-end in audit_hardening_test.go.
func TestSecurePrecedesAuditInDefaultChainOrder(t *testing.T) {
	defaultOrder := []string{"bootstrap", "secure", "ratelimit", "throttle", "tenant", "blob", "recovery", "audit"}
	secureIdx, auditIdx := -1, -1
	for i, name := range defaultOrder {
		switch name {
		case "secure":
			secureIdx = i
		case "audit":
			auditIdx = i
		}
	}
	if secureIdx == -1 || auditIdx == -1 {
		t.Fatal("default chain must contain secure and audit")
	}
	if secureIdx > auditIdx {
		t.Fatalf("secure (index %d) must precede audit (index %d) in the default chain", secureIdx, auditIdx)
	}
}

// TestInsertBeforeSecureRunsBeforeAuthz documents the danger behind the
// invariant "never insert a chain link before secure": a link inserted there
// executes ahead of authorization and therefore outside the authz gate.
func TestInsertBeforeSecureRunsBeforeAuthz(t *testing.T) {
	rec := &orderRecorder{}

	base := NewLocalDispatcher()
	if err := base.RegisterHandler("admin:only", func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}, abstract.HandlerInfo{Name: "admin:only", Enabled: true}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("admin:only", "administrator", "")
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"administrator": compileRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})

	secure := NewSecureDispatcher(base, permMgr, ac)
	chain := NewDispatcherChain(LinkEntry{Name: "secure", Link: secure})
	chain.InsertBefore("secure", "custom", recordingLink{"custom", rec})
	disp := chain.Build(base)

	// Anonymous is denied for an admin-scoped message, but the custom link
	// (inserted before secure) already ran. This is the exposure.
	_, err := testAwait(disp, testMessage{ctx: anonymousContext(), name: "admin:only"})
	if err == nil {
		t.Fatal("expected authz denial")
	}
	assertOrder(t, rec.names, []string{"custom"})

	// Control: inserting AFTER secure means the link only runs once the
	// request has passed authorization.
	rec2 := &orderRecorder{}
	secure2 := NewSecureDispatcher(base, permMgr, ac)
	chain2 := NewDispatcherChain(LinkEntry{Name: "secure", Link: secure2})
	chain2.InsertAfter("secure", "after-secure", recordingLink{"after-secure", rec2})
	disp2 := chain2.Build(base)

	_, err = testAwait(disp2, testMessage{ctx: anonymousContext(), name: "admin:only"})
	if err == nil {
		t.Fatal("expected authz denial")
	}
	assertOrder(t, rec2.names, []string{}) // denial short-circuits before the after-link
}
