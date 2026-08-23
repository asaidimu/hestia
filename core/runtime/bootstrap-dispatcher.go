package runtime

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/core/abstract"
)

// BootstrapDispatcher gates message dispatch behind the bootstrapping
// state. When not bootstrapped, only handlers marked BootstrapSafe are
// allowed through. This ensures all interface types (HTTP, CLI, Wails)
// enforce the same pre-bootstrap restrictions uniformly.
type BootstrapDispatcher struct {
	next         abstract.Dispatcher
	registry     interface{ IsHandlerBootstrapSafe(name string) bool }
	bootstrapped func() bool
}

func NewBootstrapDispatcher(next abstract.Dispatcher, registry interface{ IsHandlerBootstrapSafe(name string) bool }, bootstrapped func() bool) *BootstrapDispatcher {
	return &BootstrapDispatcher{next: next, registry: registry, bootstrapped: bootstrapped}
}

func (d *BootstrapDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &BootstrapDispatcher{next: next, registry: d.registry, bootstrapped: d.bootstrapped}
}

func (d *BootstrapDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if !d.bootstrapped() && !d.registry.IsHandlerBootstrapSafe(msg.Name()) {
		return fmt.Errorf("handler %q is not available until the system is bootstrapped", msg.Name())
	}
	return d.next.Send(ctx, msg, onComplete)
}

var _ abstract.Dispatcher = (*BootstrapDispatcher)(nil)
