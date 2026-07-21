package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SessionToken struct {
	SessionID string `json:"sid"`
	UserID    string `json:"uid"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	CreatedAt int64  `json:"crt"`
}

type SessionService struct {
	secret []byte
}

func NewSessionService(secret string) *SessionService {
	return &SessionService{secret: []byte(secret)}
}

func (s *SessionService) Create(userID string, absoluteTTL time.Duration) (string, *SessionToken, error) {
	now := time.Now().Unix()
	st := &SessionToken{
		SessionID: uuid.Must(uuid.NewV7()).String(),
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now + int64(absoluteTTL.Seconds()),
		CreatedAt: now,
	}
	token, err := s.encode(st)
	if err != nil {
		return "", nil, fmt.Errorf("encode session: %w", err)
	}
	return token, st, nil
}

func (s *SessionService) Refresh(st *SessionToken) (string, *SessionToken, error) {
	now := time.Now().Unix()
	refreshed := &SessionToken{
		SessionID: st.SessionID,
		UserID:    st.UserID,
		IssuedAt:  now,
		ExpiresAt: st.ExpiresAt,
		CreatedAt: st.CreatedAt,
	}
	token, err := s.encode(refreshed)
	if err != nil {
		return "", nil, fmt.Errorf("encode refreshed session: %w", err)
	}
	return token, refreshed, nil
}

func (s *SessionService) Validate(token string) (*SessionToken, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token signature: %w", err)
	}

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	expected := mac.Sum(nil)[:16]
	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("invalid token signature")
	}

	var st SessionToken
	if err := json.Unmarshal(payload, &st); err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}

	return &st, nil
}

func (s *SessionService) encode(st *SessionToken) (string, error) {
	payload, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	sig := mac.Sum(nil)[:16]
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
