package integration_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/audit"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type testMessage struct {
	name string
	ctx  context.Context
}

func (m testMessage) ID() string                             { return "" }
func (m testMessage) Name() string                           { return m.name }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() data.Documenter                 { return nil }
func (m testMessage) InputChannel() <-chan data.Documenter   { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m testMessage) TenantID() string                       { return "" }
func (m testMessage) TraceID() string                        { return "" }
func (m testMessage) RequestID() string                      { return "" }
func (m testMessage) SourceIP() string                       { return "" }
func (m testMessage) UserAgent() string                      { return "" }
func (m testMessage) ResourceID() string                     { return "" }
func (m testMessage) SessionID() string                      { return "" }

type mockPersister struct {
	entries []audit.AuditEntry
}

func (m *mockPersister) Insert(_ context.Context, entry audit.AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func adminContext() context.Context {
	props := map[string]any{
		"user_id":     "u1",
		"email":       "admin@test.local",
		"permissions": []string{"administrator"},
		"token_type":  "access",
	}
	return iam.WithIdentity(context.Background(), iam.Identity{
		Permissions: []string{"administrator"},
		Properties:  props,
	})
}

// TestPanickingHandlerIsRecovered verifies that a panicking handler is
// contained by the terminal dispatcher: the panic is delivered to the
// completion callback as *runtime.PanicError and still produces an audit
// entry. Since the async refactor, panic recovery lives in LocalDispatcher
// itself (a wrapping link's recover() can never catch a panic raised in the
// terminal's goroutine), so no separate recovery link exists in the chain.
func TestPanickingHandlerIsRecovered(t *testing.T) {
	t.Parallel()

	local := runtime.NewLocalDispatcher()
	handler := func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		panic("handler panic: something went wrong")
	}
	if err := local.RegisterHandler("test:panic", handler, abstract.HandlerInfo{
		Name: "test:panic", Description: "a handler that panics", Enabled: true,
	}); err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	permMgr := runtime.NewMapPermissionManager()
	permMgr.RegisterScope("test:panic", "administrator", "")

	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	rule, err := ac.CompileCELRule("identity != null && 'administrator' in identity.permissions")
	if err != nil {
		t.Fatalf("CompileCELRule failed: %v", err)
	}
	ac.LoadRules(iam.FunctionRuleSet{"administrator": rule})

	secure := runtime.NewSecureDispatcher(local, permMgr, ac)

	persister := &mockPersister{}
	auditDisp := runtime.NewAuditDispatcher(secure, persister)

	msg := testMessage{name: "test:panic", ctx: adminContext()}
	result, err := testAwait(auditDisp, msg)
	auditDisp.Sync()

	if err == nil {
		t.Fatal("expected error from panicking handler, got nil")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Fatalf("expected error containing 'panic', got %q", err.Error())
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}

	if len(persister.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(persister.entries))
	}
	entry := persister.entries[0]
	if entry.EventName != "test:panic" {
		t.Fatalf("expected EventName 'test:panic', got %q", entry.EventName)
	}
	if entry.Status != audit.AuditStatusError {
		t.Fatalf("expected Status 'error', got %q", entry.Status)
	}
}

// testAwait dispatches m and blocks for its outcome; the test-lifecycle
// stand-in for request/response dispatch.
func testAwait(d abstract.Dispatcher, m abstract.Message) (*abstract.Result, error) {
	return dispatch.Await(context.Background(), d, m)
}
