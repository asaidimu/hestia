package core

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core/registration"
)

type testMessage struct {
	name string
	ctx  context.Context
}

func (m testMessage) ID() string                         { return "" }
func (m testMessage) Name() string                       { return m.name }
func (m testMessage) Context() context.Context            { return m.ctx }
func (m testMessage) Input() *data.Document               { return data.MustNewDocument(nil, m.ctx) }
func (m testMessage) InputChannel() <-chan *data.Document   { return nil }
func (m testMessage) BlobInputChannel() <-chan registration.Blob { return nil }

func anonymousContext() context.Context {
	props := map[string]any{
		"user_id":    "",
		"email":      "",
		"scopes":     []string{},
		"token_type": "",
	}
	return iam.WithIdentity(context.Background(), iam.Identity{
		Permissions: []string{},
		Properties:  props,
	})
}

func adminContext() context.Context {
	props := map[string]any{
		"user_id":    "u1",
		"email":      "admin@test.local",
		"scopes":     []string{"administrator"},
		"token_type": "access",
	}
	return iam.WithIdentity(context.Background(), iam.Identity{
		Permissions: []string{"administrator"},
		Properties:  props,
	})
}

func systemContext() context.Context {
	props := map[string]any{
		"scopes": []string{"system:test"},
	}
	return iam.WithIdentity(context.Background(), iam.Identity{
		Permissions: []string{"system:test"},
		Properties:  props,
	})
}

func compileRule(ac iam.AccessController, expr string) iam.FunctionRule {
	fn, err := ac.CompileCELRule(expr)
	if err != nil {
		panic(err)
	}
	return fn
}

func TestSecureDispatcher_AnonymousDeniedForAdminScope(t *testing.T) {
	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("collections:_user:read", "administrator", "")

	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"public":        compileRule(ac, "true"),
		"authenticated": compileRule(ac, "identity != null"),
		"administrator": compileRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: anonymousContext(), name: "collections:_user:read"})
	if err == nil {
		t.Fatal("expected ErrAccessDenied for anonymous user on admin-scoped query, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected access denied error, got: %v", err)
	}
}

func TestSecureDispatcher_AdminAllowedForAdminScope(t *testing.T) {
	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("collections:_user:read", "administrator", "")

	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"public":        compileRule(ac, "true"),
		"authenticated": compileRule(ac, "identity != null"),
		"administrator": compileRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: adminContext(), name: "collections:_user:read"})
	if err != nil {
		t.Fatalf("expected no error for admin user on admin-scoped query, got: %v", err)
	}
}

func TestSecureDispatcher_AnonymousAllowedForPublicScope(t *testing.T) {
	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("auth:session:create", "public", "")

	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"public": compileRule(ac, "true"),
	})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: anonymousContext(), name: "auth:session:create"})
	if err != nil {
		t.Fatalf("expected no error for anonymous user on public-scoped query, got: %v", err)
	}
}

func TestSecureDispatcher_SystemIdentityBypassesCheck(t *testing.T) {
	permMgr := NewMapPermissionManager()
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: systemContext(), name: "anything"})
	if err != nil {
		t.Fatalf("expected no error for system identity, got: %v", err)
	}
}

// --- helpers ---

type discarder struct{}

func (d discarder) Write(p []byte) (int, error) { return len(p), nil }

type noopDispatcher struct{}

func (d noopDispatcher) Send(Message) (*registration.Result, error) { return &registration.Result{}, nil }

var _ Dispatcher = noopDispatcher{}
