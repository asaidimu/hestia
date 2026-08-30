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

func NewSanitizationDispatcher(next abstract.Dispatcher) *SanitizationDispatcher {
	return &SanitizationDispatcher{next: next}
}

func (d *SanitizationDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &SanitizationDispatcher{next: next}
}

func (d *SanitizationDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	scope := featureScope(msg.Name())

	// Set scope on message context so downstream handlers/dispatchers see it.
	var enrichedMsg abstract.Message
	if scope != "" {
		tctx := common.ContextWithSanitizationScope(msg.Context(), scope)
		enrichedMsg = &sanitizationMessage{Message: msg, ctx: tctx}
	} else {
		enrichedMsg = msg
	}

	return d.next.Send(ctx, enrichedMsg, func(cctx context.Context, res *abstract.Result, handlerErr error) {
		if res != nil && handlerErr == nil {
			sanitizeResult(res, cctx)
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
}

type sanitizationMessage struct {
	abstract.Message
	ctx context.Context
}

func (m *sanitizationMessage) Context() context.Context { return m.ctx }

var _ abstract.Dispatcher = (*SanitizationDispatcher)(nil)
