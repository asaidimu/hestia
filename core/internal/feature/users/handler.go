package users

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
)

func NewGetUserHandler(users *UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID, _ := doc.GetOr("arguments.user_id", "").(string)

		d, err := users.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &registration.Result{Document: d}, nil
	}
}

func NewUpdateUserHandler(users *UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID, _ := doc.GetOr("arguments.user_id", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)

		fields := map[string]any{}
		if v, exists := body["name"]; exists {
			fields["name"], _ = v.(string)
		}
		if v, exists := body["email"]; exists {
			fields["email"], _ = v.(string)
		}
		if v, exists := body["permissions"]; exists {
			switch arr := v.(type) {
			case []string:
				fields["permissions"] = arr
			case []any:
				perms := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						perms = append(perms, s)
					}
				}
				fields["permissions"] = perms
			}
		}
		if v, exists := body["verified"]; exists {
			fields["verified"], _ = v.(bool)
		}
		if len(fields) == 0 {
			return nil, fmt.Errorf("no fields to update")
		}

		if err := users.Update(ctx, userID, fields); err != nil {
			return nil, err
		}

		d, err := users.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &registration.Result{Document: d}, nil
	}
}

func NewChangePasswordHandler(users *UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID, _ := doc.GetOr("arguments.user_id", "").(string)
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		currentPassword, _ := body["current"].(string)
		newPassword, _ := body["new"].(string)

		d, err := users.GetByID(ctx, userID)
		if err != nil {
			return nil, runtime.ErrNotFound.WithOperation("change_password")
		}

		if users.IsDeleted(d) {
			return nil, runtime.ErrUserDeleted.WithOperation("change_password")
		}

		storedPassword, err := d.GetString("password")
		if err != nil {
			return nil, fmt.Errorf("invalid user data")
		}

		if !runtime.CheckPassword(currentPassword, storedPassword) {
			return nil, runtime.ErrInvalidCredentials.WithOperation("change_password")
		}

		if err := users.ChangePassword(ctx, userID, newPassword); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewDeleteUserHandler(users *UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		userID, _ := doc.GetOr("arguments.user_id", "").(string)
		permanent, _ := doc.GetOr("modifiers.permanent", false).(bool)

		if permanent {
			if err := users.HardDelete(ctx, userID); err != nil {
				return nil, err
			}
		} else {
			if err := users.SoftDelete(ctx, userID); err != nil {
				return nil, err
			}
		}
		return &registration.Result{}, nil
	}
}

func NewUserCreateDocumentHandler(users *UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		bodyRaw := doc.GetOr("payload", nil)

		var body map[string]any
		if bodyRaw != nil {
			body, _ = bodyRaw.(map[string]any)
		}
		if len(body) == 0 {
			return nil, common.NewSystemError("DOCUMENT_REQUIRED", "request body must be a valid JSON document")
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
		}
		b, _ := json.Marshal(body)
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, common.NewSystemError("PARSE_DOCUMENT", fmt.Sprintf("invalid JSON: %s", err.Error()))
		}
		if req.Email == "" || req.Password == "" || req.Name == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "email, password, and name are required")
		}

		d, err := users.Register(ctx, req.Email, req.Password, req.Name)
		if err != nil {
			return nil, err
		}

		return &registration.Result{Document: d}, nil
	}
}

func NewUserUpdateDocumentHandler(users *UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		documentID, _ := doc.GetOr("arguments.document_id", "").(string)
		bodyRaw := doc.GetOr("payload", nil)

		var body map[string]any
		if bodyRaw != nil {
			body, _ = bodyRaw.(map[string]any)
		}
		if len(body) == 0 {
			return nil, common.NewSystemError("DOCUMENT_REQUIRED", "request body must be a valid JSON document")
		}

		var req struct {
			Name        *string   `json:"name,omitempty"`
			Email       *string   `json:"email,omitempty"`
			Permissions []string  `json:"permissions,omitempty"`
			Verified    *bool     `json:"verified,omitempty"`
		}
		b, _ := json.Marshal(body)
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, common.NewSystemError("PARSE_DOCUMENT", fmt.Sprintf("invalid JSON: %s", err.Error()))
		}

		fields := map[string]any{}
		if req.Name != nil {
			fields["name"] = *req.Name
		}
		if req.Email != nil {
			fields["email"] = *req.Email
		}
		if req.Permissions != nil {
			fields["permissions"] = req.Permissions
		}
		if req.Verified != nil {
			fields["verified"] = *req.Verified
		}
		if len(fields) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "no fields to update")
		}

		if err := users.Update(ctx, documentID, fields); err != nil {
			return nil, err
		}

		d, err := users.GetByID(ctx, documentID)
		if err != nil {
			return nil, err
		}

		return &registration.Result{Document: d}, nil
	}
}
