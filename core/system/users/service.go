package users

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/users/model"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"
)

// UsersService is the service for the users domain. The model collection is
// initialized in the constructor by resolving persistence from the
// runtime.Runtime DI container; the struct is scaffolded once and then owned
// by the feature author.
type UsersService struct {
	model *model.SystemUsers
}

func NewUsersService(rt abstract.Container) (*UsersService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)

	m, err := model.InitSystemUsersModel(persist, logger)
	if err != nil {
		return nil, err
	}
	return &UsersService{model: m}, nil
}

// CreateUser registers a new user.
//
// @hestia.register(
//   name="system:users:user:create",
//   intent="create",
//   rule="administrator",
//   description="Create a new user",
// )
func (s *UsersService) CreateUser(ctx context.Context, msg abstract.Message, input *model.UserRegisterInput) (*model.UserPublic, error) {
	tenantID := ""
	if input.TenantID != nil {
		tenantID = *input.TenantID
	}
	user, err := s.model.Register(ctx, input.Email, input.Password, input.Name, tenantID, input.Data, input.Permissions...)
	if err != nil {
		return nil, err
	}
	return user.Public(), nil
}

// GetUser returns a user by ID.
//
// @hestia.register(
//   name="system:users:user:get",
//   intent="read",
//   rule="administrator",
//   description="Get user by ID",
//   resource_id="user_id",
// )
func (s *UsersService) GetUser(ctx context.Context, msg abstract.Message, input *model.UserGetInput) (*model.UserPublic, error) {
	user, err := s.model.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	return user.Public(), nil
}

// UpdateUser updates a user profile.
//
// @hestia.register(
//   name="system:users:user:update",
//   intent="update",
//   rule="administrator",
//   description="Update user profile",
//   resource_id="user_id",
// )
func (s *UsersService) UpdateUser(ctx context.Context, msg abstract.Message, input *model.UserUpdateInput) (*model.UserPublic, error) {
	user, err := s.model.UpdateUserProfile(ctx, input.ID, &input.UserUpdate)
	if err != nil {
		return nil, err
	}
	return user.Public(), nil
}

// ChangePassword changes a user's password.
//
// @hestia.register(
//   name="system:users:password:change",
//   intent="update",
//   rule="administrator",
//   description="Change account password",
//   resource_id="user_id",
// )
func (s *UsersService) ChangePassword(ctx context.Context, msg abstract.Message, input *model.UserChangePasswordInput) error {
	return s.model.ChangePassword(ctx, input.UserID, input.Current, input.New)
}

// DeleteUser deletes a user account.
//
// @hestia.register(
//   name="system:users:user:delete",
//   intent="delete",
//   rule="administrator",
//   description="Delete user account",
//   resource_id="user_id",
// )
func (s *UsersService) DeleteUser(ctx context.Context, msg abstract.Message, input *model.UserDeleteInput) error {
	return s.model.Delete(ctx, input.UserID)
}
