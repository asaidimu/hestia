package operations

import (
	"github.com/asaidimu/go-anansi/v8/core/document"

	corepkg "github.com/asaidimu/hestia/core/runtime"
)

type HealthView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Ok                     bool `anansi:"ok"`
	Bootstrapped           bool `anansi:"bootstrapped"`
}

type HealthOutput struct {
	Document HealthView `anansi:"document"`
}

type HTTPMapping struct {
	Method  string `anansi:"method"`
	Route   string `anansi:"route"`
	Pattern string `anansi:"pattern"`
}

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

type DocumentationOutput struct {
	Documents []EndpointDoc `anansi:"documents"`
}

type CapabilitiesOutput struct {
	Document corepkg.CapabilitiesDocument `anansi:"document"`
}

type MessageOutput struct {
	Message string `anansi:"message"`
}

type SchedulerJobInfo struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string   `anansi:"name"`
	Expr                   string   `anansi:"expr"`
	Next                   string   `anansi:"next"`
	Prev                   string   `anansi:"prev"`
	Paused                 bool     `anansi:"paused"`
	Tags                   []string `anansi:"tags"`
}

type SchedulerListOutput struct {
	Documents []SchedulerJobInfo `anansi:"documents"`
}
