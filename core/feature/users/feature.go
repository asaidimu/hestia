package users

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/collections"
	"github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	UserModel *model.SystemUsers
	Persist   persistence.Persistence
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:users:user:create", Handler: NewCreateUserHandler(deps.UserModel), Description: "Create a new user", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: model.UserRegisterInputSchema(), Payload: definition.FieldTypeObject}, Output: model.UserOutputSchema()},
		{Name: "system:users:user:query", Handler: collections.NewNamedCollectionQueryHandler("_user_", deps.Persist), Description: "Query users collection", Enabled: true, Intent: abstract.Query, Input: runtime.Input{Schema: model.UserQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: model.UserQueryOutputSchema()},
		{Name: "system:users:user:get", Handler: NewGetUserHandler(deps.UserModel), Description: "Get user by ID", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Schema: model.UserGetInputSchema(),
			Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id",
		}, Output: model.UserOutputSchema()},
		{Name: "system:users:user:update", Handler: NewUpdateUserHandler(deps.UserModel), Description: "Update user", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: model.UserUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id", Payload: definition.FieldTypeObject}, Output: model.UserOutputSchema()},
		{Name: "system:users:password:change", Handler: NewChangePasswordHandler(deps.UserModel), Description: "Change user password", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: model.UserChangePasswordInputSchema(), Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id", Payload: definition.FieldTypeObject}, Output: model.MessageOutputSchema()},
		{Name: "system:users:user:delete", Handler: NewDeleteUserHandler(deps.UserModel), Description: "Delete user", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Schema: model.UserDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id"}},
	}
}
