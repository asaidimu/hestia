package abstract

import (
	"context"
	"github.com/asaidimu/go-anansi/v8/core/data"
)

type Message interface {
	ID() string
	Name() string
	Context() context.Context
	Input() *data.Document
	InputChannel() <-chan *data.Document
	BlobInputChannel() <-chan Blob

	TenantID() string
	TraceID() string
	RequestID() string
	SourceIP() string
	UserAgent() string
	ResourceID() string
	SessionID() string
}
