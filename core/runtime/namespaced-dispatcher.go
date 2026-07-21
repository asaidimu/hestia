package runtime

import (
	"strings"

	"github.com/asaidimu/hestia/core/registration"
)

type NamespacedDispatcher struct {
	prefix   string
	next     Dispatcher
	hydrator func(Message) (Message, error)
}

func NewNamespacedDispatcher(prefix string, next Dispatcher, hydrator func(Message) (Message, error)) *NamespacedDispatcher {
	return &NamespacedDispatcher{prefix: prefix, next: next, hydrator: hydrator}
}

func (d *NamespacedDispatcher) Send(msg Message) (*registration.Result, error) {
	if !strings.HasPrefix(msg.Name(), d.prefix) {
		return d.next.Send(msg)
	}
	hydrated, err := d.hydrator(msg)
	if err != nil {
		return nil, err
	}
	return d.next.Send(hydrated)
}

var _ Dispatcher = (*NamespacedDispatcher)(nil)
