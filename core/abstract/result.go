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
	Documents  data.DocumentSet
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

type Result struct {
	Kind            ResultKind
	Document        data.Documenter
	Documents       data.DocumentSet
	Page            *Page
	Blob            Blob
	DocumentChannel <-chan data.Documenter
	BlobChannel     <-chan Blob
	SessionToken    string
	Metadata        map[string]any
}
