package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// UserGetInput binds the dispatch input document's arguments into a user ID.
type UserGetInput struct {
	UserID string `input:"arguments.user_id"`
}

// UserChangePasswordInput binds the dispatch input document for a password
// change: the target user ID plus the current and new passwords.
type UserChangePasswordInput struct {
	UserID  string `input:"arguments.user_id"`
	Current string `input:"payload.current"`
	New     string `input:"payload.new"`
}

// UserDeleteInput binds the dispatch input document's arguments into a user ID.
type UserDeleteInput struct {
	UserID string `input:"arguments.user_id"`
}

// UserUpdateInput binds the dispatch input document's arguments and payload
// into the UserUpdate projection. The projection's _id_ field carries the
// input:"arguments.user_id" tag, so the resource ID lands in UserUpdate.ID.
type UserUpdateInput struct {
	UserUpdate
}

// UserRegisterInput binds the dispatch input document's payload into the
// UserRegister projection. The projection carries input:"payload.*" tags, so
// the registration fields are inherited flat and land under payload.
type UserRegisterInput struct {
	UserRegister
}

// UserQueryInput describes the query modifier inputs for listing users.
type UserQueryInput struct {
	Name     string `input:"arguments.name"`
	Username string `input:"payload.username,omitempty"`
	Limit    int    `input:"payload.limit,omitempty"`
	Cursor   string `input:"payload.cursor,omitempty"`
}

// UserOutput is the single-user response contract, exposing the password-free
// UserPublic projection.
type UserOutput struct {
	Document UserPublic `anansi:"document"`
}

// UserQueryOutput is the paginated user list response contract.
type UserQueryOutput struct {
	Documents []UserPublic `anansi:"page.documents"`
	Total     int          `anansi:"page.pagination.total"`
	Cursor    string       `anansi:"page.pagination.cursor"`
	Limit     int          `anansi:"page.pagination.limit"`
}

// MessageOutput is a generic message response contract.
type MessageOutput struct {
	Message string `anansi:"message"`
}

func UserQueryInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserQueryInput]("input")
}
func UserQueryOutputSchema() *definition.Schema { return dispatch.SchemaFromType[UserQueryOutput]() }
func UserOutputSchema() *definition.Schema      { return dispatch.SchemaFromType[UserOutput]() }
func UserUpdateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserUpdateInput]("input", true)
}
func UserGetInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserGetInput]("input")
}
func UserRegisterInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserRegisterInput]("input", true)
}
func UserChangePasswordInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserChangePasswordInput]("input")
}
func UserDeleteInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserDeleteInput]("input")
}
func MessageOutputSchema() *definition.Schema { return dispatch.SchemaFromType[MessageOutput]() }
