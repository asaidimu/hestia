package integration_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
)

type testMessage struct {
	name string
	ctx  context.Context
}

func (m testMessage) ID() string                              { return "" }
func (m testMessage) Name() string                            { return m.name }
func (m testMessage) Context() context.Context                 { return m.ctx }
func (m testMessage) Input() *data.Document                    { return nil }
func (m testMessage) InputChannel() <-chan *data.Document      { return nil }
func (m testMessage) BlobInputChannel() <-chan registration.Blob { return nil }

type mockPersister struct {
	entries []runtime.AuditEntry
}

func (m *mockPersister) Insert(_ context.Context, entry runtime.AuditEntry) error {
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

func TestPanickingHandlerIsRecovered(t *testing.T) {
	t.Parallel()

	local := runtime.NewLocalDispatcher()
	handler := func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		panic("handler panic: something went wrong")
	}
	if err := local.RegisterHandler("test:panic", handler, runtime.HandlerInfo{
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
	recovery := runtime.NewRecoveryDispatcher(secure, zap.NewNop())

	persister := &mockPersister{}
	audit := runtime.NewAuditDispatcher(recovery, persister)

	msg := testMessage{name: "test:panic", ctx: adminContext()}
	result, err := audit.Send(msg)

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
	if entry.Status != runtime.AuditStatusError {
		t.Fatalf("expected Status 'error', got %q", entry.Status)
	}
}
