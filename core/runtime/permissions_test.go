package runtime

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

type testMessage struct {
	name string
	ctx  context.Context
}

func (m testMessage) ID() string                             { return "" }
func (m testMessage) Name() string                           { return m.name }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() data.Documenter                 { return data.MustNewDocument(nil, m.ctx) }
func (m testMessage) InputChannel() <-chan abstract.StreamItem   { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m testMessage) TenantID() string                       { return runtimecontext.GetTenantID(m.ctx) }
func (m testMessage) TraceID() string                        { return runtimecontext.GetTraceID(m.ctx) }
func (m testMessage) RequestID() string                      { return runtimecontext.GetRequestID(m.ctx) }
func (m testMessage) SourceIP() string                       { return runtimecontext.GetSourceIP(m.ctx) }
func (m testMessage) UserAgent() string                      { return runtimecontext.GetUserAgent(m.ctx) }
func (m testMessage) ResourceID() string                     { return runtimecontext.GetResourceID(m.ctx) }
func (m testMessage) SessionID() string                      { return runtimecontext.GetSessionID(m.ctx) }

func anonymousContext() context.Context {
	props := map[string]any{
		"user_id":     "",
		"email":       "",
		"permissions": []string{},
		"token_type":  "",
	}
	return iam.WithIdentity(context.Background(), iam.Identity{
		Permissions: []string{},
		Properties:  props,
	})
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

func systemContext() context.Context {
	props := map[string]any{
		"permissions": []string{"system:test"},
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

	_, err := testAwait(disp, testMessage{ctx: anonymousContext(), name: "collections:_user:read"})
	if err == nil {
		t.Fatal("expected ErrAccessDenied for anonymous user on admin-scoped query, got nil")
	}
	if !strings.Contains(err.Error(), "AUTH_REQUIRED") {
		t.Fatalf("expected auth required error, got: %v", err)
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

	_, err := testAwait(disp, testMessage{ctx: adminContext(), name: "collections:_user:read"})
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

	_, err := testAwait(disp, testMessage{ctx: anonymousContext(), name: "auth:session:create"})
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

	_, err := testAwait(disp, testMessage{ctx: systemContext(), name: "anything"})
	if err != nil {
		t.Fatalf("expected no error for system identity, got: %v", err)
	}
}

// --- helpers ---

type discarder struct{}

func (d discarder) Write(p []byte) (int, error) { return len(p), nil }

type noopDispatcher struct{}

func (d noopDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	abstract.Complete(onComplete, ctx, &abstract.Result{}, nil)
	return nil
}

var _ abstract.Dispatcher = noopDispatcher{}
