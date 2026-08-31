package abstract

import (
	"context"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
)

// StreamItem is one element of a streamed input channel. Doc carries the
// per-item input document (one NDJSON payload wrapped in the request envelope
// by the transport); Err carries a framing or per-item validation failure so
// the handler owns its abort-vs-collect policy. Exactly one of Doc/Err is
// meaningful: Doc is nil when Err is set.
type StreamItem struct {
	Doc *document.Document
	Err error
}

type Message interface {
	ID() string
	Name() string
	Context() context.Context
	Input() data.Documenter
	InputChannel() <-chan StreamItem
	BlobInputChannel() <-chan Blob

	TenantID() string
	TraceID() string
	RequestID() string
	SourceIP() string
	UserAgent() string
	ResourceID() string
	SessionID() string
}
