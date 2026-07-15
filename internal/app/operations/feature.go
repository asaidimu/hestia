package operations

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/app/audit"
	corepkg "github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Dependencies struct {
	Logger        *zap.Logger
	Disp          *corepkg.LocalDispatcher
	Bootstrapped  func() bool
	OnBootstrap   func()
	OnReset       func()
	AccessLog     *audit.AccessLogModel
	Persist       persistence.Persistence
	Registrations *[]abstract.MessageRegistration
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:core:health:check", Handler: NewSystemStatusHandler(deps.Bootstrapped), Description: "Health check", Enabled: true, Intent: registration.Read, BootstrapSafe: true, Output: healthOutputSchema()},
		{Name: "system:core:capability:list", Handler: corepkg.NewListCapabilitiesHandler(deps.Disp), Description: "List all registered handlers", Enabled: true, Intent: registration.Read, Output: capabilitiesOutputSchema()},
		{Name: "system:core:capability:set", Handler: corepkg.NewSetCapabilityEnabledHandler(deps.Disp), Description: "Enable or disable a handler", Enabled: true, Intent: registration.Update, Input: corepkg.Input{Schema: capabilityNameInputSchema()}, Output: messageOutputSchema()},
		{Name: "system:core:audit:log", Handler: NewLogAccessHandler(deps.AccessLog), Description: "Record an access log entry", Enabled: true, Internal: true, Intent: registration.Create},
		{Name: "system:core:docs:list", Handler: NewDocumentationHandler(deps.Registrations), Description: "Endpoint documentation", Enabled: true, Intent: registration.Read, BootstrapSafe: true, Output: documentationOutputSchema()},
		{Name: "system:core:bootstrap:mark", Handler: NewMarkBootstrappedHandler(deps.OnBootstrap), Description: "Mark system as bootstrapped", Enabled: true, Internal: true, Intent: registration.Create, Output: messageOutputSchema()},
		{Name: "system:core:reset", Handler: NewResetHandler(deps.OnReset), Description: "Reset system to initial state", Enabled: true, Intent: registration.Read, Output: messageOutputSchema()},
	}
}
