package audit

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
	"github.com/asaidimu/hestia/core/abstract"
)

type Dependencies struct {
	Persist persistence.Persistence
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:audit:log:query", Handler: collections.NewNamedCollectionQueryHandler("_audit_log_", deps.Persist), Description: "Query audit logs", Enabled: true, Intent: registration.Query, Input: runtime.Input{Schema: logQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: logQueryOutputSchema()},
		{Name: "system:audit:log:export", Handler: collections.NewNamedCollectionQueryHandler("_audit_log_", deps.Persist), Description: "Export audit logs", Enabled: true, Intent: registration.Update, Input: runtime.Input{Schema: logQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: logQueryOutputSchema()},
		{Name: "system:audit:log:stream", Handler: logStreamHandler(deps.Persist), Description: "Stream audit log entries in real-time", Enabled: true, Intent: registration.Stream, Input: runtime.Input{Schema: logStreamInputSchema()}, Output: logStreamOutputSchema()},
	}
}
