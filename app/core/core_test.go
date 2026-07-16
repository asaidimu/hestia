package core

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core/registration"
)

func TestMapPermissionManager(t *testing.T) {
	pm := NewMapPermissionManager()
	pm.RegisterScope("test:cmd", "administrator", "test command")

	msg := testMessage{name: "test:cmd", ctx: context.Background()}
	scope, err := pm.Resolve(msg)
	if err != nil {
		t.Fatalf("Resolve failed for registered scope: %v", err)
	}
	if scope != "administrator" {
		t.Fatalf("expected scope 'administrator', got %q", scope)
	}

	_, err = pm.Resolve(testMessage{name: "unknown:cmd", ctx: context.Background()})
	if err == nil {
		t.Fatal("expected error for unregistered scope, got nil")
	}

	caps := pm.ListCapabilities()
	if len(caps) != 1 {
		t.Fatalf("expected 1 capability, got %d", len(caps))
	}
	if caps[0].Name != "test:cmd" {
		t.Fatalf("expected name 'test:cmd', got %q", caps[0].Name)
	}
	if caps[0].Scope != "administrator" {
		t.Fatalf("expected scope 'administrator', got %q", caps[0].Scope)
	}
}

func TestLocalDispatcher(t *testing.T) {
	d := NewLocalDispatcher()

	var called bool
	var gotName string
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		called = true
		gotName = msg.Name()
		return &registration.Result{}, nil
	}

	err := d.RegisterHandler("test:cmd", handler, HandlerInfo{
		Name: "test:cmd", Description: "test", Enabled: true,
	})
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	msg := testMessage{name: "test:cmd", ctx: context.Background()}
	_, err = d.Send(msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
	if gotName != "test:cmd" {
		t.Fatalf("expected name 'test:cmd', got %q", gotName)
	}

	_, err = d.Send(testMessage{name: "unknown:cmd", ctx: context.Background()})
	if err == nil {
		t.Fatal("expected error for unknown handler, got nil")
	}

	err = d.RegisterHandler("test:cmd", handler, HandlerInfo{Name: "test:cmd"})
	if err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestLocalDispatcherListHandlers(t *testing.T) {
	d := NewLocalDispatcher()
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		return &registration.Result{}, nil
	}

	d.RegisterHandler("cmd:a", handler, HandlerInfo{Name: "cmd:a", Enabled: true})
	d.RegisterHandler("cmd:b", handler, HandlerInfo{Name: "cmd:b", Enabled: false})
	d.RegisterHandler("cmd:c", handler, HandlerInfo{Name: "cmd:c", Enabled: true})

	list := d.ListHandlers()
	if len(list) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(list))
	}

	var enabledCount int
	for _, h := range list {
		if h.Enabled {
			enabledCount++
		}
	}
	if enabledCount != 2 {
		t.Fatalf("expected 2 enabled, got %d", enabledCount)
	}
}

func TestLocalDispatcherSetEnabled(t *testing.T) {
	d := NewLocalDispatcher()
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		return &registration.Result{}, nil
	}

	d.RegisterHandler("test:cmd", handler, HandlerInfo{Name: "test:cmd", Enabled: true})

	msg := testMessage{name: "test:cmd", ctx: context.Background()}

	_, err := d.Send(msg)
	if err != nil {
		t.Fatalf("Send should succeed when enabled: %v", err)
	}

	if err := d.SetHandlerEnabled("test:cmd", false); err != nil {
		t.Fatalf("SetHandlerEnabled(false) failed: %v", err)
	}

	_, err = d.Send(msg)
	if err == nil {
		t.Fatal("expected error when handler disabled, got nil")
	}

	if err := d.SetHandlerEnabled("test:cmd", true); err != nil {
		t.Fatalf("SetHandlerEnabled(true) failed: %v", err)
	}

	_, err = d.Send(msg)
	if err != nil {
		t.Fatalf("Send should succeed when re-enabled: %v", err)
	}

	err = d.SetHandlerEnabled("nonexistent", true)
	if err == nil {
		t.Fatal("expected error for unknown handler, got nil")
	}
}

