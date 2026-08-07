package users

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/users/schema"
)

func NewGetUserHandler(users *schema.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input schema.UserGetInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		user, err := users.GetByID(ctx, input.UserID)
		if err != nil {
			return nil, err
		}

		doc, err := user.Document()
		return &abstract.Result{Document: doc}, err
	}
}

func NewUpdateUserHandler(users *schema.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input schema.UserUpdateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		updated, err := users.UpdateUserProfile(ctx, input.UserID, &input.UserUpdate)
		if err != nil {
			return nil, err
		}

		doc, err := updated.Document()
		return &abstract.Result{Document: doc}, err
	}
}

func NewChangePasswordHandler(users *schema.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input schema.UserChangePasswordInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		if err := users.ChangePassword(ctx, input.UserID, input.Current, input.New); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewDeleteUserHandler(users *schema.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input schema.UserDeleteInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		if err := users.Delete(ctx, input.UserID); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}
