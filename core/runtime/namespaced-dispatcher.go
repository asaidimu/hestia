package runtime

import (
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

func (d *NamespacedDispatcher) Send(msg abstract.Message) (*abstract.Result, error) {
	if !strings.HasPrefix(msg.Name(), d.prefix) {
		return d.next.Send(msg)
	}
	hydrated, err := d.hydrator(msg)
	if err != nil {
		return nil, err
	}
	return d.next.Send(hydrated)
}

var _ abstract.Dispatcher = (*NamespacedDispatcher)(nil)
