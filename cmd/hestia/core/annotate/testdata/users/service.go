package users

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
)

// CreateUserInput is the input document for CreateUser.
type CreateUserInput struct {
	ID   string `input:"arguments.id"`
	Name string `input:"name"`
}

// User is the output document for the users service.
type User struct {
	ID   string
	Name string
}

// CreateUser creates a new user.
//
// @hestia.register(
//   name="system:users:user:create",
//   intent="create",
//   rule="administrator",
//   description="Create a new user",
//   resource_id="id",
// )
func (s *UsersService) CreateUser(ctx context.Context, msg abstract.Message, input *CreateUserInput) (*User, error) {
	return nil, nil
}

// DeleteUserInput binds the resource ID.
type DeleteUserInput struct {
	UserID string `input:"arguments.user_id"`
}

// DeleteUser removes a user by ID.
//
// @hestia.register(
//   name="system:users:user:delete",
//   intent="delete",
//   rule="administrator",
//   resource_id="user_id",
// )
func (s *UsersService) DeleteUser(ctx context.Context, msg abstract.Message, input *DeleteUserInput) error {
	return nil
}

// CheckUser has no input document at all.
//
// @hestia.register(
//   name="system:users:user:check",
//   intent="check",
//   rule="authenticated",
// )
func (s *UsersService) CheckUser(ctx context.Context, msg abstract.Message) error {
	return nil
}

// ListUsersInput has no arguments tag → ResourceIDField stays empty.
type ListUsersInput struct {
	Cursor string `input:"payload.cursor"`
}

// ListUsers returns many users.
//
// @hestia.register(
//   name="system:collections:user:query",
//   intent="query",
//   rule="authenticated",
// )
func (s *UsersService) ListUsers(ctx context.Context, msg abstract.Message, input *ListUsersInput) ([]*User, error) {
	return nil, nil
}

// AckCallbackInput binds the callback payload.
type AckCallbackInput struct {
	Ref string `input:"payload.ref"`
}

// AckCallback registers a fire-and-forget empty result.
//
// @hestia.register(
//   name="system:users:callback:ack",
//   intent="create",
//   rule="authenticated",
//   description="Acknowledge a callback without waiting",
//   fire_and_forget="true",
// )
func (s *UsersService) AckCallback(ctx context.Context, msg abstract.Message, input *AckCallbackInput) error {
	return nil
}