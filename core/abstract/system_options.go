package abstract

// SystemScopePrefix is the prefix for all system-level permission scopes.
//
// Build-time configuration only: override via
//
	// go build -ldflags '-X github.com/asaidimu/hestia/core/abstract.SystemScopePrefix=hestia'
//
// The variable must never be written at runtime (audit A-7): permissions
// are minted with the prefix at session creation and re-checked with it on
// every dispatch, so a runtime mutation would silently invalidate live
// credentials or — worse — re-scope them mid-flight. There is no setter;
// keep it that way.
var SystemScopePrefix = "system"

type DispatcherLink interface {
	Wrap(next Dispatcher) Dispatcher
}

type ChainEditor interface {
	InsertBefore(name string, newName string, link DispatcherLink)
	InsertAfter(name string, newName string, link DispatcherLink)
	Remove(name string)
}
