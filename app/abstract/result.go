package abstract

import (
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/data"
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
	}
	return nil
}

type Page struct {
	Documents  data.DocumentSet
	Pagination *query.PaginationInfo
}

type Blob struct {
	Data        []byte
	ContentType string
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

type Result struct {
	Kind            ResultKind
	Document        *data.Document
	Documents       data.DocumentSet
	Page            *Page
	Blob            Blob
	DocumentChannel <-chan *data.Document
	BlobChannel     <-chan Blob
	SessionToken    string
}

func NewDocumentResult(doc *data.Document) *Result {
	return &Result{Kind: ResultKindDocument, Document: doc}
}

func NewDocumentsResult(docs data.DocumentSet) *Result {
	return &Result{Kind: ResultKindDocuments, Documents: docs}
}

func NewPageResult(page *Page) *Result {
	return &Result{Kind: ResultKindPage, Page: page}
}

func NewBlobResult(blob Blob) *Result {
	return &Result{Kind: ResultKindBlob, Blob: blob}
}

func NewDocumentChannelResult(ch <-chan *data.Document) *Result {
	return &Result{Kind: ResultKindDocumentChannel, DocumentChannel: ch}
}

func NewBlobChannelResult(ch <-chan Blob) *Result {
	return &Result{Kind: ResultKindBlobChannel, BlobChannel: ch}
}
