package audit

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Dependencies struct {
	Persist persistence.Persistence
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:audit:log:query", Handler: logQueryHandler(deps.Persist, 100), Description: "Query access logs", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: logQueryInputSchema()}, Output: logEntryOutputSchema()},
		{Name: "system:audit:log:export", Handler: logQueryHandler(deps.Persist, 5000), Description: "Export access logs", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: logQueryInputSchema()}, Output: logEntryOutputSchema()},
		{Name: "system:audit:log:stream", Handler: logStreamHandler(deps.Persist), Description: "Stream access log entries in real-time", Enabled: true, Intent: registration.Stream, Input: core.Input{Schema: logStreamInputSchema()}, Output: logStreamOutputSchema()},
	}
}
