package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/asaidimu/hestia/app/core"
)

type JWTService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	resetTTL   time.Duration
}

type jwtClaims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Scopes    []string `json:"scopes"`
	TokenType string   `json:"token_type"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string, accessTTL, refreshTTL, resetTTL time.Duration) *JWTService {
	if accessTTL <= 0 {
		accessTTL = core.DefaultAccessTokenTTL
	}
	if refreshTTL <= 0 {
		refreshTTL = core.DefaultRefreshTokenTTL
	}
	if resetTTL <= 0 {
		resetTTL = core.DefaultResetTokenTTL
	}
	return &JWTService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		resetTTL:   resetTTL,
	}
}

func NewJWTServiceWithDefaults(secret string) *JWTService {
	return NewJWTService(secret, 0, 0, 0)
}

func (s *JWTService) GenerateAccessToken(userID, email string, scopes []string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID:    userID,
		Email:     email,
		Scopes:    scopes,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.Must(uuid.NewV7()).String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hestia",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) GenerateRefreshToken(userID, email string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID:    userID,
		Email:     email,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.Must(uuid.NewV7()).String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hestia",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) GenerateResetToken(userID, email string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID:    userID,
		Email:     email,
		TokenType: "password_reset",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.Must(uuid.NewV7()).String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.resetTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hestia",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) ValidateToken(tokenString string) (*core.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, core.ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, core.ErrInvalidToken.WithCause(err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, core.ErrInvalidToken.WithCause(err)
	}

	expiresAt := int64(0)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Unix()
	}

	return &core.Claims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Scopes:    claims.Scopes,
		TokenType: claims.TokenType,
		TokenID:   claims.ID,
		ExpiresAt: expiresAt,
	}, nil
}
