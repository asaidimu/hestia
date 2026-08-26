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
)

type credentialProvider struct {
	sessionSvc     *SessionService
	secret         []byte
	getUserVersion func(ctx context.Context, userID string) (int, error)
}

func NewCredentialsProvider(sessionSvc *SessionService, resetSecret string) abstract.CredentialsProvider {
	return NewCredentialsProviderWithVersion(sessionSvc, resetSecret, nil)
}

func NewCredentialsProviderWithVersion(sessionSvc *SessionService, resetSecret string, getUserVersion func(ctx context.Context, userID string) (int, error)) abstract.CredentialsProvider {
	if getUserVersion == nil {
		getUserVersion = func(_ context.Context, _ string) (int, error) { return 0, nil }
	}
	return &credentialProvider{
		sessionSvc:     sessionSvc,
		secret:         []byte(resetSecret),
		getUserVersion: getUserVersion,
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
	st := &SessionToken{
		SessionID:    info.SessionID,
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

func (p *credentialProvider) ValidateResetToken(tokenString string) (string, error) {
	parts := strings.SplitN(tokenString, ".", 2)
	if len(parts) != 2 {
		return "", common.NewSystemError("INVALID_TOKEN_FORMAT", "invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", common.SystemErrorFrom(err).WithOperation("ValidateResetToken").WithMessage("invalid token payload")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", common.SystemErrorFrom(err).WithOperation("ValidateResetToken").WithMessage("invalid token signature")
	}

	mac := hmac.New(sha256.New, p.secret)
	mac.Write(payload)
	expected := mac.Sum(nil)[:16]
	if !hmac.Equal(sig, expected) {
		return "", common.NewSystemError("INVALID_TOKEN_SIGNATURE", "invalid token signature")
	}

	fields := strings.SplitN(string(payload), ":", 3)
	if len(fields) != 3 {
		return "", common.NewSystemError("INVALID_TOKEN_PAYLOAD", "invalid token payload")
	}

	exp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return "", common.SystemErrorFrom(err).WithOperation("ValidateResetToken").WithMessage("invalid token expiry")
	}

	if time.Now().Unix() > exp {
		return "", common.NewSystemError("TOKEN_EXPIRED", "token expired")
	}

	return fields[0], nil
}
