package apikeys

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/core/identity"
)

func NewListAPIKeysHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID, _ := doc.GetOr("arguments.user_id", "").(string)
		if userID == "" {
			if claims, ok := identity.ClaimsFromContext(ctx); ok {
				userID = claims.UserID
			}
		}

		docs, err := keys.List(ctx, userID)
		if err != nil {
			return nil, err
		}

		return &registration.Result{Documents: docs}, nil
	}
}

func NewGetAPIKeyHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID, _ := doc.GetOr("arguments.user_id", "").(string)
		if userID == "" {
			if claims, ok := identity.ClaimsFromContext(ctx); ok {
				userID = claims.UserID
			}
		}
		keyID, _ := doc.GetOr("arguments.key_id", "").(string)

		d, err := keys.Get(ctx, keyID, userID)
		if err != nil {
			return nil, err
		}

		if prefix, err := d.GetString("prefix"); err == nil {
			var hint string
			if rawKey, err := d.GetString("key"); err == nil && len(rawKey) >= 4 {
				hint = rawKey[len(rawKey)-4:]
			}
			if hint != "" {
				d.Set("prefix", prefix+"..."+hint)
			}
		}

		return &registration.Result{Document: d}, nil
	}
}

func NewCreateAPIKeyHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID := ""
		if claims, ok := identity.ClaimsFromContext(ctx); ok {
			userID = claims.UserID
		}
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		name, _ := body["name"].(string)

		generated, err := keys.Generate()
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}

		req := &CreateKeyRequest{
			Name: name,
		}
		if v, ok := body["environment"]; ok {
			req.Environment, _ = v.(string)
		}
		if v, ok := body["operations"]; ok {
			switch arr := v.(type) {
			case []string:
				req.Operations = arr
			case []any:
				for _, item := range arr {
					if s, ok := item.(string); ok {
						req.Operations = append(req.Operations, s)
					}
				}
			}
		}
		if v, ok := body["expiry"]; ok {
			req.Expiry, _ = v.(string)
		}
		if v, ok := body["limits"]; ok {
			req.Limits, _ = v.(map[string]any)
		}
		if v, ok := body["ip"]; ok {
			req.IP, _ = v.(map[string]any)
		}

		d, err := keys.Create(ctx, generated, userID, req)
		if err != nil {
			return nil, fmt.Errorf("create key: %w", err)
		}

		sane, err := d.Sanitize()
		if err != nil {
			return nil, err
		}
		sane.Set("key", generated.FullKey)
		return &registration.Result{Document: sane}, nil
	}
}

func NewUpdateAPIKeyHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID := ""
		if claims, ok := identity.ClaimsFromContext(ctx); ok {
			userID = claims.UserID
		}
		keyID, _ := doc.GetOr("arguments.key_id", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)

		var req UpdateKeyRequest
		if v, exists := body["name"]; exists {
			s := v.(string)
			req.Name = &s
		}
		if v, exists := body["status"]; exists {
			s := v.(string)
			req.Status = &s
		}
		if v, exists := body["expiry"]; exists {
			s := v.(string)
			req.Expiry = &s
		}
		if v, exists := body["environment"]; exists {
			s := v.(string)
			req.Environment = &s
		}
		if v, exists := body["operations"]; exists {
			switch arr := v.(type) {
			case []string:
				req.Operations = arr
			case []any:
				for _, item := range arr {
					if s, ok := item.(string); ok {
						req.Operations = append(req.Operations, s)
					}
				}
			}
		}
		if v, exists := body["limits"]; exists {
			req.Limits, _ = v.(map[string]any)
		}
		if v, exists := body["ip"]; exists {
			req.IP, _ = v.(map[string]any)
		}

		d, err := keys.Update(ctx, keyID, userID, &req)
		if err != nil {
			return nil, err
		}
		return &registration.Result{Document: d}, nil
	}
}

func NewDeleteAPIKeyHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID := ""
		if claims, ok := identity.ClaimsFromContext(ctx); ok {
			userID = claims.UserID
		}
		keyID, _ := doc.GetOr("arguments.key_id", "").(string)

		if err := keys.Delete(ctx, keyID, userID); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewRotateAPIKeyHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID := ""
		if claims, ok := identity.ClaimsFromContext(ctx); ok {
			userID = claims.UserID
		}
		keyID, _ := doc.GetOr("arguments.key_id", "").(string)

		generated, d, err := keys.Rotate(ctx, keyID, userID)
		if err != nil {
			return nil, err
		}
		d.Set("key", generated.FullKey)
		return &registration.Result{Document: d}, nil
	}
}

func NewAPIKeyCreateDocumentHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		bodyRaw := doc.GetOr("payload", nil)

		var body map[string]any
		if bodyRaw != nil {
			body, _ = bodyRaw.(map[string]any)
		}
		if len(body) == 0 {
			return nil, common.NewSystemError("DOCUMENT_REQUIRED", "request body must be a valid JSON document")
		}

		claims, ok := identity.ClaimsFromContext(ctx)
		if !ok {
			return nil, common.NewSystemError("AUTH_REQUIRED", "authentication context missing")
		}

		var req CreateKeyRequest
		b, _ := json.Marshal(body)
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, common.NewSystemError("PARSE_DOCUMENT", fmt.Sprintf("invalid JSON: %s", err.Error()))
		}
		if req.Name == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "name is required")
		}

		generated, err := keys.Generate()
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}

		d, err := keys.Create(ctx, generated, claims.UserID, &req)
		if err != nil {
			return nil, fmt.Errorf("create key: %w", err)
		}

		d.Set("key", generated.FullKey)
		return &registration.Result{Document: d}, nil
	}
}

func NewAPIKeyUpdateDocumentHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		documentID, _ := doc.GetOr("document_id", "").(string)
		bodyRaw := doc.GetOr("body", nil)

		var body map[string]any
		if bodyRaw != nil {
			body, _ = bodyRaw.(map[string]any)
		}
		if len(body) == 0 {
			return nil, common.NewSystemError("DOCUMENT_REQUIRED", "request body must be a valid JSON document")
		}

		claims, ok := identity.ClaimsFromContext(ctx)
		if !ok {
			return nil, common.NewSystemError("AUTH_REQUIRED", "authentication context missing")
		}

		var req UpdateKeyRequest
		b, _ := json.Marshal(body)
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, common.NewSystemError("PARSE_DOCUMENT", fmt.Sprintf("invalid JSON: %s", err.Error()))
		}

		d, err := keys.Update(ctx, documentID, claims.UserID, &req)
		if err != nil {
			return nil, err
		}

		return &registration.Result{Document: d}, nil
	}
}

func NewAPIKeyDeleteDocumentHandler(keys *APIKeyModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		documentID, _ := doc.GetOr("document_id", "").(string)

		claims, ok := identity.ClaimsFromContext(ctx)
		if !ok {
			return nil, common.NewSystemError("AUTH_REQUIRED", "authentication context missing")
		}

		if err := keys.Delete(ctx, documentID, claims.UserID); err != nil {
			return nil, err
		}

		return &registration.Result{}, nil
	}
}
