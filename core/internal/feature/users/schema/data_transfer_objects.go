package schema

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// UserGetInput binds the dispatch input document's arguments into a user ID.
type UserGetInput struct {
	UserID string `input:"arguments.user_id"`
}

// UserUpdateInput embeds the UserUpdate projection (carrying the
// input:"payload.*" tags) plus the dispatch-layer user ID binding.
type UserUpdateInput struct {
	UserUpdate
	UserID       string `input:"arguments.user_id"`
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

// UserQueryInput describes the query modifier inputs for listing users.
type UserQueryInput struct {
	Username string `anansi:"payload.username,omitempty"`
	Limit    int    `anansi:"payload.limit,omitempty"`
	Cursor   string `anansi:"payload.cursor,omitempty"`
}

// UserOutput is the single-user response contract, exposing the password-free
// UserPublic projection.
type UserOutput struct {
	Document UserPublic `anansi:"data"`
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

func UserQueryInputSchema() *definition.Schema  { return dispatch.SchemaFromType[UserQueryInput]() }
func UserQueryOutputSchema() *definition.Schema { return dispatch.SchemaFromType[UserQueryOutput]() }
func UserOutputSchema() *definition.Schema      { return dispatch.SchemaFromType[UserOutput]() }
func UserUpdateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserUpdateInput]("input", true)
}
func UserGetInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserGetInput]("input")
}
func UserChangePasswordInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserChangePasswordInput]("input")
}
func UserDeleteInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[UserDeleteInput]("input")
}
func MessageOutputSchema() *definition.Schema { return dispatch.SchemaFromType[MessageOutput]() }
