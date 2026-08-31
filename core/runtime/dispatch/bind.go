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
// Handle serves the single-document request/response contract used by
// Create/Read/Update/Delete/Query/Check registrations. Streaming
// registrations use HandleInputStream (many items over a channel) or
// HandleOutputStream (single input, streamed Result) instead.
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

// HandleDocument wraps a handler that returns a document model, binding and
// releasing the input document exactly once. If fn returns a nil model (without
// error), the result is nil — useful for write operations that produce no output.
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
//
// ---------------------------------------------------------------------------
// Streaming adapters
// ---------------------------------------------------------------------------

// Item pairs a bound value with a possible per-item error — a channel alone
// can't carry "this one item failed to bind" the way a return value can, and
// treating every bind failure as fatal to the whole stream would abort a
// 10,000-record import over one malformed line. Whether a failed item aborts
// the stream or is collected and counted is the handler's policy.
type Item[TIn any] struct {
	Value TIn
	Err   error
}

// StreamHandler is the business-logic shape HandleInputStream wraps: many
// bound input items arriving over msg.InputChannel(), producing a single
// Result once the stream ends (e.g. an import summary).
//
// Lifecycle contract: the adapter guarantees that when fn returns — normally,
// early, or via panic — the input producer (the transport's body reader) is
// always drained to completion and every received document is released. fn
// should still consume `items` to close in the normal path; abandoning it
// early is safe but wastes the rest of the body.
type StreamHandler[TIn any] func(ctx context.Context, msg abstract.Message, items <-chan Item[TIn]) (*abstract.Result, error)

// HandleInputStream adapts a StreamHandler[TIn] into an abstract.MessageHandler.
// Each StreamItem from msg.InputChannel() is bound into TIn via the "input"
// struct tag (or surfaced as Item.Err for per-item framing/validation
// failures), then handed to fn. Every intermediate document is released
// exactly once, whether or not binding succeeds.
func HandleInputStream[TIn any](fn StreamHandler[TIn]) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		out := make(chan Item[TIn])
		fnDone := make(chan struct{})
		go func() {
			defer close(out)
			defer func() {
				// Drain-and-release: when fn returned early, panicked, or ctx
				// fired, this guarantees the producer can run to EOF and
				// close instead of blocking forever on a send with no
				// receiver (fasthttp's RequestCtx.Done() never fires on
				// disconnect — see the S-19 fix), and that no pooled document
				// leaks.
				for rest := range msg.InputChannel() {
					if rest.Doc != nil {
						rest.Doc.Release()
					}
				}
			}()
			for si := range msg.InputChannel() {
				var item Item[TIn]
				switch {
				case si.Err != nil:
					if si.Doc != nil {
						si.Doc.Release()
					}
					item = Item[TIn]{Err: si.Err}
				default:
					var v TIn
					err := si.Doc.BindToTagWithContext(ctx, &v, "input")
					si.Doc.Release()
					item = Item[TIn]{Value: v, Err: err}
				}
				select {
				case out <- item:
				case <-fnDone:
					return // deferred drain takes over
				case <-ctx.Done():
					return
				}
			}
		}()
		defer close(fnDone)
		return fn(ctx, msg, out)
	}
}

// HandleOutputStream: single bound input, Result payload delivered
// incrementally (ResultKindDocumentChannel / ResultKindBlobChannel).
// Mechanically identical to Handle — the separate name exists purely so
// call sites self-document that the Result they build is a channel kind.
func HandleOutputStream[TIn any](fn Handler[TIn]) abstract.MessageHandler {
	return Handle(fn)
}
