package dispatch

import (
	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
)

func NewDocumentResult(doc *document.Document) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindDocument, Document: doc}
}

func NewDocumentsResult(docs []*document.Document) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindDocuments, Documents: docs}
}

func NewPageResult(page *abstract.Page) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindPage, Page: page}
}

func NewBlobResult(blob abstract.Blob) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindBlob, Blob: blob}
}

func NewDocumentChannelResult(ch <-chan *document.Document) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindDocumentChannel, DocumentChannel: ch}
}

func NewBlobChannelResult(ch <-chan abstract.Blob) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindBlobChannel, BlobChannel: ch}
}

// documentModel is satisfied by every struct that embeds document.DocumentModel,
// whose promoted Document() method returns the schema-bound *document.Document.
type documentModel interface {
	Document() (*document.Document, error)
}

// NewDocumentResultFrom builds a Result carrying a document constructed from an
// anansi-tagged view struct via document.New. view must be a pointer to a struct
// embedding document.DocumentModel.
func NewDocumentResultFrom[T any, P interface {
	*T
	documentModel
}](view P) (*abstract.Result, error) {
	model := document.New[T](view)
	doc, err := (P(model)).Document()
	if err != nil {
		return nil, err
	}
	return NewDocumentResult(doc), nil
}
