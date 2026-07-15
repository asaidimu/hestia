package api

import (
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	s := NewService("super-secret-key")
	if s == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestGenerateAndValidate(t *testing.T) {
	s := NewService("super-secret-key")
	userID := "usr_abc123"
	email := "alice@example.com"
	scopes := []string{"read", "write"}
	ttl := 30 * time.Minute

	token, c1, err := s.Generate(userID, email, scopes, ttl)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if token == "" {
		t.Fatal("Generate returned empty token")
	}
	if c1.UserID != userID || c1.Email != email || c1.TokenType != "session" {
		t.Fatalf("unexpected claims: %+v", c1)
	}

	c2, err := s.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if c2.UserID != userID {
		t.Fatalf("UserID: got %q, want %q", c2.UserID, userID)
	}
	if c2.Email != email {
		t.Fatalf("Email: got %q, want %q", c2.Email, email)
	}
	if c2.TokenType != "session" {
		t.Fatalf("TokenType: got %q, want session", c2.TokenType)
	}
	if c2.TokenID != c1.TokenID {
		t.Fatalf("TokenID: got %q, want %q", c2.TokenID, c1.TokenID)
	}
	if c2.ExpiresAt != c1.ExpiresAt {
		t.Fatalf("ExpiresAt: got %d, want %d", c2.ExpiresAt, c1.ExpiresAt)
	}
	if len(c2.Scopes) != len(scopes) {
		t.Fatalf("Scopes len: got %d, want %d", len(c2.Scopes), len(scopes))
	}
	for i, s := range scopes {
		if c2.Scopes[i] != s {
			t.Fatalf("Scopes[%d]: got %q, want %q", i, c2.Scopes[i], s)
		}
	}
}

func TestValidateInvalidToken(t *testing.T) {
	s := NewService("secret")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no delimiter", "justasinglepart"},
		{"garbage", "!!!.!!!"},
		{"bad base64 payload", "not-valid-base64!!.c2ln"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Validate(tt.token)
			if err == nil {
				t.Fatal("expected error for invalid token")
			}
		})
	}
}

func TestValidateExpiredToken(t *testing.T) {
	s := NewService("secret")
	token, _, err := s.Generate("u1", "u@b.com", nil, -1*time.Hour)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	_, err = s.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}
