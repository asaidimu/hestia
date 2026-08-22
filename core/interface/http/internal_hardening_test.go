package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

// fakeCredProvider issues sessions whose token string simply encodes the
// SessionInfo. It exists to exercise middleware revocation/expiry logic
// without pulling in the auth feature (which imports core/interface/http and
// would create an import cycle in tests). Session crypto itself is covered by
// the auth package tests.
type fakeCredProvider struct{}

func (fakeCredProvider) CreateSession(userID string, ttl time.Duration) (string, *abstract.SessionInfo, error) {
	now := time.Now().Unix()
	info := &abstract.SessionInfo{
		SessionID:    "sid-" + userID,
		UserID:       userID,
		IssuedAt:     now,
		ExpiresAt:    now + int64(ttl.Seconds()),
		CreatedAt:    now,
		TokenVersion: 0,
	}
	token, err := encodeSessionInfo(info)
	if err != nil {
		return "", nil, err
	}
	return token, info, nil
}

func (fakeCredProvider) ValidateSession(tokenString string) (*abstract.SessionInfo, error) {
	return decodeSessionInfo(tokenString)
}

func (fakeCredProvider) RefreshSession(info *abstract.SessionInfo) (string, error) {
	return encodeSessionInfo(info)
}

func (fakeCredProvider) IssueResetToken(_ string) (string, error) {
	return "", fmt.Errorf("not supported by test provider")
}

func (fakeCredProvider) ValidateResetToken(_ string) (string, error) {
	return "", fmt.Errorf("not supported by test provider")
}

func encodeSessionInfo(info *abstract.SessionInfo) (string, error) {
	b, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeSessionInfo(token string) (*abstract.SessionInfo, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var info abstract.SessionInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, err
	}
	if time.Now().Unix() > info.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}
	return &info, nil
}

type fakeUserIdentity struct {
	id        string
	email     string
	tenantID  string
	perms     []string
	tokenVer  int
	activeErr error
}

func (u fakeUserIdentity) GetID() string        { return u.id }
func (u fakeUserIdentity) GetEmail() string     { return u.email }
func (u fakeUserIdentity) GetTenantID() string  { return u.tenantID }
func (u fakeUserIdentity) GetPermissions() []string { return u.perms }
func (u fakeUserIdentity) GetTokenVersion() int { return u.tokenVer }

type fakeUserResolver struct {
	users map[string]fakeUserIdentity
}

func (r *fakeUserResolver) GetActiveByID(_ context.Context, userID string) (abstract.UserIdentity, error) {
	if u, ok := r.users[userID]; ok {
		if u.activeErr != nil {
			return nil, u.activeErr
		}
		return u, nil
	}
	return nil, runtime.ErrNotFound
}

// TestInternalRegistrationsAreRoutingOnly pins the surface contract: a
// registration flagged Internal:true is never installed as an HTTP route — it
// is dispatchable in-process but not reachable from the transport.
func TestInternalRegistrationsAreRoutingOnly(t *testing.T) {
	mt := newMockTransport()
	internal := abstract.MessageRegistration{
		Name:     "sys:internal:secret:run",
		Intent:   abstract.Check,
		Internal: true,
		Handler: func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
			return &abstract.Result{}, nil
		},
	}
	public := abstract.MessageRegistration{
		Name:   "sys:public:ping:run",
		Intent: abstract.Check,
		Handler: func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
			return &abstract.Result{}, nil
		},
	}

	orch := &Interface{
		trans:        mt,
		disp:         &mockDispatcher{},
		regs:         []abstract.MessageRegistration{internal, public},
		bootstrapped: true,
	}
	orch.installDispatcherRegistrations()

	mt.mu.Lock()
	defer mt.mu.Unlock()
	if _, ok := mt.handlers["POST /sys/internal/secret/run"]; ok {
		t.Fatal("internal registration must not be installed as an HTTP route")
	}
	if _, ok := mt.handlers["POST /sys/public/ping/run"]; !ok {
		t.Fatal("public registration must be installed as an HTTP route")
	}
}

// TestBootstrapSafeInstallSkipsInternal pins that internal registrations are
// also excluded from the pre-bootstrap surface.
func TestBootstrapSafeInstallSkipsInternal(t *testing.T) {
	mt := newMockTransport()
	internal := abstract.MessageRegistration{
		Name:         "sys:internal:bootstrap:run",
		Intent:       abstract.Check,
		Internal:     true,
		BootstrapSafe: true,
		Handler: func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
			return &abstract.Result{}, nil
		},
	}

	orch := &Interface{
		trans:        mt,
		disp:         &mockDispatcher{},
		regs:         []abstract.MessageRegistration{internal},
		bootstrapped: false,
	}
	orch.installBootstrapSafeRegistrations()

	mt.mu.Lock()
	defer mt.mu.Unlock()
	if len(mt.handlers) != 0 {
		t.Fatalf("internal bootstrap-safe registration must not be exposed pre-bootstrap, got %d routes", len(mt.handlers))
	}
}

