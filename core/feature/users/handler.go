package users

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/users/model"
)

func NewCreateUserHandler(users *model.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.UserRegisterInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		tenantID := ""
		if input.TenantID != nil {
			tenantID = *input.TenantID
		}

		// TODO: This is absurd
		created, err := users.Register(ctx, input.Email, input.Password, input.Name, tenantID, input.Data, input.Permissions...)
		if err != nil {
			return nil, err
		}

		doc, err := created.Document()
		return &abstract.Result{Document: doc}, err
	}
}

func NewGetUserHandler(users *model.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.UserGetInput
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

func NewUpdateUserHandler(users *model.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.UserUpdateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		updated, err := users.UpdateUserProfile(ctx, input.ID, &input.UserUpdate)
		if err != nil {
			return nil, err
		}

		doc, err := updated.Document()
		return &abstract.Result{Document: doc}, err
	}
}

func NewChangePasswordHandler(users *model.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.UserChangePasswordInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		if err := users.ChangePassword(ctx, input.UserID, input.Current, input.New); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewDeleteUserHandler(users *model.SystemUsers) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.UserDeleteInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		if err := users.Delete(ctx, input.UserID); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}
