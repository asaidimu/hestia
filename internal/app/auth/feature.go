package auth

import (
	"time"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/abstract"
)

type Dependencies struct {
	UserModel           *users.UserModel
	CredentialsProvider abstract.CredentialsProvider
	APIKeyAuth          *APIKeyAuthenticator
	AdminUserID         string
	SessionTTL          time.Duration
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:auth:session:create", Handler: NewCreateSessionHandler(deps.UserModel, deps.CredentialsProvider, deps.SessionTTL), Description: "Authenticate and receive a session token", Enabled: true, Intent: registration.Create, BootstrapSafe: true, Input: core.Input{Schema: loginInputSchema(), Payload: definition.FieldTypeObject}, Output: loginOutputSchema()},
		{Name: "system:auth:user:register", Handler: NewRegisterHandler(deps.UserModel), Description: "Register a new user", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: registerInputSchema(), Payload: definition.FieldTypeObject}, Output: userOutputSchema()},
		{Name: "system:auth:session:delete", Handler: NewDeleteSessionHandler(), Description: "Logout", Enabled: true, Intent: registration.Delete, BootstrapSafe: true, Input: core.Input{Payload: definition.FieldTypeObject}},
		{Name: "system:auth:password:reset", Handler: NewPasswordResetHandler(deps.UserModel, deps.CredentialsProvider), Description: "Request password reset email", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: passwordResetInputSchema(), Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:auth:password:confirm", Handler: NewPasswordConfirmHandler(deps.UserModel, deps.CredentialsProvider), Description: "Confirm password reset with token", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: passwordConfirmInputSchema(), Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:auth:session:validate", Handler: NewValidateSessionHandler(deps.CredentialsProvider), Description: "Validate a session token", Enabled: true, Internal: true, Intent: registration.Read, Output: claimsOutputSchema()},
		{Name: "system:auth:apikey:validate", Handler: NewValidateAPIKeyHandler(deps.APIKeyAuth), Description: "Validate an API key", Enabled: true, Internal: true, Intent: registration.Read, Output: claimsOutputSchema()},
		{Name: "system:auth:bootstrap:password:set", Handler: NewSetBootstrapPasswordHandler(deps.UserModel, deps.AdminUserID), Description: "Set bootstrap admin password", Enabled: true, Intent: registration.Update, BootstrapSafe: true, Input: core.Input{Schema: bootstrapPasswordInputSchema(), Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
	}
}
