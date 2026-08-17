package dispatch

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
)

// Handler is the business-logic shape Handle wraps: a single bound input,
// plus ctx and the raw Message for handlers that still need claims, tenant
// ID, or other Message accessors beyond what landed in TIn.
//
// Handle is for the single-document request/response contract used by
// Create/Read/Update/Delete/Query/Check registrations — not for streaming
// registrations (see HandleInputStream in the spec): a streaming handler's
// contract is "many items over a channel," not "one document, bound once."
type Handler[TIn any] func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Result, error)

// Handle adapts a Handler[TIn] into an abstract.MessageHandler. It binds
// msg.Input() into a *TIn using the "input" struct tag, releases the pooled
// input document, then invokes fn. The document is released whether or not
// binding succeeds, since Handle is its last owner either way.
func Handle[TIn any](fn Handler[TIn]) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input TIn
		doc := msg.Input()
		defer doc.Release()

		if err := doc.BindToTagWithContext(ctx, &input, "input"); err != nil {
			return nil, err
		}
		return fn(ctx, msg, &input)
	}
}

// HandleDocument composes Handle with the Document-result boilerplate every
// Get/Create/Update-shaped handler repeats by hand: call the business
// method, propagate its error, wrap a successful model into a Result.
// Built on Handle — input bind/release happens exactly once, in one place.
func HandleDocument[TIn any, TOut documentModel](fn func(ctx context.Context, msg abstract.Message, input *TIn) (TOut, error)) abstract.MessageHandler {
	return Handle(func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Result, error) {
		var zero TOut
		model, err := fn(ctx, msg, input)
		if err != nil {
			return nil, err
		}
		if any(model) == any(zero) {
			return nil, nil
		}
		doc, err := model.Document()
		if err != nil {
			return nil, err
		}
		return NewDocumentResult(doc), nil
	})
}

// DocumentsHandler is HandleDocument's list-shaped counterpart: fn maps
// business models to *document.Document itself (the per-item .Document()
// call can't be generalized further — the mapping loop is still the
// caller's, but the Result construction and error-check boilerplate around
// it isn't).
type DocumentsHandler[TIn any] func(ctx context.Context, msg abstract.Message, input *TIn) ([]*document.Document, error)

// HandleDocuments wraps a DocumentsHandler into an abstract.MessageHandler,
// binding and releasing the input document exactly once.
func HandleDocuments[TIn any](fn DocumentsHandler[TIn]) abstract.MessageHandler {
	return Handle(func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Result, error) {
		docs, err := fn(ctx, msg, input)
		if err != nil {
			return nil, err
		}
		return NewDocumentsResult(docs), nil
	})
}

// PageHandler returns a *abstract.Page from business logic; HandlePage wraps
// it with input bind/release and Result construction.
type PageHandler[TIn any] func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Page, error)

func HandlePage[TIn any](fn PageHandler[TIn]) abstract.MessageHandler {
	return Handle(func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Result, error) {
		page, err := fn(ctx, msg, input)
		if err != nil {
			return nil, err
		}
		return NewPageResult(page), nil
	})
}

// BlobHandler returns a *abstract.Blob from business logic; HandleBlob wraps
// it with input bind/release and Result construction.
type BlobHandler[TIn any] func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Blob, error)

func HandleBlob[TIn any](fn BlobHandler[TIn]) abstract.MessageHandler {
	return Handle(func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Result, error) {
		blob, err := fn(ctx, msg, input)
		if err != nil {
			return nil, err
		}
		return NewBlobResult(*blob), nil
	})
}

// EmptyHandler is for Delete/ChangePassword-shaped handlers: business logic
// that returns only an error, and on success a bare &Result{}.
type EmptyHandler[TIn any] func(ctx context.Context, msg abstract.Message, input *TIn) error

func HandleEmpty[TIn any](fn EmptyHandler[TIn]) abstract.MessageHandler {
	return Handle(func(ctx context.Context, msg abstract.Message, input *TIn) (*abstract.Result, error) {
		if err := fn(ctx, msg, input); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	})
}
