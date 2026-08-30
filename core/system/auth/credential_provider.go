package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/auth/model"
	"go.uber.org/zap"
)

type credentialProvider struct {
	sessionSvc     *SessionService
	secret         []byte
	getUserVersion func(ctx context.Context, userID string) (int, error)
	blocklist      *TokenBlocklist
}

func NewCredentialsProvider(sessionSvc *SessionService, resetSecret string) abstract.CredentialsProvider {
	return NewCredentialsProviderWithVersion(sessionSvc, resetSecret, nil, nil)
}

func NewCredentialsProviderWithVersion(sessionSvc *SessionService, resetSecret string, getUserVersion func(ctx context.Context, userID string) (int, error), blocklist *TokenBlocklist) abstract.CredentialsProvider {
	if getUserVersion == nil {
		getUserVersion = func(_ context.Context, _ string) (int, error) { return 0, nil }
	}
	return &credentialProvider{
		sessionSvc:     sessionSvc,
		secret:         []byte(resetSecret),
		getUserVersion: getUserVersion,
		blocklist:      blocklist,
	}
}

func (p *credentialProvider) CreateSession(userID string, ttl time.Duration) (string, *abstract.SessionInfo, error) {
	tokenVersion, _ := p.getUserVersion(context.Background(), userID)
	token, st, err := p.sessionSvc.Create(userID, ttl, tokenVersion)
	if err != nil {
		return "", nil, err
	}
	return token, &abstract.SessionInfo{
		SessionID:    st.SessionID,
		UserID:       st.UserID,
		IssuedAt:     st.IssuedAt,
		ExpiresAt:    st.ExpiresAt,
		CreatedAt:    st.CreatedAt,
		TokenVersion: st.TokenVersion,
	}, nil
}

func (p *credentialProvider) ValidateSession(tokenString string) (*abstract.SessionInfo, error) {
	st, err := p.sessionSvc.Validate(tokenString)
	if err != nil {
		return nil, err
	}
	// S-4 enforcement: a revoked (logged-out or rotated-out) session ID
	// fails validation on every transport, because they all funnel through
	// this choke point. Query errors fail open (a blocklist outage must
	// not lock out every session) but are logged by the blocklist.
	if p.blocklist.RevokedSafe(context.Background(), st.SessionID) {
		return nil, common.NewSystemError("TOKEN_REVOKED", "session has been revoked")
	}
	return &abstract.SessionInfo{
		SessionID:    st.SessionID,
		UserID:       st.UserID,
		IssuedAt:     st.IssuedAt,
		ExpiresAt:    st.ExpiresAt,
		CreatedAt:    st.CreatedAt,
		TokenVersion: st.TokenVersion,
	}, nil
}

func (p *credentialProvider) RefreshSession(info *abstract.SessionInfo) (string, error) {
	// S-4: rotate the session ID on refresh — a refreshed session must not
	// keep the identifier it was created with. The previous ID is revoked
	// (best effort) so a stolen pre-refresh token dies at the next refresh.
	previous := info.SessionID
	st := &SessionToken{
		SessionID:    uuid.Must(uuid.NewV7()).String(),
		UserID:       info.UserID,
		IssuedAt:     info.IssuedAt,
		ExpiresAt:    info.ExpiresAt,
		CreatedAt:    info.CreatedAt,
		TokenVersion: info.TokenVersion,
	}
	token, _, err := p.sessionSvc.Refresh(st)
	if err != nil {
		return "", err
	}
	if p.blocklist != nil && previous != "" && previous != st.SessionID {
		if err := p.blocklist.Revoke(context.Background(), previous, info.UserID, info.ExpiresAt); err != nil {
			// Losing this revocation keeps the old token valid until its
			// absolute expiry — make that loud.
			p.blocklist.logger.Error("session rotation: failed to revoke previous sid",
				zap.String("previous_sid", previous), zap.Error(err))
		}
	}
	return token, nil
}

func (p *credentialProvider) IssueResetToken(userID string) (string, error) {
	now := time.Now()
	exp := now.Add(5 * time.Minute).Unix()
	payload := fmt.Sprintf("%s:%d:%s", userID, exp, uuid.Must(uuid.NewV7()).String())
	mac := hmac.New(sha256.New, p.secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encoded + "." + sig, nil
}

// parseResetToken verifies the HMAC and decodes the payload into its
// userID, nonce and expiry components.
func (p *credentialProvider) parseResetToken(tokenString string) (string, string, int64, error) {
	parts := strings.SplitN(tokenString, ".", 2)
	if len(parts) != 2 {
		return "", "", 0, common.NewSystemError("INVALID_TOKEN_FORMAT", "invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", 0, common.SystemErrorFrom(err).WithOperation("ValidateResetToken").WithMessage("invalid token payload")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", 0, common.SystemErrorFrom(err).WithOperation("ValidateResetToken").WithMessage("invalid token signature")
	}

	mac := hmac.New(sha256.New, p.secret)
	mac.Write(payload)
	expected := mac.Sum(nil)[:16]
	if !hmac.Equal(sig, expected) {
		return "", "", 0, common.NewSystemError("INVALID_TOKEN_SIGNATURE", "invalid token signature")
	}

	fields := strings.SplitN(string(payload), ":", 3)
	if len(fields) != 3 {
		return "", "", 0, common.NewSystemError("INVALID_TOKEN_PAYLOAD", "invalid token payload")
	}

	exp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return "", "", 0, common.SystemErrorFrom(err).WithOperation("ValidateResetToken").WithMessage("invalid token expiry")
	}

	if time.Now().Unix() > exp {
		return "", "", 0, common.NewSystemError("TOKEN_EXPIRED", "token expired")
	}

	return fields[0], fields[2], exp, nil
}

func (p *credentialProvider) ValidateResetToken(tokenString string) (string, error) {
	userID, nonce, _, err := p.parseResetToken(tokenString)
	if err != nil {
		return "", err
	}
	// S-13: reject already-consumed tokens early (best-effort; the
	// authoritative enforcement is the unique-index insert below).
	if p.blocklist.RevokedSafe(context.Background(), "reset:"+nonce) {
		return "", common.NewSystemError("TOKEN_ALREADY_USED", "reset token has already been used")
	}
	return userID, nil
}

// ConsumeResetToken marks a validated reset token as used (S-13). The
// `jti` unique index makes concurrent double-spend impossible: exactly one
// insert wins and the loser is rejected as a replay.
func (p *credentialProvider) ConsumeResetToken(ctx context.Context, tokenString string) error {
	if p.blocklist == nil {
		return nil
	}
	userID, nonce, exp, err := p.parseResetToken(tokenString)
	if err != nil {
		return err
	}
	jti := "reset:" + nonce
	row := model.SystemTokenBlocklist{Jti: jti, UserID: userID, Exp: exp}
	if _, err := p.blocklist.model.Create(ctx, model.NewSystemTokenBlocklist(row)); err != nil {
		if revoked, rerr := p.blocklist.Revoked(ctx, jti); rerr == nil && revoked {
			return common.NewSystemError("TOKEN_ALREADY_USED", "reset token has already been used")
		}
		return common.SystemErrorFrom(err).WithOperation("ConsumeResetToken").WithMessage("persist consumed reset token")
	}
	return nil
}
