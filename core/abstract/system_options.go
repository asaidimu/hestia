package abstract

// SystemScopePrefix is the prefix for all system-level permission scopes.
// Override at build time: go build -ldflags '-X github.com/asaidimu/hestia/core/abstract.SystemScopePrefix=hestia'
var SystemScopePrefix = "system"

type DispatcherLink interface {
	Wrap(next Dispatcher) Dispatcher
}

type ChainEditor interface {
	InsertBefore(name string, newName string, link DispatcherLink)
	InsertAfter(name string, newName string, link DispatcherLink)
	Remove(name string)
}
