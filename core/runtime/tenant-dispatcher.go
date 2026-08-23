package runtime

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

type TenantDispatcher struct {
	next     abstract.Dispatcher
	resolver func(context.Context) string
}

func NewTenantDispatcher(next abstract.Dispatcher, resolver func(context.Context) string) *TenantDispatcher {
	return &TenantDispatcher{next: next, resolver: resolver}
}

func (d *TenantDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &TenantDispatcher{next: next, resolver: d.resolver}
}

func (d *TenantDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if tenantID := d.resolver(msg.Context()); tenantID != "" {
		tctx := runtimecontext.ContextWithTenantID(msg.Context(), tenantID)
		msg = &tenantMessage{Message: msg, ctx: tctx}
	}
	return d.next.Send(ctx, msg, onComplete)
}

type tenantMessage struct {
	abstract.Message
	ctx context.Context
}

func (m *tenantMessage) Context() context.Context { return m.ctx }

var _ abstract.Dispatcher = (*TenantDispatcher)(nil)
