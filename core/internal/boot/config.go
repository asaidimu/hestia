package boot

import (
	"github.com/asaidimu/hestia/core/runtime"
)

var ProjectName = "hestia"

func NewConfig() (*runtime.Config, error) {
	return runtime.LoadConfig(ProjectName)
}
