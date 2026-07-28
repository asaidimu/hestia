package dispatch

import (
	"github.com/asaidimu/hestia/core/abstract"
	"go.uber.org/zap"
)

type SystemOptions struct {
	OnBootstrapped    func()
	OnReset           func()
	Logger            *zap.Logger
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool

	DispatcherChainFunc func(chain abstract.ChainEditor)
}
