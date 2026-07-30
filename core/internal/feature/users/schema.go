package users

import (
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/internal/feature/users/schema"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type UserGetInput struct {
	data.DocumentModel
	UserID string `anansi:"arguments.user_id"`
}

type UserUpdateInput struct {
	data.DocumentModel
	UserID  string            `anansi:"arguments.user_id"`
	Payload schema.SystemUser `anansi:"payload"`
}

type UserChangePasswordInput struct {
	data.DocumentModel
	UserID  string `anansi:"arguments.user_id"`
	Current string `anansi:"payload.current"`
	New     string `anansi:"payload.new"`
}

type UserDeleteInput struct {
	data.DocumentModel
	UserID string `anansi:"arguments.user_id"`
}

type UserQueryInput struct {
	data.DocumentModel
	Username string `anansi:"payload.username,omitempty"`
	Limit    int    `anansi:"payload.limit,omitempty"`
	Cursor   string `anansi:"payload.cursor,omitempty"`
}

type UserOutput struct {
	data.DocumentModel
	Document schema.SystemUser `anansi:"document"`
}

type UserQueryOutput struct {
	data.DocumentModel
	Documents []schema.SystemUser `anansi:"page.documents"`
	Total     int                 `anansi:"page.pagination.total"`
	Cursor    string              `anansi:"page.pagination.cursor"`
	Limit     int                 `anansi:"page.pagination.limit"`
}

type MessageOutput struct {
	data.DocumentModel
	Message string `anansi:"message"`
}

func userQueryInputSchema() *definition.Schema              { return dispatch.SchemaFromType[UserQueryInput]() }
func userQueryOutputSchema() *definition.Schema             { return dispatch.SchemaFromType[UserQueryOutput]() }
func userOutputSchema() *definition.Schema                  { return dispatch.SchemaFromType[UserOutput]() }
func userUpdateInputSchema() *definition.Schema             { return dispatch.SchemaFromType[UserUpdateInput]() }
func userGetInputSchema() *definition.Schema                { return dispatch.SchemaFromType[UserGetInput]() }
func userChangePasswordInputSchema() *definition.Schema     { return dispatch.SchemaFromType[UserChangePasswordInput]() }
func userDeleteInputSchema() *definition.Schema             { return dispatch.SchemaFromType[UserDeleteInput]() }
func messageOutputSchema() *definition.Schema               { return dispatch.SchemaFromType[MessageOutput]() }
