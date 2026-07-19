package abstract

import "time"

type Claims struct {
	UserID     string   `json:"user_id"`
	Email      string   `json:"email"`
	Scopes     []string `json:"permissions"`
	Operations []string `json:"operations,omitempty"`
	TokenType  string   `json:"token_type"`
	TokenID    string   `json:"token_id,omitempty"`
	ExpiresAt  int64    `json:"expires_at,omitempty"`
}

type SessionInfo struct {
	SessionID string
	UserID    string
	IssuedAt  int64
	ExpiresAt int64
	CreatedAt int64
}

type CredentialsProvider interface {
	CreateSession(userID string, ttl time.Duration) (token string, info *SessionInfo, err error)
	ValidateSession(tokenString string) (*SessionInfo, error)
	RefreshSession(info *SessionInfo) (newToken string, err error)
	IssueResetToken(userID string) (token string, err error)
	ValidateResetToken(tokenString string) (userID string, err error)
}
