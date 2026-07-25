package transport

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/registration"
)

type Input struct {
	Name    string
	Context context.Context
	ID      string
	Document *data.Document
	Intent  registration.Verb
}

type dispatchMessage struct {
	id      string
	name    string
	ctx     context.Context
	input   *data.Document
	inputCh chan *data.Document
}

func (m *dispatchMessage) ID() string                         { return m.id }
func (m *dispatchMessage) Name() string                       { return m.name }
func (m *dispatchMessage) Context() context.Context            { return m.ctx }
func (m *dispatchMessage) Input() *data.Document               { return m.input }
func (m *dispatchMessage) InputChannel() <-chan *data.Document { return m.inputCh }
func (m *dispatchMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

func (m *dispatchMessage) TenantID() string   { return abstract.GetTenantID(m.ctx) }
func (m *dispatchMessage) TraceID() string    { return abstract.GetTraceID(m.ctx) }
func (m *dispatchMessage) RequestID() string  { return abstract.GetRequestID(m.ctx) }
func (m *dispatchMessage) SourceIP() string   { return abstract.GetSourceIP(m.ctx) }
func (m *dispatchMessage) UserAgent() string  { return abstract.GetUserAgent(m.ctx) }
func (m *dispatchMessage) ResourceID() string { return abstract.GetResourceID(m.ctx) }
func (m *dispatchMessage) SessionID() string  { return abstract.GetSessionID(m.ctx) }

func Dispatch(disp abstract.Dispatcher, in Input) (*registration.Result, error) {
	msgID := in.ID
	if msgID == "" {
		msgID = abstract.MustNewID()
	}

	msg := &dispatchMessage{
		id:    msgID,
		name:  in.Name,
		ctx:   in.Context,
		input: in.Document,
	}

	if in.Intent == registration.Stream {
		msg.inputCh = make(chan *data.Document)
	}

	result, err := disp.Send(msg)
	if err != nil {
		return nil, err
	}

	if in.Intent == registration.Stream {
		msg.inputCh <- data.MustNewDocument(map[string]any{}, in.Context)
	}

	return result, nil
}
