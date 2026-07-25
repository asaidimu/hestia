package runtime

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/registration"
)

type TenantDispatcher struct {
	next     Dispatcher
	resolver func(context.Context) string
}

func NewTenantDispatcher(next Dispatcher, resolver func(context.Context) string) *TenantDispatcher {
	return &TenantDispatcher{next: next, resolver: resolver}
}

func (d *TenantDispatcher) Wrap(next Dispatcher) Dispatcher {
	return &TenantDispatcher{next: next, resolver: d.resolver}
}

func (d *TenantDispatcher) Send(msg Message) (*registration.Result, error) {
	if tenantID := d.resolver(msg.Context()); tenantID != "" {
		ctx := ContextWithTenantID(msg.Context(), tenantID)
		msg = &tenantMessage{Message: msg, ctx: ctx}
	}
	return d.next.Send(msg)
}

type tenantMessage struct {
	abstract.Message
	ctx context.Context
}

func (m *tenantMessage) Context() context.Context { return m.ctx }

var _ Dispatcher = (*TenantDispatcher)(nil)
