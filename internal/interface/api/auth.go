package api

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/app/core/identity"
)

// TODO: use constants for message names
func (o *Interface) validateBearer(ctx context.Context, req Request, token string) (*identity.Claims, error) {
	sysCtx := identity.SystemContext(ctx)

	validateMsg := abstract.NewMessage("system:auth:token:validate", sysCtx, data.MustNewDocument(map[string]any{
		"token": token,
	}, sysCtx))
	validateResult, err := o.internalDisp.Send(validateMsg)
	if err != nil {
		return nil, common.NewSystemError("UNAUTHORIZED", err.Error())
	}
	claims := extractClaims(validateResult.Document)

	if claims.TokenID != "" {
		blMsg := abstract.NewMessage("system:auth:token:check", identity.SystemContext(ctx), data.MustNewDocument(map[string]any{
			"token_id": claims.TokenID,
		}, identity.SystemContext(ctx)))
		blResult, err := o.internalDisp.Send(blMsg)
		if err != nil {
			o.opts.Logger.Warn("failed to check token blocklist", zap.Error(err))
		} else if blResult != nil && blResult.Document != nil {
			v, _ := blResult.Document.GetOr("blocklisted", false).(bool)
			if v {
				return nil, common.NewSystemError("UNAUTHORIZED", "token has been revoked")
			}
		} else {
		}
	} else {
	}
	return claims, nil
}

func extractClaims(doc *data.Document) *identity.Claims {
	if doc == nil {
		return &identity.Claims{}
	}
	userID, _ := doc.GetOr("user_id", "").(string)
	email, _ := doc.GetOr("email", "").(string)
	scopes, _ := doc.GetOr("scopes", []string{}).([]string)
	tokenType, _ := doc.GetOr("token_type", "").(string)
	tokenID, _ := doc.GetOr("token_id", "").(string)
	expiresAt, _ := doc.GetOr("expires_at", int64(0)).(int64)
	return &identity.Claims{
		UserID:    userID,
		Email:     email,
		Scopes:    scopes,
		TokenType: tokenType,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}
}

func (o *Interface) authenticateAPIKey(ctx context.Context, req Request, key string) (*identity.Claims, error) {
	sysCtx := identity.SystemContext(ctx)
	apiKeyMsg := abstract.NewMessage("system:auth:apikey:validate", sysCtx, data.MustNewDocument(map[string]any{
		"api_key": key,
	}, sysCtx))
	result, err := o.internalDisp.Send(apiKeyMsg)
	if err != nil {
		return nil, common.NewSystemError("UNAUTHORIZED", err.Error())
	}
	return extractClaims(result.Document), nil
}
