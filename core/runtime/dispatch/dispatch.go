// @note #arch-20260821-006 issue status=open priority=P2 tags=#arch,#duplication : Duplicate message implementation in dispatch package
//
// dispatchMessage (dispatch.go:20-41) and genericMessage (message.go:13-35)
// both implement abstract.Message with identical method bodies delegating to
// runtimecontext.GetTenantID(m.ctx), runtimecontext.GetTraceID(m.ctx), etc.
//
// This is 14 lines of pure duplication across two files in the same package.
//
// Resolution: Unify into a single type or embed a shared implementation.
// One option is to make dispatchMessage embed a genericMessage field and
// override only the methods that differ (InputChannel and BlobInputChannel).
package dispatch

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

type DispatchInput struct {
	Name     string
	Context  context.Context
	ID       string
	Document data.Documenter
	Intent   abstract.Verb
}

type dispatchMessage struct {
	id      string
	name    string
	ctx     context.Context
	input   data.Documenter
	inputCh chan data.Documenter
}

func (m *dispatchMessage) ID() string                             { return m.id }
func (m *dispatchMessage) Name() string                           { return m.name }
func (m *dispatchMessage) Context() context.Context               { return m.ctx }
func (m *dispatchMessage) Input() data.Documenter                 { return m.input }
func (m *dispatchMessage) InputChannel() <-chan data.Documenter    { return m.inputCh }
func (m *dispatchMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

func (m *dispatchMessage) TenantID() string   { return runtimecontext.GetTenantID(m.ctx) }
func (m *dispatchMessage) TraceID() string    { return runtimecontext.GetTraceID(m.ctx) }
func (m *dispatchMessage) RequestID() string  { return runtimecontext.GetRequestID(m.ctx) }
func (m *dispatchMessage) SourceIP() string   { return runtimecontext.GetSourceIP(m.ctx) }
func (m *dispatchMessage) UserAgent() string  { return runtimecontext.GetUserAgent(m.ctx) }
func (m *dispatchMessage) ResourceID() string { return runtimecontext.GetResourceID(m.ctx) }
func (m *dispatchMessage) SessionID() string  { return runtimecontext.GetSessionID(m.ctx) }

const streamChannelBuffer = 1

func Dispatch(disp abstract.Dispatcher, in DispatchInput) (*abstract.Result, error) {
	msgID := in.ID
	if msgID == "" {
		msgID = MustNewID()
	}

	msg := &dispatchMessage{
		id:    msgID,
		name:  in.Name,
		ctx:   in.Context,
		input: in.Document,
	}

	if in.Intent == abstract.Stream {
		msg.inputCh = make(chan data.Documenter, streamChannelBuffer)
	}

	result, err := disp.Send(msg)
	if err != nil {
		return nil, err
	}

	if in.Intent == abstract.Stream {
		msg.inputCh <- data.MustNewDocument(map[string]any{}, in.Context)
		close(msg.inputCh)
	}

	return result, nil
}
