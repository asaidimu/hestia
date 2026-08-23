package runtime

import (
	"context"
	"strings"

	"github.com/asaidimu/hestia/core/abstract"
)

type NamespacedDispatcher struct {
	prefix   string
	next     abstract.Dispatcher
	hydrator func(abstract.Message) (abstract.Message, error)
}

func NewNamespacedDispatcher(prefix string, next abstract.Dispatcher, hydrator func(abstract.Message) (abstract.Message, error)) *NamespacedDispatcher {
	return &NamespacedDispatcher{prefix: prefix, next: next, hydrator: hydrator}
}

func (d *NamespacedDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &NamespacedDispatcher{prefix: d.prefix, next: next, hydrator: d.hydrator}
}

func (d *NamespacedDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if !strings.HasPrefix(msg.Name(), d.prefix) {
		return d.next.Send(ctx, msg, onComplete)
	}
	hydrated, err := d.hydrator(msg)
	if err != nil {
		return err
	}
	return d.next.Send(ctx, hydrated, onComplete)
}

var _ abstract.Dispatcher = (*NamespacedDispatcher)(nil)
