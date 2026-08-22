package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

const testSecret = "test-session-secret-please-change-me"

// TestSessionToken_IsNotJWT pins the format contract: hestia session tokens
// are a two-part opaque blob (payload.signature) — NOT a JWT. JWTs are
// three-part (header.payload.signature). This is a deliberate design decision
// that the docs must never contradict.
func TestSessionToken_IsNotJWT(t *testing.T) {
	svc := NewSessionService(testSecret)
	token, _, err := svc.Create("u1", time.Hour, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Fatalf("session token must have exactly 2 dot-separated parts (got %d); a JWT has 3", len(parts))
	}
	if parts[0] == "" || parts[1] == "" {
		t.Fatal("neither payload nor signature may be empty")
	}
}

// TestSessionToken_PayloadContainsNoSecret verifies the signing secret never
// leaks into the token payload (only uid/sid/timestamps/version are present).
func TestSessionToken_PayloadContainsNoSecret(t *testing.T) {
	svc := NewSessionService(testSecret)
	token, _, err := svc.Create("u1", time.Hour, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	payload := strings.Split(token, ".")[0]
	if strings.Contains(payload, testSecret) {
		t.Fatal("session token payload must never contain the signing secret")
	}

	// The decoded payload must contain exactly the session claims.
	rawPayload, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	for _, key := range []string{"sid", "uid", "iat", "exp", "crt", "tv"} {
		if !strings.Contains(string(rawPayload), `"`+key+`"`) {
			t.Fatalf("payload must contain claim key %q, got %s", key, string(rawPayload))
		}
	}
	if strings.Contains(string(rawPayload), "password") {
		t.Fatal("payload must never contain password material")
	}
}

// TestSessionToken_TamperedPayloadRejected flips a byte in the payload and
// expects signature verification to fail.
func TestSessionToken_TamperedPayloadRejected(t *testing.T) {
	svc := NewSessionService(testSecret)
	token, _, err := svc.Create("u1", time.Hour, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	parts := strings.Split(token, ".")
	tampered := "A" + parts[0][1:] + "." + parts[1]
	_, err = svc.Validate(tampered)
	if err == nil {
		t.Fatal("tampered payload must fail validation, got nil")
	}
	if !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature error, got: %v", err)
	}
}

// TestSessionToken_TamperedSignatureRejected flips a byte in the signature.
func TestSessionToken_TamperedSignatureRejected(t *testing.T) {
	svc := NewSessionService(testSecret)
	token, _, err := svc.Create("u1", time.Hour, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + "B" + parts[1][1:]
	if _, err := svc.Validate(tampered); err == nil {
		t.Fatal("tampered signature must fail validation, got nil")
	}
}

// TestSessionToken_RejectsTokenSignedByDifferentSecret verifies tokens are
// invalidated when the server secret rotates.
func TestSessionToken_RejectsTokenSignedByDifferentSecret(t *testing.T) {
	token, _, err := NewSessionService("old-secret").Create("u1", time.Hour, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := NewSessionService("new-secret").Validate(token); err == nil {
		t.Fatal("token signed with a different secret must be rejected, got nil")
	}
}

// TestSessionToken_CarriesTokenVersion verifies the token embeds the user's
// token version so the HTTP middleware can revoke sessions on password/secret
// rotation (token_version bump).
func TestSessionToken_CarriesTokenVersion(t *testing.T) {
	svc := NewSessionService(testSecret)
	token, _, err := svc.Create("u1", time.Hour, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	validated, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if validated.TokenVersion != 7 {
		t.Fatalf("expected TokenVersion 7, got %d", validated.TokenVersion)
	}
}

// TestSessionService_ValidateDoesNotEnforceExpiry documents the responsibility
// boundary: SessionService.Validate verifies authenticity ONLY. Expiry and
// TestSessionService_ValidateRejectsExpiredToken verifies that Validate
// enforces ExpiresAt, so callers don't need to check expiry themselves.
func TestSessionService_ValidateRejectsExpiredToken(t *testing.T) {
	svc := NewSessionService(testSecret)

	// An already-expired token (negative TTL) must be rejected.
	token, _, err := svc.Create("u1", -time.Hour, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = svc.Validate(token)
	if err == nil {
		t.Fatal("Validate must reject expired tokens")
	}
}

// TestSessionToken_RefreshPreservesVersionAndSession verifies Refresh keeps
// the session id and token version (revocation must survive a refresh) while
// bumping the issued-at timestamp for the sliding window.
func TestSessionToken_RefreshPreservesVersionAndSession(t *testing.T) {
	svc := NewSessionService(testSecret)
	token, st, err := svc.Create("u1", time.Hour, 3)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = token

	refreshedToken, refreshed, err := svc.Refresh(st)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if refreshed.SessionID != st.SessionID {
		t.Fatalf("Refresh changed session id: %q -> %q", st.SessionID, refreshed.SessionID)
	}
	if refreshed.TokenVersion != 3 {
		t.Fatalf("Refresh must preserve token version, got %d", refreshed.TokenVersion)
	}

	validated, err := svc.Validate(refreshedToken)
	if err != nil {
		t.Fatalf("refreshed token must validate: %v", err)
	}
	if validated.UserID != "u1" || validated.SessionID != st.SessionID {
		t.Fatalf("refreshed token claims changed: %+v", validated)
	}
}

func TestSessionToken_FormatErrors(t *testing.T) {
	svc := NewSessionService(testSecret)
	for _, tok := range []string{"", "no-dot-here", "onlyone.", "....", "a.b.c"} {
		if _, err := svc.Validate(tok); err == nil {
			t.Fatalf("expected error for malformed token %q, got nil", tok)
		}
	}
}
