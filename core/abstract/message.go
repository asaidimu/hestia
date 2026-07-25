package abstract

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/google/uuid"
)

type Message interface {
	ID() string
	Name() string
	Context() context.Context
	Input() *data.Document
	InputChannel() <-chan *data.Document
	BlobInputChannel() <-chan Blob

	// Context accessors provide typed, discoverable access to
	// request-scoped metadata carried in the context.
	TenantID() string
	TraceID() string
	RequestID() string
	SourceIP() string
	UserAgent() string
	ResourceID() string
	SessionID() string
}

type genericMessage struct {
	id      string
	name    string
	ctx     context.Context
	input   *data.Document
	inputCh <-chan *data.Document
	blobCh  <-chan Blob
}

func (m *genericMessage) ID() string                          { return m.id }
func (m *genericMessage) Name() string                        { return m.name }
func (m *genericMessage) Context() context.Context             { return m.ctx }
func (m *genericMessage) Input() *data.Document                { return m.input }
func (m *genericMessage) InputChannel() <-chan *data.Document  { return m.inputCh }
func (m *genericMessage) BlobInputChannel() <-chan Blob        { return m.blobCh }

func (m *genericMessage) TenantID() string   { return GetTenantID(m.ctx) }
func (m *genericMessage) TraceID() string    { return GetTraceID(m.ctx) }
func (m *genericMessage) RequestID() string  { return GetRequestID(m.ctx) }
func (m *genericMessage) SourceIP() string   { return GetSourceIP(m.ctx) }
func (m *genericMessage) UserAgent() string  { return GetUserAgent(m.ctx) }
func (m *genericMessage) ResourceID() string { return GetResourceID(m.ctx) }
func (m *genericMessage) SessionID() string  { return GetSessionID(m.ctx) }

func NewMessage(name string, ctx context.Context, input *data.Document) Message {
	return &genericMessage{id: MustNewID(), name: name, ctx: ctx, input: input}
}

func MustNewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
