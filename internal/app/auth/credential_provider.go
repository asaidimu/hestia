package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/asaidimu/hestia/app/abstract"
)

type credentialProvider struct {
	sessionSvc *SessionService
	secret     []byte
}

func NewCredentialsProvider(sessionSvc *SessionService, resetSecret string) abstract.CredentialsProvider {
	return &credentialProvider{
		sessionSvc: sessionSvc,
		secret:     []byte(resetSecret),
	}
}

func (p *credentialProvider) CreateSession(userID string, ttl time.Duration) (string, *abstract.SessionInfo, error) {
	token, st, err := p.sessionSvc.Create(userID, ttl)
	if err != nil {
		return "", nil, err
	}
	return token, &abstract.SessionInfo{
		SessionID: st.SessionID,
		UserID:    st.UserID,
		IssuedAt:  st.IssuedAt,
		ExpiresAt: st.ExpiresAt,
		CreatedAt: st.CreatedAt,
	}, nil
}

func (p *credentialProvider) ValidateSession(tokenString string) (*abstract.SessionInfo, error) {
	st, err := p.sessionSvc.Validate(tokenString)
	if err != nil {
		return nil, err
	}
	return &abstract.SessionInfo{
		SessionID: st.SessionID,
		UserID:    st.UserID,
		IssuedAt:  st.IssuedAt,
		ExpiresAt: st.ExpiresAt,
		CreatedAt: st.CreatedAt,
	}, nil
}

func (p *credentialProvider) RefreshSession(info *abstract.SessionInfo) (string, error) {
	st := &SessionToken{
		SessionID: info.SessionID,
		UserID:    info.UserID,
		IssuedAt:  info.IssuedAt,
		ExpiresAt: info.ExpiresAt,
		CreatedAt: info.CreatedAt,
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
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:10])
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encoded + "." + sig, nil
}

func (p *credentialProvider) ValidateResetToken(tokenString string) (string, error) {
	parts := strings.SplitN(tokenString, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid token payload: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token signature: %w", err)
	}

	mac := hmac.New(sha256.New, p.secret)
	mac.Write(payload)
	expected := mac.Sum(nil)[:10]
	if !hmac.Equal(sig, expected) {
		return "", fmt.Errorf("invalid token signature")
	}

	fields := strings.SplitN(string(payload), ":", 3)
	if len(fields) != 3 {
		return "", fmt.Errorf("invalid token payload")
	}

	exp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid token expiry: %w", err)
	}

	if time.Now().Unix() > exp {
		return "", fmt.Errorf("token expired")
	}

	return fields[0], nil
}
