package audit

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/collections"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	Persist persistence.Persistence
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:audit:log:query", Handler: collections.NewNamedCollectionQueryHandler("_audit_log_", deps.Persist), Description: "Query audit logs", Enabled: true, Intent: abstract.Query, Input: runtime.Input{Schema: LogQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: LogQueryOutputSchema()},
		{Name: "system:audit:log:export", Handler: collections.NewNamedCollectionQueryHandler("_audit_log_", deps.Persist), Description: "Export audit logs", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: LogQueryInputSchema(), Payload: definition.FieldTypeRecord}, Output: LogQueryOutputSchema()},
		{Name: "system:audit:log:stream", Handler: logStreamHandler(deps.Persist), Description: "Stream audit log entries in real-time", Enabled: true, Intent: abstract.Stream, Input: runtime.Input{Schema: LogStreamInputSchema()}, Output: LogStreamOutputSchema()},
	}
}
