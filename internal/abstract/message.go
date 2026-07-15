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

func NewMessage(name string, ctx context.Context, input *data.Document) Message {
	return &genericMessage{id: MustNewID(), name: name, ctx: ctx, input: input}
}

func MustNewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
