package dispatch

import (
	"github.com/asaidimu/hestia/core/abstract"
	"go.uber.org/zap"
)

type SystemOptions struct {
	OnBootstrapped func()
	OnReset        func()

	// OnRestartRequired interprets runtime.ErrRestartRequired outcomes from
	// the dispatch chain and the update apply path (audit A-15). The stock
	// host exits cleanly so its supervisor restarts the process; embedded
	// hosts may schedule their own restart or defer it. When nil, the chain
	// has no restart observer and restart-required outcomes surface to the
	// client as 503 RESTART_REQUIRED without any process termination.
	Logger            *zap.Logger
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool

	DispatcherChainFunc func(chain abstract.ChainEditor)

	OnRestartRequired func(error)
}
