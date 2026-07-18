package auth

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/users"
)

type credentialProvider struct {
	jwtSvc       *JWTService
	blocklistSvc *TokenBlocklistService
	userModel    *users.UserModel
}

func NewCredentialsProvider(jwtSvc *JWTService, blocklistSvc *TokenBlocklistService, userModel *users.UserModel) abstract.CredentialsProvider {
	return &credentialProvider{
		jwtSvc:       jwtSvc,
		blocklistSvc: blocklistSvc,
		userModel:    userModel,
	}
}

func (p *credentialProvider) IssueAccess(userID, email string, scopes []string) (string, error) {
	return p.jwtSvc.GenerateAccessToken(userID, email, scopes)
}

func (p *credentialProvider) IssueRefresh(userID, email string) (string, error) {
	return p.jwtSvc.GenerateRefreshToken(userID, email)
}

func (p *credentialProvider) IssueReset(userID, email string) (string, error) {
	return p.jwtSvc.GenerateResetToken(userID, email)
}

func (p *credentialProvider) Validate(tokenString string) (*abstract.Claims, error) {
	return p.jwtSvc.ValidateToken(tokenString)
}

func (p *credentialProvider) Revoked(ctx context.Context, tokenID string) (bool, error) {
	return p.blocklistSvc.IsBlocklisted(ctx, tokenID)
}

func (p *credentialProvider) Revoke(ctx context.Context, claims *abstract.Claims) error {
	if claims.TokenID == "" {
		return fmt.Errorf("cannot revoke a token without a TokenID")
	}
	return p.blocklistSvc.Blocklist(ctx, claims.TokenID, claims.ExpiresAt, claims.UserID)
}

func (p *credentialProvider) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := p.jwtSvc.ValidateToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token")
	}
	if claims.TokenType != "refresh" {
		return "", "", fmt.Errorf("invalid refresh token")
	}

	if claims.TokenID != "" {
		blocklisted, err := p.blocklistSvc.IsBlocklisted(ctx, claims.TokenID)
		if err != nil {
			return "", "", fmt.Errorf("check blocklist: %w", err)
		}
		if blocklisted {
			return "", "", fmt.Errorf("refresh token has been revoked")
		}
	}

	scopes := claims.Scopes
	user, err := p.userModel.GetByID(ctx, claims.UserID)
	if err == nil {
		if rawPerms, err := user.GetStringArray("permissions"); err == nil && len(rawPerms) > 0 {
			scopes = rawPerms
		}
	}

	if claims.TokenID != "" && claims.ExpiresAt > 0 {
		if err := p.blocklistSvc.Blocklist(ctx, claims.TokenID, claims.ExpiresAt, claims.UserID); err != nil {
			return "", "", fmt.Errorf("rotate refresh token: %w", err)
		}
	}

	accessToken, err := p.jwtSvc.GenerateAccessToken(claims.UserID, claims.Email, scopes)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	newRefresh, err := p.jwtSvc.GenerateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, newRefresh, nil
}
