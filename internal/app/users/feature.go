package users

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/internal/app/collections"
	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Dependencies struct {
	UserModel *UserModel
	Persist   persistence.Persistence
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:users:user:query", Handler: collections.NewNamedCollectionQueryHandler("_user_", deps.Persist), Description: "Query users collection", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: userQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: userQueryOutputSchema()},
		{Name: "system:users:user:get", Handler: NewGetUserHandler(deps.UserModel), Description: "Get user by ID", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: userGetInputSchema(),
			Arguments: map[string]definition.FieldType{"user_id": definition.FieldTypeString},
		}, Output: userOutputSchema()},
		{Name: "system:users:user:update", Handler: NewUpdateUserHandler(deps.UserModel), Description: "Update user", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: userUpdateInputSchema(), Arguments: map[string]definition.FieldType{"user_id": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: userOutputSchema()},
		{Name: "system:users:password:change", Handler: NewChangePasswordHandler(deps.UserModel), Description: "Change user password", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: userChangePasswordInputSchema(), Arguments: map[string]definition.FieldType{"user_id": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:users:user:delete", Handler: NewDeleteUserHandler(deps.UserModel), Description: "Delete user", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: userDeleteInputSchema(), Arguments: map[string]definition.FieldType{"user_id": definition.FieldTypeString}, Modifiers: map[string]definition.FieldType{"permanent": definition.FieldTypeString}}},
	}
}
