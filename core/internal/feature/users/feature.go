package users

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/internal/feature/users/schema"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	UserModel *schema.SystemUsers
	Persist   persistence.Persistence
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:users:user:query", Handler: collections.NewNamedCollectionQueryHandler("_user_", deps.Persist), Description: "Query users collection", Enabled: true, Intent: abstract.Query, Input: runtime.Input{Schema: schema.UserQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: schema.UserQueryOutputSchema()},
		{Name: "system:users:user:get", Handler: NewGetUserHandler(deps.UserModel), Description: "Get user by ID", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Schema: schema.UserGetInputSchema(),
			Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id",
		}, Output: schema.UserOutputSchema()},
		{Name: "system:users:user:update", Handler: NewUpdateUserHandler(deps.UserModel), Description: "Update user", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: schema.UserUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id", Payload: definition.FieldTypeObject}, Output: schema.UserOutputSchema()},
		{Name: "system:users:password:change", Handler: NewChangePasswordHandler(deps.UserModel), Description: "Change user password", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: schema.UserChangePasswordInputSchema(), Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id", Payload: definition.FieldTypeObject}, Output: schema.MessageOutputSchema()},
		{Name: "system:users:user:delete", Handler: NewDeleteUserHandler(deps.UserModel), Description: "Delete user", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Schema: schema.UserDeleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}}, ResourceIDField: "user_id"}},
	}
}
