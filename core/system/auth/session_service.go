package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/asaidimu/go-anansi/v8/core/common"
)

type SessionToken struct {
	SessionID    string `json:"sid"`
	UserID       string `json:"uid"`
	IssuedAt     int64  `json:"iat"`
	ExpiresAt    int64  `json:"exp"`
	CreatedAt    int64  `json:"crt"`
	TokenVersion int    `json:"tv"`
}

type SessionService struct {
	secret []byte
}

func NewSessionService(secret string) *SessionService {
	return &SessionService{secret: []byte(secret)}
}

func (s *SessionService) Create(userID string, absoluteTTL time.Duration, tokenVersion int) (string, *SessionToken, error) {
	now := time.Now().Unix()
	st := &SessionToken{
		SessionID:    uuid.Must(uuid.NewV7()).String(),
		UserID:       userID,
		IssuedAt:     now,
		ExpiresAt:    now + int64(absoluteTTL.Seconds()),
		CreatedAt:    now,
		TokenVersion: tokenVersion,
	}
	token, err := s.encode(st)
	if err != nil {
		return "", nil, common.SystemErrorFrom(err).WithOperation("CreateSession").WithMessage("encode session")
	}
	return token, st, nil
}

func (s *SessionService) Refresh(st *SessionToken) (string, *SessionToken, error) {
	now := time.Now().Unix()
	refreshed := &SessionToken{
		SessionID:    st.SessionID,
		UserID:       st.UserID,
		IssuedAt:     now,
		ExpiresAt:    st.ExpiresAt,
		CreatedAt:    st.CreatedAt,
		TokenVersion: st.TokenVersion,
	}
	token, err := s.encode(refreshed)
	if err != nil {
		return "", nil, common.SystemErrorFrom(err).WithOperation("Refresh").WithMessage("encode refreshed session")
	}
	return token, refreshed, nil
}

// @note #review2-20260821-001 issue resolved P1 #security,#auth,#review : SessionService.Validate does not check ExpiresAt
// @assignee opencode
// Moved ExpiresAt check into SessionService.Validate so expiry is enforced at the token level. Removed redundant expiry check from middleware.go. Callers in wails/dispatch.go that were missing the check now get it automatically.
//
// Validate() cryptographically verifies the HMAC and decodes the payload, but never compares ExpiresAt against time.Now(). Expiry is only enforced by callers that happen to re-check info.ExpiresAt after calling ValidateSession -- the HTTP middleware (core/interface/http/middleware.go) does this correctly, but utils/wails/dispatch.go has at least two call sites (around the post-login and dispatch-response handling paths) that call CredProvider.ValidateSession and consume the result without checking ExpiresAt at all. A method literally named Validate should be the single source of truth for token validity, including expiry; relying on every caller to remember a manual time check is fragile and has already produced inconsistent enforcement across call sites. Compare to ValidateResetToken in credential_provider.go, which does check its own expiry inline. Recommend moving the ExpiresAt (and ideally TokenVersion, though that IS consistently checked in middleware.go) comparison into SessionService.Validate itself.
func (s *SessionService) Validate(token string) (*SessionToken, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, common.NewSystemError("INVALID_TOKEN_FORMAT", "invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, common.SystemErrorFrom(err).WithOperation("Validate").WithMessage("invalid token payload")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, common.SystemErrorFrom(err).WithOperation("Validate").WithMessage("invalid token signature")
	}

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	expected := mac.Sum(nil)[:16]
	if !hmac.Equal(sig, expected) {
		return nil, common.NewSystemError("INVALID_TOKEN_SIGNATURE", "invalid token signature")
	}

	var st SessionToken
	if err := json.Unmarshal(payload, &st); err != nil {
		return nil, common.SystemErrorFrom(err).WithOperation("Validate").WithMessage("invalid token payload")
	}

	if time.Now().Unix() > st.ExpiresAt {
		return nil, common.NewSystemError("TOKEN_EXPIRED", "token expired")
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
