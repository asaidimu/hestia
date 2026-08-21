// @note #cruft-20260821-016 issue status=open priority=P2 tags=#cruft,#dead-code : Stale schema functions in operations/outputs.go
// @see #8uuufn
//
// The schema functions (healthOutputSchema, documentationOutputSchema,
// capabilitiesOutputSchema, messageOutputSchema, schedulerListOutputSchema)
// are dead code. The generated registrations use dispatch.SchemaFromType
// directly.
//
// Resolution: remove the schema functions. The output types themselves are
// still used by the service methods and registrations.
package operations

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	corepkg "github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// HealthView is the wire shape of a health check response.
type HealthView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Ok                     bool `anansi:"ok"`
	Bootstrapped           bool `anansi:"bootstrapped"`
}

// HealthOutput is the envelope declaring the health check schema.
type HealthOutput struct {
	Document HealthView `anansi:"document"`
}

func healthOutputSchema() *definition.Schema { return dispatch.SchemaFromType[HealthOutput]() }

// HTTPMapping is the wire shape of an endpoint's HTTP method and route.
type HTTPMapping struct {
	Method  string `anansi:"method"`
	Route   string `anansi:"route"`
	Pattern string `anansi:"pattern"`
}

// EndpointDoc is the wire shape of one documented endpoint.
type EndpointDoc struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string         `anansi:"name"`
	Description            string         `anansi:"description"`
	Enabled                bool           `anansi:"enabled"`
	Intent                 string         `anansi:"intent"`
	BootstrapSafe          bool           `anansi:"bootstrap_safe"`
	Internal               bool           `anansi:"internal"`
	HTTP                   HTTPMapping    `anansi:"http"`
	Input                  map[string]any `anansi:"input,omitempty"`
	Output                 map[string]any `anansi:"output,omitempty"`
}

// DocumentationOutput is the envelope declaring the endpoint documentation
// list schema. The documents array keeps the top-level documents marker that
// serializeOutputField requires for list responses.
type DocumentationOutput struct {
	Documents []EndpointDoc `anansi:"documents"`
}

func documentationOutputSchema() *definition.Schema { return dispatch.SchemaFromType[DocumentationOutput]() }

// CapabilitiesOutput is the envelope declaring the capabilities schema.
type CapabilitiesOutput struct {
	Document corepkg.CapabilitiesDocument `anansi:"document"`
}

func capabilitiesOutputSchema() *definition.Schema { return dispatch.SchemaFromType[CapabilitiesOutput]() }

// MessageOutput declares the schema of a plain status message response.
type MessageOutput struct {
	Message string `anansi:"message"`
}

func messageOutputSchema() *definition.Schema { return dispatch.SchemaFromType[MessageOutput]() }

// SchedulerJobInfo is the wire shape of a registered scheduler job.
type SchedulerJobInfo struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string   `anansi:"name"`
	Expr                   string   `anansi:"expr"`
	Next                   string   `anansi:"next"`
	Prev                   string   `anansi:"prev"`
	Paused                 bool     `anansi:"paused"`
	Tags                   []string `anansi:"tags"`
}

// SchedulerListOutput is the envelope declaring the scheduler job list schema.
type SchedulerListOutput struct {
	Documents []SchedulerJobInfo `anansi:"documents"`
}

func schedulerListOutputSchema() *definition.Schema { return dispatch.SchemaFromType[SchedulerListOutput]() }
