package abstract

import (
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

type Verb int

const (
	Create Verb = iota + 1
	Read
	Update
	Delete
	Query
	Stream
	Check
)

func (v Verb) String() string {
	switch v {
	case Create:
		return "CREATE"
	case Read:
		return "READ"
	case Update:
		return "UPDATE"
	case Delete:
		return "DELETE"
	case Query:
		return "QUERY"
	case Stream:
		return "STREAM"
	case Check:
		return "CHECK"
	}
	return ""
}

func (v Verb) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func (v *Verb) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "CREATE":
		*v = Create
	case "READ":
		*v = Read
	case "UPDATE":
		*v = Update
	case "DELETE":
		*v = Delete
	case "QUERY":
		*v = Query
	case "STREAM":
		*v = Stream
	case "CHECK":
		*v = Check
	}
	return nil
}

type Page struct {
	Documents  []*document.Document
	Pagination *query.PaginationInfo
}

type ResultKind int

const (
	ResultKindDocument ResultKind = iota + 1
	ResultKindDocuments
	ResultKindPage
	ResultKindBlob
	ResultKindDocumentChannel
	ResultKindBlobChannel
)

// Result is the single envelope every handler returns. Its document-bearing
// fields are typed concretely as *document.Document (never the data.Documenter
// interface), so a deprecated data.Document construction path can't be
// smuggled into a result: the compile-time type closes that off structurally.
type Result struct {
	Kind            ResultKind
	Document        *document.Document
	Documents       []*document.Document
	Page            *Page
	Blob            Blob
	DocumentChannel <-chan *document.Document
	BlobChannel     <-chan Blob
	SessionToken    string
	Metadata        map[string]any
}

// Release returns pooled resources owned by the result to their pools: the
// single Document, every document in Documents and in Page.Documents, and the
// Blob buffer. It is safe to call on a nil or already-released result, and a
// second call is a no-op.
//
// Streaming kinds (ResultKindDocumentChannel / ResultKindBlobChannel) are not
// touched: the documents flowing over those channels are owned by whoever is
// draining the channel, not by the result.
//
// After Release, the result must not be read for its documents; scalar fields
// (Kind, SessionToken, Metadata) remain valid.
func (r *Result) Release() {
	if r == nil {
		return
	}
	if r.Document != nil {
		r.Document.Release()
		r.Document = nil
	}
	releaseDocs(r.Documents)
	r.Documents = nil
	if r.Page != nil {
		releaseDocs(r.Page.Documents)
		r.Page.Documents = nil
	}
	r.Blob.Free()
}

func releaseDocs(docs []*document.Document) {
	for _, d := range docs {
		if d != nil {
			d.Release()
		}
	}
}
