package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	svc := NewJWTService("test-secret", 0, 0, 0)

	token, err := svc.GenerateAccessToken("user-1", "test@example.com", []string{"read:*"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("expected UserID 'user-1', got '%s'", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got '%s'", claims.Email)
	}
	if claims.TokenType != "access" {
		t.Errorf("expected TokenType 'access', got '%s'", claims.TokenType)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "read:*" {
		t.Errorf("expected scopes [read:*], got %v", claims.Scopes)
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	svc := NewJWTService("test-secret", 0, 0, 0)

	token, err := svc.GenerateRefreshToken("user-1", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("expected TokenType 'refresh', got '%s'", claims.TokenType)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	svc := NewJWTService("test-secret", 0, 0, 0)
	otherSvc := NewJWTService("different-secret", 0, 0, 0)

	token, err := svc.GenerateAccessToken("user-1", "test@example.com", nil)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = otherSvc.ValidateToken(token)
	if err == nil {
		t.Error("expected error for token signed with different secret")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	svc := NewJWTService("test-secret", 0, 0, 0)

	claims := jwtClaims{
		UserID:    "user-1",
		Email:     "test@example.com",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "hestia",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = svc.ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc := NewJWTService("test-secret", 0, 0, 0)

	token, err := svc.GenerateResetToken("user-1", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate reset token: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate reset token: %v", err)
	}

	if claims.TokenType != "password_reset" {
		t.Errorf("expected TokenType 'password_reset', got '%s'", claims.TokenType)
	}
}
