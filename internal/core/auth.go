package core

import (
	"context"
	"strings"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"
)

// SystemScopePrefix is the prefix for all system-level permission scopes.
// Override at build time: go build -ldflags '-X github.com/asaidimu/hestia/internal/core.SystemScopePrefix=hestia'
var SystemScopePrefix = "system"

func IsSystemIdentity(ctx context.Context) bool {
	identity, ok := iam.GetIdentity(ctx)
	if !ok {
		return false
	}
	for _, p := range identity.Permissions {
		if strings.HasPrefix(p, SystemScopePrefix+":") {
			return true
		}
	}
	return false
}

type Claims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Scopes    []string `json:"scopes"`
	TokenType string   `json:"token_type"`
	TokenID   string   `json:"token_id,omitempty"`
	ExpiresAt int64    `json:"expires_at,omitempty"`
}


type JWTService interface {
	GenerateAccessToken(userID, email string, scopes []string) (string, error)
	GenerateRefreshToken(userID, email string) (string, error)
	GenerateResetToken(userID, email string) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
}

type SessionService interface {
	Generate(userID, email string, scopes []string, ttl time.Duration) (string, *Claims, error)
	Validate(token string) (*Claims, error)
}
