package auth

import (
	"time"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/feature/apikeys/model"
	"github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	UserModel           *model.SystemUsers
	CredentialsProvider abstract.CredentialsProvider
	APIKeyAuth          *APIKeyAuthenticator
	APIKeyModel         *apikeysmodel.SystemAPIKeys
	AdminUserID         string
	SessionTTL          time.Duration
	Notifier            abstract.Notifier
	AppURL              string
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:auth:session:create", Handler: NewCreateSessionHandler(deps.UserModel, deps.CredentialsProvider, deps.SessionTTL), Description: "Authenticate and receive a session token", Enabled: true, Intent: abstract.Create, BootstrapSafe: true, Input: runtime.Input{Schema: LoginInputSchema(), Payload: definition.FieldTypeObject}, Output: loginOutputSchema()},
		{Name: "system:auth:session:delete", Handler: NewDeleteSessionHandler(), Description: "Logout", Enabled: true, Intent: abstract.Delete, BootstrapSafe: true, Input: runtime.Input{Schema: DeleteSessionInputSchema(), Payload: definition.FieldTypeObject}},
		{Name: "system:auth:password:reset", Handler: NewPasswordResetHandler(deps.UserModel, deps.CredentialsProvider, deps.Notifier, deps.AppURL), Description: "Request password reset email", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: PasswordResetInputSchema(), Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:auth:password:confirm", Handler: NewPasswordConfirmHandler(deps.UserModel, deps.CredentialsProvider), Description: "Confirm password reset with token", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: PasswordConfirmInputSchema(), Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:auth:session:validate", Handler: NewValidateSessionHandler(deps.CredentialsProvider), Description: "Validate a session token", Enabled: true, Internal: true, Intent: abstract.Read, Output: claimsOutputSchema()},
		{Name: "system:auth:apikey:validate", Handler: NewValidateAPIKeyHandler(deps.APIKeyAuth), Description: "Validate an API key", Enabled: true, Internal: true, Intent: abstract.Read, Output: claimsOutputSchema()},
		{Name: "system:auth:bootstrap:password:set", Handler: NewSetBootstrapPasswordHandler(deps.UserModel, deps.AdminUserID), Description: "Set bootstrap admin password", Enabled: true, Intent: abstract.Update, BootstrapSafe: true, Input: runtime.Input{Schema: BootstrapPasswordInputSchema(), Payload: definition.FieldTypeObject}, Output: messageOutputSchema()},
		{Name: "system:auth:token:elevate", Handler: NewElevateTokenHandler(deps.UserModel, deps.APIKeyModel), Description: "Issue an ephemeral API key for privilege elevation", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: ElevateInputSchema(), Payload: definition.FieldTypeObject}, Output: elevateOutputSchema()},
	}
}
