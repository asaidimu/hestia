package dispatch

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/google/uuid"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

// message is the single abstract.Message implementation in this package.
// All request metadata (tenant, trace, session, ...) is read from the
// embedded context via runtimecontext accessors.
type message struct {
	id      string
	name    string
	ctx     context.Context
	input   data.Documenter
	inputCh chan data.Documenter // exposed receive-only via InputChannel
	blobCh  <-chan abstract.Blob
}

func (m *message) ID() string                             { return m.id }
func (m *message) Name() string                           { return m.name }
func (m *message) Context() context.Context               { return m.ctx }
func (m *message) Input() data.Documenter                 { return m.input }
func (m *message) InputChannel() <-chan data.Documenter    { return m.inputCh }
func (m *message) BlobInputChannel() <-chan abstract.Blob { return m.blobCh }

func (m *message) TenantID() string   { return runtimecontext.GetTenantID(m.ctx) }
func (m *message) TraceID() string    { return runtimecontext.GetTraceID(m.ctx) }
func (m *message) RequestID() string  { return runtimecontext.GetRequestID(m.ctx) }
func (m *message) SourceIP() string   { return runtimecontext.GetSourceIP(m.ctx) }
func (m *message) UserAgent() string  { return runtimecontext.GetUserAgent(m.ctx) }
func (m *message) ResourceID() string { return runtimecontext.GetResourceID(m.ctx) }
func (m *message) SessionID() string  { return runtimecontext.GetSessionID(m.ctx) }

func NewMessage(name string, ctx context.Context, input data.Documenter) abstract.Message {
	return &message{id: MustNewID(), name: name, ctx: ctx, input: input}
}

func MustNewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
