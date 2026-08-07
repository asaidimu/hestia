package dispatch

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/google/uuid"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

type genericMessage struct {
	id      string
	name    string
	ctx     context.Context
	input   *data.Document
	inputCh <-chan *data.Document
	blobCh  <-chan abstract.Blob
}

func (m *genericMessage) ID() string                             { return m.id }
func (m *genericMessage) Name() string                           { return m.name }
func (m *genericMessage) Context() context.Context               { return m.ctx }
func (m *genericMessage) Input() *data.Document                  { return m.input }
func (m *genericMessage) InputChannel() <-chan *data.Document    { return m.inputCh }
func (m *genericMessage) BlobInputChannel() <-chan abstract.Blob { return m.blobCh }

func (m *genericMessage) TenantID() string   { return runtimecontext.GetTenantID(m.ctx) }
func (m *genericMessage) TraceID() string    { return runtimecontext.GetTraceID(m.ctx) }
func (m *genericMessage) RequestID() string  { return runtimecontext.GetRequestID(m.ctx) }
func (m *genericMessage) SourceIP() string   { return runtimecontext.GetSourceIP(m.ctx) }
func (m *genericMessage) UserAgent() string  { return runtimecontext.GetUserAgent(m.ctx) }
func (m *genericMessage) ResourceID() string { return runtimecontext.GetResourceID(m.ctx) }
func (m *genericMessage) SessionID() string  { return runtimecontext.GetSessionID(m.ctx) }

func NewMessage(name string, ctx context.Context, input *data.Document) abstract.Message {
	return &genericMessage{id: MustNewID(), name: name, ctx: ctx, input: input}
}

func MustNewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
