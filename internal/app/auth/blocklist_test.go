package auth

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/hestia/internal/utility/persistest"
)

func setupBlocklistService(t *testing.T) (*TokenBlocklistService, func()) {
	t.Helper()
	p := persistest.NewPersistence(t)
	return NewTokenBlocklistService(p), func() {}
}

func TestBlocklistService_BlocklistAndCheck(t *testing.T) {
	svc, cleanup := setupBlocklistService(t)
	defer cleanup()
	ctx := context.Background()

	blocklisted, err := svc.IsBlocklisted(ctx, "nonexistent-jti")
	if err != nil {
		t.Fatalf("IsBlocklisted failed: %v", err)
	}
	if blocklisted {
		t.Error("expected false for non-existent jti")
	}

	jti := "test-jti-12345"
	exp := time.Now().Add(1 * time.Hour).Unix()
	if err := svc.Blocklist(ctx, jti, exp, "user-1"); err != nil {
		t.Fatalf("Blocklist failed: %v", err)
	}

	blocklisted, err = svc.IsBlocklisted(ctx, jti)
	if err != nil {
		t.Fatalf("IsBlocklisted failed: %v", err)
	}
	if !blocklisted {
		t.Error("expected true for blocklisted jti")
	}
}

func TestBlocklistService_MultipleEntries(t *testing.T) {
	svc, cleanup := setupBlocklistService(t)
	defer cleanup()
	ctx := context.Background()

	entries := []struct {
		jti    string
		userID string
	}{
		{"jti-1", "user-1"},
		{"jti-2", "user-2"},
		{"jti-3", "user-1"},
	}

	exp := time.Now().Add(1 * time.Hour).Unix()
	for _, e := range entries {
		if err := svc.Blocklist(ctx, e.jti, exp, e.userID); err != nil {
			t.Fatalf("Blocklist %s failed: %v", e.jti, err)
		}
	}

	for _, e := range entries {
		blocklisted, err := svc.IsBlocklisted(ctx, e.jti)
		if err != nil {
			t.Fatalf("IsBlocklisted %s failed: %v", e.jti, err)
		}
		if !blocklisted {
			t.Errorf("expected %s to be blocklisted", e.jti)
		}
	}

	blocklisted, err := svc.IsBlocklisted(ctx, "non-existent")
	if err != nil {
		t.Fatalf("IsBlocklisted failed: %v", err)
	}
	if blocklisted {
		t.Error("expected non-existent jti to not be blocklisted")
	}
}

func TestBlocklistService_DuplicateJTI(t *testing.T) {
	svc, cleanup := setupBlocklistService(t)
	defer cleanup()
	ctx := context.Background()

	jti := "duplicate-jti"
	exp := time.Now().Add(1 * time.Hour).Unix()
	if err := svc.Blocklist(ctx, jti, exp, "user-1"); err != nil {
		t.Fatalf("first Blocklist failed: %v", err)
	}

	if err := svc.Blocklist(ctx, jti, exp, "user-2"); err == nil {
		t.Error("expected error for duplicate jti")
	}
}
