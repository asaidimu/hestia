package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/asaidimu/hestia/internal/core"
)

type Service struct {
	secret []byte
}

func NewService(secret string) *Service {
	return &Service{secret: []byte(secret)}
}

func (s *Service) Generate(userID, email string, scopes []string, ttl time.Duration) (string, *core.Claims, error) {
	now := time.Now()
	exp := now.Add(ttl).Unix()
	nonce := uuid.Must(uuid.NewV7()).String()
	payload := fmt.Sprintf("%s:%s:%s:%d:%s", userID, email, strings.Join(scopes, ","), exp, nonce)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:10])
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))

	claims := &core.Claims{
		UserID:    userID,
		Email:     email,
		Scopes:    scopes,
		TokenType: "session",
		TokenID:   nonce,
		ExpiresAt: exp,
	}
	return encoded + "." + sig, claims, nil
}

func (s *Service) Validate(token string) (*core.Claims, error) {
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
	expected := mac.Sum(nil)[:10]
	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("invalid token signature")
	}

	fields := strings.SplitN(string(payload), ":", 5)
	if len(fields) != 5 {
		return nil, fmt.Errorf("invalid token payload fields")
	}

	userID := fields[0]
	email := fields[1]
	scopes := []string{}
	if fields[2] != "" {
		scopes = strings.Split(fields[2], ",")
	}
	exp, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid token expiry: %w", err)
	}
	nonce := fields[4]

	if time.Now().Unix() > exp {
		return nil, fmt.Errorf("token expired")
	}

	return &core.Claims{
		UserID:    userID,
		Email:     email,
		Scopes:    scopes,
		TokenType: "session",
		TokenID:   nonce,
		ExpiresAt: exp,
	}, nil
}
