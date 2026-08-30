package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/auth"
	"github.com/asaidimu/hestia/core/system/auth/model"
)

func newBlocklistProvider(t *testing.T) (abstract.CredentialsProvider, *auth.TokenBlocklist) {
	t.Helper()
	model.DangerouslyResetSystemTokenBlocklistsModel()
	bl, err := auth.NewTokenBlocklist(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("NewTokenBlocklist: %v", err)
	}
	svc := auth.NewSessionService("test-blocklist-secret")
	return auth.NewCredentialsProviderWithVersion(svc, "reset-secret", nil, bl), bl
}

// S-4: logout must actually revoke — a revoked session ID fails validation
// everywhere, because every transport funnels through ValidateSession.
func TestSessionRevocation(t *testing.T) {
	prov, bl := newBlocklistProvider(t)

	token, info, err := prov.CreateSession("user-1", time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := prov.ValidateSession(token); err != nil {
		t.Fatalf("ValidateSession (pre-revoke): %v", err)
	}

	if err := bl.Revoke(context.Background(), info.SessionID, "user-1", info.ExpiresAt); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := bl.Revoke(context.Background(), info.SessionID, "user-1", info.ExpiresAt); err != nil {
		t.Fatalf("Revoke (idempotent): %v", err)
	}

	_, err = prov.ValidateSession(token)
	if err == nil {
		t.Fatal("expected revoked session to fail validation")
	}
	if !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("expected revocation error, got: %v", err)
	}
}

// S-4: refresh rotates the session ID and revokes the previous one.
func TestSessionRotationOnRefresh(t *testing.T) {
	prov, bl := newBlocklistProvider(t)

	_, info, err := prov.CreateSession("user-1", time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	newToken, err := prov.RefreshSession(info)
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}

	fresh, err := prov.ValidateSession(newToken)
	if err != nil {
		t.Fatalf("ValidateSession (rotated): %v", err)
	}
	if fresh.SessionID == info.SessionID {
		t.Fatal("expected session ID rotation on refresh")
	}
	if revoked, _ := bl.Revoked(context.Background(), info.SessionID); !revoked {
		t.Fatal("expected previous session ID to be revoked")
	}
}

// S-13: a reset token is single-use; the unique index on jti rejects any
// second consumption.
func TestResetTokenConsumption(t *testing.T) {
	prov, _ := newBlocklistProvider(t)

	tok, err := prov.IssueResetToken("user-1")
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}
	if _, err := prov.ValidateResetToken(tok); err != nil {
		t.Fatalf("ValidateResetToken (fresh): %v", err)
	}
	if err := prov.ConsumeResetToken(context.Background(), tok); err != nil {
		t.Fatalf("ConsumeResetToken: %v", err)
	}
	if err := prov.ConsumeResetToken(context.Background(), tok); err == nil {
		t.Fatal("expected replay to be rejected")
	} else if !strings.Contains(err.Error(), "already been used") {
		t.Fatalf("expected replay error, got: %v", err)
	}
	if _, err := prov.ValidateResetToken(tok); err == nil {
		t.Fatal("expected consumed token to fail validation")
	}

	// Another user's token is unaffected by the consumption.
	other, err := prov.IssueResetToken("user-2")
	if err != nil {
		t.Fatalf("IssueResetToken (user-2): %v", err)
	}
	if _, err := prov.ValidateResetToken(other); err != nil {
		t.Fatalf("ValidateResetToken (user-2): %v", err)
	}
}

// Blocklist rows carry their token's expiry; pruning removes expired rows
// and must leave live revocations intact.
func TestBlocklistPrune(t *testing.T) {
	_, bl := newBlocklistProvider(t)

	sid := "prune-test-live-sid"
	if err := bl.Revoke(context.Background(), sid, "user-1", time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatalf("Revoke (live): %v", err)
	}

	stale := "prune-test-stale-sid"
	if err := bl.Revoke(context.Background(), stale, "user-1", time.Now().Add(-time.Minute).Unix()); err != nil {
		t.Fatalf("Revoke (stale): %v", err)
	}

	removed, err := bl.Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed == 0 {
		t.Fatal("expected at least one expired row to be pruned")
	}
	if revoked, _ := bl.Revoked(context.Background(), stale); revoked {
		t.Fatal("stale row should have been pruned")
	}
	if revoked, _ := bl.Revoked(context.Background(), sid); !revoked {
		t.Fatal("live row must survive pruning")
	}
}
