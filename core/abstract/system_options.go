package abstract

type DispatcherLink interface {
	Wrap(next Dispatcher) Dispatcher
}

type ChainEditor interface {
	InsertBefore(name string, link DispatcherLink)
	InsertAfter(name string, link DispatcherLink)
	Remove(name string)
}
