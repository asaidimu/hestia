package abstract

import "go.uber.org/zap"

// DispatcherLink wraps a Dispatcher with cross-cutting behaviour.
type DispatcherLink interface {
	Wrap(next Dispatcher) Dispatcher
}

// ChainEditor allows inserting/removing links in a dispatch chain.
type ChainEditor interface {
	InsertBefore(name string, link DispatcherLink)
	InsertAfter(name string, link DispatcherLink)
	Remove(name string)
}

type SystemOptions struct {
	OnBootstrapped    func()
	OnReset           func()
	Logger            *zap.Logger
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool

	// DispatcherChainFunc is called after the default chain is
	// populated. Users can InsertBefore/InsertAfter/Remove links.
	DispatcherChainFunc func(chain ChainEditor)
}