// ---- auth middleware revocation / expiry ----

func middlewareInterface(t *testing.T, resolver *fakeUserResolver) (*Interface, abstract.CredentialsProvider) {
	t.Helper()
	return &Interface{
		credProv:     fakeCredProvider{},
		cookieCfg:    runtime.CookieConfig{SessionName: "session"},
		userModel:    resolver,
		idleTTL:      time.Hour,
		refreshTTL:   30 * time.Minute,
		noRefreshOps: map[string]struct{}{},
	}, fakeCredProvider{}
}

func sessionCookie(t *testing.T, credProv abstract.CredentialsProvider, userID string, ttl time.Duration) string {
	t.Helper()
	token, _, err := credProv.CreateSession(userID, ttl)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return token
}

func TestAuthMiddleware_RevokedSessionRejected(t *testing.T) {
	resolver := &fakeUserResolver{users: map[string]fakeUserIdentity{
		"u1": {id: "u1", email: "a@b.com", tokenVer: 1}, // user version bumped after the token was issued
	}}
	o, credProv := middlewareInterface(t, resolver)

	token := sessionCookie(t, credProv, "u1", time.Hour)
	req := Request{Cookies: map[string]string{"session": token}}

	nextCalled := false
	_, err := o.authMiddleware(context.Background(), req, func(ctx context.Context, r Request) (Response, error) {
		nextCalled = true
		return Response{Status: 200}, nil
	})
	if err == nil {
		t.Fatal("stale token_version session must be rejected, got nil")
	}
	if nextCalled {
		t.Fatal("handler must not run for a revoked session")
	}
}

func TestAuthMiddleware_ExpiredSessionRejected(t *testing.T) {
	resolver := &fakeUserResolver{users: map[string]fakeUserIdentity{
		"u1": {id: "u1", email: "a@b.com", tokenVer: 0},
	}}
	o, credProv := middlewareInterface(t, resolver)

	// Negative TTL produces an already-expired token.
	token := sessionCookie(t, credProv, "u1", -time.Hour)
	req := Request{Cookies: map[string]string{"session": token}}

	nextCalled := false
	_, err := o.authMiddleware(context.Background(), req, func(ctx context.Context, r Request) (Response, error) {
		nextCalled = true
		return Response{Status: 200}, nil
	})
	if err == nil {
		t.Fatal("expired session must be rejected, got nil")
	}
	if nextCalled {
		t.Fatal("handler must not run for an expired session")
	}
}

func TestAuthMiddleware_ValidSessionProceeds(t *testing.T) {
	resolver := &fakeUserResolver{users: map[string]fakeUserIdentity{
		"u1": {id: "u1", email: "a@b.com", perms: []string{"administrator"}, tokenVer: 0},
	}}
	o, credProv := middlewareInterface(t, resolver)

	token := sessionCookie(t, credProv, "u1", time.Hour)
	req := Request{Cookies: map[string]string{"session": token}}

	nextCalled := false
	resp, err := o.authMiddleware(context.Background(), req, func(ctx context.Context, r Request) (Response, error) {
		nextCalled = true
		return Response{Status: 200}, nil
	})
	if err != nil {
		t.Fatalf("valid session must proceed: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("expected status 200, got %d", resp.Status)
	}
	if !nextCalled {
		t.Fatal("handler must run for a valid session")
	}
}

func TestAuthMiddleware_UnknownUserRejected(t *testing.T) {
	resolver := &fakeUserResolver{users: map[string]fakeUserIdentity{}}
	o, credProv := middlewareInterface(t, resolver)

	token := sessionCookie(t, credProv, "ghost", time.Hour)
	req := Request{Cookies: map[string]string{"session": token}}

	nextCalled := false
	_, err := o.authMiddleware(context.Background(), req, func(ctx context.Context, r Request) (Response, error) {
		nextCalled = true
		return Response{Status: 200}, nil
	})
	if err == nil {
		t.Fatal("session for an unknown/inactive user must be rejected, got nil")
	}
	if nextCalled {
		t.Fatal("handler must not run for an unknown user")
	}
}

var _ abstract.UserResolver = (*fakeUserResolver)(nil)
var _ abstract.UserIdentity = fakeUserIdentity{}