func TestLocalDispatcherDelete(t *testing.T) {
	d := NewLocalDispatcher()
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		return &registration.Result{}, nil
	}

	d.RegisterHandler("test:cmd", handler, HandlerInfo{Name: "test:cmd", Enabled: true})

	_, err := d.GetHandler("test:cmd")
	if err != nil {
		t.Fatalf("GetHandler before delete failed: %v", err)
	}

	if err := d.DeleteHandler("test:cmd"); err != nil {
		t.Fatalf("DeleteHandler failed: %v", err)
	}

	_, err = d.GetHandler("test:cmd")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}

	if err := d.DeleteHandler("nonexistent"); err != nil {
		t.Fatalf("DeleteHandler on nonexistent should not error: %v", err)
	}
}

func TestErrorSentinels(t *testing.T) {
	sentinels := []error{
		ErrAccessDenied,
		ErrNotFound,
		ErrAlreadyExists,
		ErrValidation,
		ErrInvalidRequest,
		ErrUnauthorized,
		ErrInvalidCredentials,
		ErrInternal,
		ErrNotImplemented,
		ErrServiceUnavailable,
		ErrSchemaRequired,
		ErrSchemaMissingName,
		ErrCollectionExists,
		ErrReservedName,
		ErrDocumentRequired,
		ErrParseDocument,
		ErrDocumentNotFound,
		ErrAuthRequired,
		ErrMissingParam,
		ErrInvalidQDSL,
		ErrEmailExists,
		ErrUserDeleted,
		ErrForbidden,
		ErrPermissionNotRegistered,
		ErrInvalidToken,
	}
	for _, e := range sentinels {
		if e == nil {
			t.Error("sentinel error is nil")
		}
	}
}

// ---- added tests ----

type mockPersister struct {
	entries []AuditEntry
}

func (m *mockPersister) Insert(_ context.Context, entry AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func TestAuditDispatcher(t *testing.T) {
	persister := &mockPersister{}
	next := NewLocalDispatcher()
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		return &registration.Result{}, nil
	}
	if err := next.RegisterHandler("test:cmd", handler, HandlerInfo{Name: "test:cmd", Enabled: true}); err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	disp := NewAuditDispatcher(next, persister)

	msg := testMessage{name: "test:cmd", ctx: context.Background()}
	_, err := disp.Send(msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if len(persister.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(persister.entries))
	}
	if persister.entries[0].EventName != "test:cmd" {
		t.Fatalf("expected EventName test:cmd, got %q", persister.entries[0].EventName)
	}
	if persister.entries[0].Status != AuditStatusSuccess {
		t.Fatalf("expected Status success, got %q", persister.entries[0].Status)
	}
	if persister.entries[0].LatencyMs < 0 {
		t.Fatal("expected non-negative LatencyMs")
	}
	if persister.entries[0].ServiceName != "hestia" {
		t.Fatalf("expected ServiceName hestia, got %q", persister.entries[0].ServiceName)
	}
}

func TestAuditDispatcherError(t *testing.T) {
	persister := &mockPersister{}
	next := NewLocalDispatcher()
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		return nil, ErrValidation
	}
	if err := next.RegisterHandler("test:fail", handler, HandlerInfo{Name: "test:fail", Enabled: true}); err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	disp := NewAuditDispatcher(next, persister)

	ctx := ContextWithAuditIdentity(context.Background(), "u1", ActorTypeUser, AuthMethodPassword)
	ctx = ContextWithAuditTransport(ctx, "10.0.0.1", "curl", "req-1")
	msg := testMessage{name: "test:fail", ctx: ctx}
	disp.Send(msg)

	if len(persister.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(persister.entries))
	}
	if persister.entries[0].Status != AuditStatusError {
		t.Fatalf("expected Status Error, got %q", persister.entries[0].Status)
	}
	if persister.entries[0].ActorID != "u1" {
		t.Fatalf("expected ActorID u1, got %q", persister.entries[0].ActorID)
	}
	if persister.entries[0].AuthMethod != AuthMethodPassword {
		t.Fatalf("expected AuthMethod password, got %q", persister.entries[0].AuthMethod)
	}
	if persister.entries[0].RequestID != "req-1" {
		t.Fatalf("expected RequestID req-1, got %q", persister.entries[0].RequestID)
	}
	if persister.entries[0].SourceIP != "10.0.0.1" {
		t.Fatalf("expected SourceIP 10.0.0.1, got %q", persister.entries[0].SourceIP)
	}
}

