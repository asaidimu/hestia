package abstract

import (
	"context"
	"time"
)

// UserIdentity exposes the identity-relevant subset of an active user for
// authentication middleware and session revocation checks.
type UserIdentity interface {
	GetID() string
	GetEmail() string
	GetTenantID() string
	GetPermissions() []string
	GetTokenVersion() int
}

type UserResolver interface {
	GetActiveByID(ctx context.Context, userID string) (UserIdentity, error)
}

type Claims struct {
	UserID     string   `json:"user_id"`
	Email      string   `json:"email"`
	TenantID   string   `json:"tenant_id"`
	Scopes     []string `json:"permissions"`
	Operations []string `json:"operations,omitempty"`
	TokenType  string   `json:"token_type"`
	TokenID    string   `json:"token_id,omitempty"`
	ExpiresAt  int64    `json:"expires_at,omitempty"`
}

type SessionInfo struct {
	SessionID    string
	UserID       string
	IssuedAt     int64
	ExpiresAt    int64
	CreatedAt    int64
	TokenVersion int
}

type CredentialsProvider interface {
	CreateSession(userID string, ttl time.Duration) (token string, info *SessionInfo, err error)
	ValidateSession(tokenString string) (*SessionInfo, error)
	RefreshSession(info *SessionInfo) (newToken string, err error)
	IssueResetToken(userID string) (token string, err error)
	ValidateResetToken(tokenString string) (userID string, err error)
}
