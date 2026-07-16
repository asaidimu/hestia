package identity_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/identity"
)

func TestClaimsFromContext(t *testing.T) {
	claims := &core.Claims{
		UserID:    "u1",
		Email:     "test@example.com",
		Scopes:    []string{"admin"},
		TokenType: "access",
	}

	ctx := identity.ContextWithClaims(context.Background(), claims)
	got, ok := identity.ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("ClaimsFromContext returned false for context with claims")
	}
	if got.UserID != claims.UserID {
		t.Fatalf("expected UserID %q, got %q", claims.UserID, got.UserID)
	}
	if got.Email != claims.Email {
		t.Fatalf("expected Email %q, got %q", claims.Email, got.Email)
	}
	if len(got.Scopes) != 1 || got.Scopes[0] != "admin" {
		t.Fatalf("expected Scopes [admin], got %v", got.Scopes)
	}
	if got.TokenType != claims.TokenType {
		t.Fatalf("expected TokenType %q, got %q", claims.TokenType, got.TokenType)
	}
}

func TestClaimsFromContextMissing(t *testing.T) {
	_, ok := identity.ClaimsFromContext(context.Background())
	if ok {
		t.Fatal("ClaimsFromContext returned true for empty context")
	}
}

func TestContextWithClaimsSetsIAMIdentity(t *testing.T) {
	claims := &core.Claims{
		UserID:    "u2",
		Email:     "u2@test.com",
		Scopes:    []string{"read", "write"},
		TokenType: "access",
	}

	ctx := identity.ContextWithClaims(context.Background(), claims)
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		t.Fatal("expected IAM identity in context")
	}
	if len(ident.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(ident.Permissions))
	}
	props, ok := ident.Properties.(map[string]any)
	if !ok {
		t.Fatal("expected Properties to be map[string]any")
	}
	userID, _ := props["user_id"].(string)
	if userID != "u2" {
		t.Fatalf("expected user_id u2, got %q", userID)
	}
	email, _ := props["email"].(string)
	if email != "u2@test.com" {
		t.Fatalf("expected email u2@test.com, got %q", email)
	}
}

func TestSystemContext(t *testing.T) {
	ctx := identity.SystemContext(context.Background())

	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		t.Fatal("SystemContext did not set an IAM identity")
	}

	var hasSystemScope bool
	for _, p := range ident.Permissions {
		if p == "system:http" {
			hasSystemScope = true
			break
		}
	}
	if !hasSystemScope {
		t.Fatal("expected system:http permission in system context")
	}

	props, ok := ident.Properties.(map[string]any)
	if !ok {
		t.Fatal("expected Properties to be map[string]any")
	}
	sys, ok := props["system"].(string)
	if !ok || sys != "http" {
		t.Fatalf("expected Properties[system] = http, got %q", sys)
	}
}
