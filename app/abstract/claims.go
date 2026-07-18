package abstract

import "context"

type Claims struct {
	UserID     string   `json:"user_id"`
	Email      string   `json:"email"`
	Scopes     []string `json:"permissions"`
	Operations []string `json:"operations,omitempty"`
	TokenType  string   `json:"token_type"`
	TokenID    string   `json:"token_id,omitempty"`
	ExpiresAt  int64    `json:"expires_at,omitempty"`
}

type CredentialsProvider interface {
	IssueAccess(userID, email string, scopes []string) (token string, err error)
	IssueRefresh(userID, email string) (token string, err error)
	IssueReset(userID, email string) (token string, err error)
	Validate(tokenString string) (*Claims, error)

	// Refresh validates a refresh token, blocklists the old one (rotation),
	// re-reads current user scopes, and issues a new access+refresh pair.
	Refresh(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error)

	// Revoked checks whether a token has been blocklisted.
	Revoked(ctx context.Context, tokenID string) (bool, error)

	Revoke(ctx context.Context, claims *Claims) error
}
