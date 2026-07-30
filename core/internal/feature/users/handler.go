package users

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/util"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

func NewGetUserHandler(users *UserModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input UserGetInput
		if err := msg.Input().BindTo(&input); err != nil {
			return nil, err
		}
		d, err := users.GetByID(ctx, input.UserID)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: d}, nil
	}
}

func NewUpdateUserHandler(users *UserModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input UserUpdateInput
		if err := msg.Input().BindTo(&input); err != nil {
			return nil, err
		}

		patch, err := input.Payload.Patch()
		if err != nil {
			return nil, err
		}
		fields := patch.ToMap()

		disabledChanged := false
		if disabledRaw, err := msg.Input().Get("payload.disabled"); err == nil {
			disabledFloat, _ := disabledRaw.(float64)
			if disabledFloat != 0 {
				fields["disabled"] = time.Now().Unix()
			} else {
				fields["disabled"] = int64(0)
			}
			disabledChanged = true
		}

		if len(fields) == 0 {
			return nil, fmt.Errorf("no fields to update")
		}

		if err := users.Update(ctx, input.UserID, fields); err != nil {
			return nil, err
		}
		if disabledChanged {
			if err := users.IncrementTokenVersion(ctx, input.UserID); err != nil {
				return nil, err
			}
		}

		d, err := users.GetByID(ctx, input.UserID)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: d}, nil
	}
}

func NewChangePasswordHandler(users *UserModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input UserChangePasswordInput
		if err := msg.Input().BindTo(&input); err != nil {
			return nil, err
		}

		d, err := users.GetByID(ctx, input.UserID)
		if err != nil {
			return nil, runtime.ErrNotFound.WithOperation("change_password")
		}

		disabled, _ := d.GetInt("disabled")
		if disabled >= 0 {
			return nil, runtime.ErrUserDisabled.WithOperation("change_password")
		}

		storedPassword, err := d.GetString("password")
		if err != nil {
			return nil, fmt.Errorf("invalid user data")
		}

		if !runtime.CheckPassword(input.Current, storedPassword) {
			return nil, runtime.ErrInvalidCredentials.WithOperation("change_password")
		}

		if err := users.ChangePassword(ctx, input.UserID, input.New); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewDeleteUserHandler(users *UserModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input UserDeleteInput
		if err := msg.Input().BindTo(&input); err != nil {
			return nil, err
		}
		if err := users.Delete(ctx, input.UserID); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewUserCreateDocumentHandler(users *UserModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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

		data, _ := body["data"].(map[string]any)
		d, err := users.Register(ctx, req.Email, req.Password, req.Name, runtimecontext.GetTenantID(ctx), data)
		if err != nil {
			return nil, err
		}

		return &abstract.Result{Document: d}, nil
	}
}

func NewUserUpdateDocumentHandler(users *UserModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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
			Name        *string        `json:"name,omitempty"`
			Email       *string        `json:"email,omitempty"`
			Permissions []string       `json:"permissions,omitempty"`
			Verified    *bool          `json:"verified,omitempty"`
			Data        map[string]any `json:"data,omitempty"`
		}
		b, _ := json.Marshal(body)
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, common.NewSystemError("PARSE_DOCUMENT", fmt.Sprintf("invalid JSON: %s", err.Error()))
		}

		fields := util.StructToMap(req)
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

		return &abstract.Result{Document: d}, nil
	}
}
