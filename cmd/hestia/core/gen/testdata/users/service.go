package users

import (
        "context"

        "github.com/asaidimu/go-anansi/v8/core/document"

        "github.com/asaidimu/hestia/core/abstract"
        dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// CreateUserInput binds the create request. Exactly one arguments tag → the
// resource ID field.
type CreateUserInput struct {
        ID   string `input:"arguments.id"`
        Name string `input:"name"`
}

// User is the model output; embedding document.DocumentModel satisfies
// dispatch.HandleDocument's documentModel constraint.
type User struct {
        document.DocumentModel
        ID   string
        Name string
}

// UsersService is the service under test.
type UsersService struct{}

func NewUsersService(rt abstract.Container) (*UsersService, error) {
        return &UsersService{}, nil
}

// CreateUser registers a document result.
//
// @hestia.register(
//   name="system:users:user:create",
//   intent="create",
//   rule="administrator",
//   description="Create a new user",
// )
func (s *UsersService) CreateUser(ctx context.Context, msg abstract.Message, input *CreateUserInput) (*User, error) {
        return nil, nil
}

// DeleteUserInput binds the resource id.
type DeleteUserInput struct {
        UserID string `input:"arguments.user_id"`
}

// DeleteUser registers an empty result.
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

// CheckUser registers a no-input empty result (inline closure).
//
// @hestia.register(
//   name="system:users:user:check",
//   intent="check",
//   rule="authenticated",
// )
func (s *UsersService) CheckUser(ctx context.Context, msg abstract.Message) error {
        return nil
}

// ListUsersInput has no arguments tag.
type ListUsersInput struct {
        Cursor string `input:"payload.cursor"`
}

// ListUsers registers a documents result. HandleDocuments expects the handler
// to map models to *document.Document itself.
//
// @hestia.register(
//   name="system:collections:user:query",
//   intent="query",
//   rule="authenticated",
// )
func (s *UsersService) ListUsers(ctx context.Context, msg abstract.Message, input *ListUsersInput) ([]*document.Document, error) {
        return nil, nil
}

// AckCallbackInput binds the callback payload.
type AckCallbackInput struct {
        Ref string `input:"payload.ref"`
}

// AckCallback registers a fire-and-forget empty result: transports accept
// the message and respond immediately without waiting for completion
// (HTTP answers 202 Accepted with the message ID).
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

// ImportUserInput is one streamed item: the NDJSON payload binds via the
// "input" tag exactly like a single-document input.
type ImportUserInput struct {
        Name string `input:"name"`
}

// ImportUsers streams incoming user records (input streaming) and returns an
// import summary. The streaming signature (<-chan dispatch.Item[TIn]) makes
// the generator emit HandleInputStream and Input.Streaming = true.
//
// @hestia.register(
//   name="system:users:user:import",
//   intent="create",
//   rule="administrator",
//   description="Bulk-import users from an NDJSON stream",
// )
func (s *UsersService) ImportUsers(ctx context.Context, msg abstract.Message, items <-chan dispatch.Item[ImportUserInput]) (*abstract.Result, error) {
        return nil, nil
}

// Ensure dispatch import is exercised (compile proof for stream detection).
var _ = dispatch.HandleEmpty[struct{}](func(context.Context, abstract.Message, *struct{}) error { return nil })
