package boot

import (
	"github.com/asaidimu/hestia/core/runtime"
)

// DefaultProjectName is used when the embedder does not supply a project
// name; it selects the .env / <project>.json manifest loaded by
// runtime.LoadConfig.
const DefaultProjectName = "hestia"

// NewConfig loads the runtime config for the named project. The project name
// is an explicit parameter (audit A-7: was an imperatively-assigned package
// global, so concurrent/embedded boots raced on it).
func NewConfig(project string) (*runtime.Config, error) {
	if project == "" {
		project = DefaultProjectName
	}
	return runtime.LoadConfig(project)
}
