package runtime

import (
        "context"
        "strings"

        "github.com/asaidimu/go-anansi/v8/core/common"
        "github.com/asaidimu/go-anansi/v8/core/document"

        "github.com/asaidimu/hestia/core/abstract"
)

// SanitizationDispatcher sets the sanitization scope on ingoing messages
// and sanitizes outgoing documents. The scope is derived from the feature
// name (second segment of the message name: "system:users:user:get" → "users").
type SanitizationDispatcher struct {
        next abstract.Dispatcher
}

// ResultSanitizationScopeMetaKey is the result-metadata key under which the
// dispatcher stashes the message's sanitization scope for STREAMED results.
// The SSE transport applies it per document while draining (see
// serializeStreamResult): an interposed sanitizer goroutine here would block
// forever once the writer stops consuming, because fasthttp's RequestCtx.Done()
// never fires on client disconnect and stream teardown is writer-owned (S-19).
const ResultSanitizationScopeMetaKey = "hestia.sanitization.result_scope"

// StreamSanitizeArgs resolves the sanitization scope stashed on a streamed
// result (see sanitizeResult for why the scope is stashed rather than applied
// inline) and re-materializes it as a context suitable for per-document
// Sanitize calls. The metadata key is consumed so it never leaks into
// response metadata. Returns nil when no scope was stashed.
func StreamSanitizeArgs(ctx context.Context, res *abstract.Result) []context.Context {
        if res.Metadata == nil {
                return nil
        }
        scopes, ok := res.Metadata[ResultSanitizationScopeMetaKey].([]string)
        delete(res.Metadata, ResultSanitizationScopeMetaKey)
        if !ok || len(scopes) == 0 {
                return nil
        }
        sctx := ctx
        for _, s := range scopes {
                sctx = common.ContextWithSanitizationScope(sctx, s)
        }
        return []context.Context{sctx}
}

func NewSanitizationDispatcher(next abstract.Dispatcher) *SanitizationDispatcher {
        return &SanitizationDispatcher{next: next}
}

func (d *SanitizationDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
        return &SanitizationDispatcher{next: next}
}

func (d *SanitizationDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
        scope := featureScope(msg.Name())

        // Set scope on message context so downstream handlers/dispatchers see it.
        // sanitizeCtx is the context sanitization runs against: the completion
        // callback's ctx is the caller's ctx, which does NOT carry the message's
        // scope, so the scoped message context is captured here instead.
        sanitizeCtx := ctx
        enrichedMsg := msg
        if scope != "" {
                tctx := common.ContextWithSanitizationScope(msg.Context(), scope)
                enrichedMsg = &sanitizationMessage{Message: msg, ctx: tctx}
                sanitizeCtx = tctx
        }

        return d.next.Send(ctx, enrichedMsg, func(cctx context.Context, res *abstract.Result, handlerErr error) {
                if res != nil && handlerErr == nil {
                        sanitizeResult(res, sanitizeCtx)
                }
                onComplete(cctx, res, handlerErr)
        })
}

// featureScope extracts the feature name from a message name.
// "system:users:user:get" → "users", "system:apikeys:key:create" → "apikeys".
func featureScope(msgName string) string {
        parts := strings.SplitN(msgName, ":", 3)
        if len(parts) >= 2 {
                return parts[1]
        }
        return ""
}

// sanitizeResult sanitizes all documents in a result using the context's scope.
func sanitizeResult(res *abstract.Result, ctx context.Context) {
        if res.Document != nil {
                if sane, err := res.Document.Sanitize(ctx); err == nil && sane != nil {
                        if d, ok := sane.(*document.Document); ok {
                                res.Document = d
                        }
                }
        }
        if res.Documents != nil {
                sanitized := make([]*document.Document, 0, len(res.Documents))
                for _, d := range res.Documents {
                        if sane, err := d.Sanitize(ctx); err == nil && sane != nil {
                                if dd, ok := sane.(*document.Document); ok {
                                        sanitized = append(sanitized, dd)
                                }
                        }
                }
                res.Documents = sanitized
        }
        if res.Page != nil && res.Page.Documents != nil {
                sanitized := make([]*document.Document, 0, len(res.Page.Documents))
                for _, d := range res.Page.Documents {
                        if sane, err := d.Sanitize(ctx); err == nil && sane != nil {
                                if dd, ok := sane.(*document.Document); ok {
                                        sanitized = append(sanitized, dd)
                                }
                        }
                }
                res.Page.Documents = sanitized
        }
        // Streamed outputs (DocumentChannel) cannot be wrapped with a sanitizer
        // goroutine here: stream teardown is writer-owned (S-19) and the writer
        // stops consuming without closing the source, so an interposed goroutine
        // would block forever on a send with no receiver. The scope is stashed on
        // the result instead; the transport applies it per document as it drains
        // (serializeStreamResult).
        if res.DocumentChannel != nil {
                if scopes := common.SanitizationScopesFromContext(ctx); len(scopes) > 0 {
                        if res.Metadata == nil {
                                res.Metadata = make(map[string]any)
                        }
                        res.Metadata[ResultSanitizationScopeMetaKey] = scopes
                }
        }
}

type sanitizationMessage struct {
        abstract.Message
        ctx context.Context
}

func (m *sanitizationMessage) Context() context.Context { return m.ctx }

var _ abstract.Dispatcher = (*SanitizationDispatcher)(nil)