func TestNamespacedDispatcher(t *testing.T) {
	next := NewLocalDispatcher()
	handler := func(ctx context.Context, msg Message) (*registration.Result, error) {
		return &registration.Result{}, nil
	}
	next.RegisterHandler("blobs:put", handler, HandlerInfo{Name: "blobs:put", Enabled: true})
	next.RegisterHandler("other:cmd", handler, HandlerInfo{Name: "other:cmd", Enabled: true})

	var hydrated bool
	hydrator := func(msg Message) (Message, error) {
		hydrated = true
		return msg, nil
	}
	disp := NewNamespacedDispatcher("blobs:", next, hydrator)

	hydrated = false
	_, err := disp.Send(testMessage{name: "blobs:put", ctx: context.Background()})
	if err != nil {
		t.Fatalf("Send failed for prefixed message: %v", err)
	}
	if !hydrated {
		t.Fatal("expected hydrator to be called for prefixed message")
	}

	hydrated = false
	_, err = disp.Send(testMessage{name: "other:cmd", ctx: context.Background()})
	if err != nil {
		t.Fatalf("Send failed for non-prefixed message: %v", err)
	}
	if hydrated {
		t.Fatal("expected hydrator not to be called for non-prefixed message")
	}
}

func TestNamespacedDispatcherHydratorError(t *testing.T) {
	next := NewLocalDispatcher()
	disp := NewNamespacedDispatcher("err:", next, func(msg Message) (Message, error) {
		return nil, ErrValidation
	})

	_, err := disp.Send(testMessage{name: "err:test", ctx: context.Background()})
	if err == nil {
		t.Fatal("expected error from hydrator, got nil")
	}
}

func TestSecureDispatcherWithPermissionManager(t *testing.T) {
	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("admin:cmd", "administrator", "")

	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"administrator": compileRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: adminContext(), name: "admin:cmd"})
	if err != nil {
		t.Fatalf("expected no error for admin, got: %v", err)
	}

	_, err = disp.Send(testMessage{ctx: anonymousContext(), name: "admin:cmd"})
	if err == nil {
		t.Fatal("expected error for anonymous, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected access denied error, got: %v", err)
	}
}

func TestSecureDispatcherWithPermissionManagerUnregisteredScope(t *testing.T) {
	permMgr := NewMapPermissionManager()
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: adminContext(), name: "unregistered:cmd"})
	if err == nil {
		t.Fatal("expected error for unregistered scope, got nil")
	}
}

func TestContextWithAuditIdentity(t *testing.T) {
	ctx := ContextWithAuditIdentity(context.Background(), "user1", ActorTypeUser, AuthMethodPassword)

	actorID, ok := ctx.Value(AuditActorIDKey).(string)
	if !ok {
		t.Fatal("expected string actorID in context")
	}
	if actorID != "user1" {
		t.Fatalf("expected actorID user1, got %q", actorID)
	}

	actorType, ok := ctx.Value(AuditActorTypeKey).(ActorType)
	if !ok {
		t.Fatal("expected ActorType in context")
	}
	if actorType != ActorTypeUser {
		t.Fatalf("expected ActorType user, got %q", actorType)
	}
}

func TestContextWithAuditIdentityEmpty(t *testing.T) {
	ctx := ContextWithAuditIdentity(context.Background(), "", ActorTypeUser, AuthMethodPassword)

	actorID, _ := ctx.Value(AuditActorIDKey).(string)
	if actorID != "" {
		t.Fatalf("expected empty actorID, got %q", actorID)
	}
}
