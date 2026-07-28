package dispatch

import (
	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
)

func NewDocumentResult(doc *data.Document) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindDocument, Document: doc}
}

func NewDocumentsResult(docs data.DocumentSet) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindDocuments, Documents: docs}
}

func NewPageResult(page *abstract.Page) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindPage, Page: page}
}

func NewBlobResult(blob abstract.Blob) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindBlob, Blob: blob}
}

func NewDocumentChannelResult(ch <-chan *data.Document) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindDocumentChannel, DocumentChannel: ch}
}

func NewBlobChannelResult(ch <-chan abstract.Blob) *abstract.Result {
	return &abstract.Result{Kind: abstract.ResultKindBlobChannel, BlobChannel: ch}
}
