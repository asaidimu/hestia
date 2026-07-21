package abstract

import "go.uber.org/zap"

type SystemOptions struct {
	OnBootstrapped    func()
	OnReset           func()
	Logger            *zap.Logger
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool

	DispatcherHooks []func(Dispatcher) Dispatcher
}
