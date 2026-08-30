package http

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

const (
	statusOK        = 200
	statusCreated   = 201
	statusAccepted  = 202
	statusNoContent = 204
	statusNotFound  = 404
	statusTooMany   = 429
)

const (
	msgSessionCreate = "system:auth:session:create"
	msgSessionDelete = "system:auth:session:delete"

	msgTokenElevate    = "system:auth:token:elevate"
	msgPasswordConfirm = "system:auth:password:confirm"
)

// authRateLimitedMessages lists every password-checking message that is
// subject to the per-IP brute-force limiter in authMiddleware: login, token
// elevation (mints a live 5-minute API key) and password-reset confirmation.
// Both the message name and the derived route pattern are registered so the
// gate works regardless of how the transport fills req.Operation.
var authRateLimitedMessages = map[string]struct{}{
	msgSessionCreate:   {},
	msgTokenElevate:    {},
	msgPasswordConfirm: {},
}

func (o *Interface) installDispatcherRegistrations() {
	for _, reg := range o.regs {
		if reg.Internal {
			continue
		}
		if err := o.installRegistration(reg); err != nil {
			o.opts.Logger.Error("failed to install registration", zap.String("name", reg.Name), zap.Error(err))
		}
	}
}

func (o *Interface) installBootstrapSafeRegistrations() {
	for _, reg := range o.regs {
		if reg.Internal {
			continue
		}
		if !reg.BootstrapSafe {
			continue
		}
		if err := o.installRegistration(reg); err != nil {
			o.opts.Logger.Error("failed to install bootstrap-safe registration", zap.String("name", reg.Name), zap.Error(err))
		}
	}
}

func (o *Interface) installRegistration(reg abstract.MessageRegistration) error {
	httpMethod := IntentToHTTPMethod(reg.Intent)
	httpPath := DeriveRoute(reg.Name, reg.Input.Arguments())
	if o.opts.APIPrefix != "" {
		httpPath = o.opts.APIPrefix + httpPath
	}
	pattern := httpMethod + " " + IntentToHTTPPath(reg.Intent, httpPath)

	var pool *document.DocumentPool
	if reg.Input.Schema != nil {
		p, err := document.NewDocumentPool(reg.Input.Schema)
		if err != nil {
			return fmt.Errorf("install %s: input pool: %w", reg.Name, err)
		}
		pool = p
	}

	if _, ok := o.noRefreshCommands[reg.Name]; ok {
		o.noRefreshOps[pattern] = struct{}{}
	}

	if _, ok := authRateLimitedMessages[reg.Name]; ok {
		// Both forms registered: message name (CLI/internal transports set
		// req.Operation to the message name) and route pattern (the HTTP
		// transport sets req.Operation to "POST /api/...").
		o.authLimitedOps[reg.Name] = struct{}{}
		o.authLimitedOps[pattern] = struct{}{}
	}

	// S-8: blob payloads are the only routes allowed bodies larger than
	// defaultBodyLimit; everything else stays at the small default.
	if strings.HasPrefix(reg.Name, "system:blobs:") {
		if ht, ok := o.trans.(*HTTPTransport); ok {
			ht.SetBodyLimit(pattern, maxRequestBodySize)
		}
	}

	o.trans.Handle(pattern, o.wrap(func(ctx context.Context, req Request) (Response, error) {
		doc, err := buildDoc(ctx, req, reg.Input, pool)
		if err != nil {
			return Response{}, common.NewSystemError("VALIDATION_ERROR", "input validation failed").WithIssues([]common.Issue{{
				Code:    "INVALID_INPUT",
				Message: err.Error(),
			}})
		}

		if issues, ok := dispatch.ValidateInputDocument(reg.Input.Schema, doc); !ok {
			return Response{}, common.NewSystemError("VALIDATION_ERROR", "input validation failed").WithIssues(issues)
		}

		if reg.Input.ResourceIDField != "" && doc != nil {
			if rid, ok := doc.GetOr("arguments."+reg.Input.ResourceIDField, "").(string); ok && rid != "" {
				ctx = runtimecontext.ContextWithResourceID(ctx, rid)
			}
		}

		idempotencyKey := ""
		if keys := req.Headers["Idempotency-Key"]; len(keys) > 0 && keys[0] != "" {
			idempotencyKey = keys[0]
			ctx = runtimecontext.ContextWithTraceID(ctx, idempotencyKey)
		}

		if reg.FireAndForget {
			id, err := dispatch.Enqueue(ctx, o.disp, dispatch.DispatchInput{
				Name:     reg.Name,
				Context:  ctx,
				ID:       idempotencyKey,
				Document: doc,
				Intent:   reg.Intent,
			})
			if err != nil {
				resp := o.attachCookieClearingResponse(Response{}, reg.Name)
				var rle *runtime.RateLimitError
				if errors.As(err, &rle) {
					if resp.Headers == nil {
						resp.Headers = map[string][]string{}
					}
					resp.Headers["Retry-After"] = []string{strconv.Itoa(rle.RetryAfter())}
					resp.Headers["X-RateLimit-Remaining"] = []string{"0"}
				}
				return resp, err
			}
			return acceptedResponse(id), nil
		}

		result, err := dispatch.Dispatch(ctx, o.disp, dispatch.DispatchInput{
			Name:     reg.Name,
			Context:  ctx,
			ID:       idempotencyKey,
			Document: doc,
			Intent:   reg.Intent,
		})
		if err != nil {
			resp := o.attachCookieClearingResponse(Response{}, reg.Name)
			var rle *runtime.RateLimitError
			if errors.As(err, &rle) {
				if resp.Headers == nil {
					resp.Headers = map[string][]string{}
				}
				resp.Headers["Retry-After"] = []string{strconv.Itoa(rle.RetryAfter())}
				resp.Headers["X-RateLimit-Remaining"] = []string{"0"}
			}
			return resp, err
		}

		defer result.Release()
		resp := serializeResponse(ctx, result, reg.Output, reg.Intent, httpPath)
		resp = o.attachCookieToResponse(resp, result, reg.Name)
		return resp, nil
	}))
	return nil
}

func buildDoc(ctx context.Context, req Request, input runtime.Input, pool *document.DocumentPool) (data.Documenter, error) {
	if pool == nil {
		return nil, nil
	}
	return BuildInputDocument(pool, input, req)
}

func serializeResponse(ctx context.Context, result *abstract.Result, output *definition.Schema, intent abstract.Verb, httpPath string) Response {
	if result == nil {
		return emptyResponse(intent)
	}

	if streamResp, ok := serializeStreamResult(ctx, result); ok {
		copyMeta(&streamResp, result)
		return streamResp
	}

	if result.Blob.Data != nil {
		resp := blobResponse(result.Blob)
		copyMeta(&resp, result)
		return resp
	}

	if result.DocumentChannel != nil {
		return drainChannelResponse(result.DocumentChannel)
	}

	if result.BlobChannel != nil {
		return Response{Status: statusOK}
	}

	if output == nil || len(output.Fields) == 0 {
		return emptyResponse(intent)
	}

	if resp, ok := serializeOutputField(result, output, intent, httpPath); ok {
		resp.Metadata = result.Metadata
		promoteMetadataHeaders(&resp)
		return resp
	}

	return emptyResponse(intent)
}

// @note #mem-20260821-003 issue resolved status=open priority=P1 tags=#memory,#goroutine : Stream channel buffer size causes goroutine leaks
// Resolved: streamChannel in core/interface/http/register.go now selects on the request context while draining the source channel and while forwarding to the response stream. On client disconnect (ctx done) the producer goroutine returns instead of blocking forever on a full buffer, eliminating the leak.
//
// streamChannel (line 171) creates a channel with buffer size 64. If the
// consumer stops reading (e.g., client disconnects), the producer goroutine
// blocks indefinitely, causing a goroutine leak.
//
// For IoT/HFT:
// - IoT devices may disconnect frequently
// - HFT systems need clean resource management
// - Goroutine leaks accumulate over time
//
// Resolution:
// 1. Use a select with context cancellation when sending
// 2. Add a timeout or context to the producer goroutine
// 3. Consider using a bounded buffer with overflow handling
// 4. Monitor goroutine count for leak detection
func streamChannel[T any](ctx context.Context, src <-chan T, transform func(T) any) (Response, bool) {
	if src == nil {
		return Response{}, false
	}
	streamCh := make(chan any, 64)
	go func() {
		defer close(streamCh)
		for {
			select {
			case v, ok := <-src:
				if !ok {
					return
				}
				select {
				case streamCh <- transform(v):
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				// Consumer went away (client disconnect): stop draining so
				// this goroutine does not leak blocked on a full buffer.
				return
			}
		}
	}()
	return Response{Status: statusOK, Body: StreamBody(streamCh)}, true
}

func serializeStreamResult(ctx context.Context, result *abstract.Result) (Response, bool) {
	if resp, ok := streamChannel(ctx, result.DocumentChannel, func(d *document.Document) any {
		sane, _ := d.Sanitize()
		return sane
	}); ok {
		return resp, true
	}
	return streamChannel(ctx, result.BlobChannel, func(b abstract.Blob) any {
		return map[string]any{"data": b.Data, "content_type": b.ContentType}
	})
}

func blobResponse(blob abstract.Blob) Response {
	return Response{
		Status:  statusOK,
		Body:    blob.Data,
		Headers: map[string][]string{"Content-Type": {blob.ContentType}},
	}
}

func drainChannelResponse(docCh <-chan *document.Document) Response {
	var docs []data.Documenter
	for d := range docCh {
		sane, _ := d.Sanitize()
		docs = append(docs, sane)
	}
	if docs == nil {
		docs = []data.Documenter{}
	}
	return Response{Status: statusOK, Body: map[string]any{"items": docs}}
}

func emptyResponse(intent abstract.Verb) Response {
	status := statusOK
	if intent == abstract.Create {
		status = statusCreated
	}
	if intent == abstract.Delete {
		status = statusNoContent
	}
	return Response{Status: status}
}

// acceptedResponse is the envelope for fire-and-forget dispatches: 202
// Accepted plus the correlation ID of the accepted message.
func acceptedResponse(id string) Response {
	return Response{
		Status: statusAccepted,
		Body: map[string]any{
			"data": map[string]any{
				"id":     id,
				"status": "accepted",
			},
			"metadata": map[string]any{},
		},
	}
}

func createStatus(intent abstract.Verb) int {
	if intent == abstract.Create {
		return statusCreated
	}
	return statusOK
}

func locationHeader(intent abstract.Verb, doc data.Documenter, httpPath string) map[string][]string {
	if intent != abstract.Create {
		return nil
	}
	if id := doc.ID(); id != "" {
		return map[string][]string{"Location": {httpPath + "/" + id}}
	}
	return nil
}

func serializeOutputField(result *abstract.Result, output *definition.Schema, intent abstract.Verb, httpPath string) (Response, bool) {
	status := createStatus(intent)

	// The concrete result payload is the ground truth for the response body,
	// not the output schema's declared shape: the schema is introspection only
	// and may be a flat model schema (service codegen) or a wrapper envelope
	// (feature DTOs). Serialize whichever result field the handler populated.
	if result.Document != nil {
		sane, _ := result.Document.Sanitize()
		return Response{
			Status:  status,
			Body:    sane,
			Headers: locationHeader(intent, result.Document, httpPath),
		}, true
	}
	if result.Documents != nil {
		items := make([]data.Documenter, 0, len(result.Documents))
		for _, d := range result.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		return Response{Status: statusOK, Body: items}, true
	}
	if result.Page != nil {
		items := make([]data.Documenter, 0, len(result.Page.Documents))
		for _, d := range result.Page.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		return Response{
			Status: statusOK,
			Body:   items,
			Page:   result.Page.Pagination,
		}, true
	}
	return Response{}, false
}

func copyMeta(resp *Response, result *abstract.Result) {
	resp.Metadata = result.Metadata
	promoteMetadataHeaders(resp)
}

var metaHeaderPromotions = map[string]string{
	"rates": "X-RateLimit",
}

func promoteMetadataHeaders(resp *Response) {
	for key, headerPrefix := range metaHeaderPromotions {
		v, ok := resp.Metadata[key]
		if !ok {
			continue
		}
		m, ok := v.(*runtime.RateLimitMeta)
		if !ok {
			continue
		}
		if resp.Headers == nil {
			resp.Headers = map[string][]string{}
		}
		resp.Headers[headerPrefix+"-Remaining"] = []string{strconv.Itoa(m.Remaining)}
		resp.Headers[headerPrefix+"-Limit"] = []string{strconv.Itoa(m.Limit)}
		resp.Headers[headerPrefix+"-Reset"] = []string{strconv.FormatInt(m.ResetAt, 10)}
	}
}

func extractSessionToken(result *abstract.Result) string {
	if result == nil {
		return ""
	}
	return result.SessionToken
}

func (o *Interface) attachCookieToResponse(resp Response, result *abstract.Result, name string) Response {
	switch name {
	case msgSessionCreate:
		token := extractSessionToken(result)
		if token == "" {
			return resp
		}
		resp.Cookies = append(resp.Cookies, newSessionCookie(
			o.cookieCfg.SessionName, token, o.cookieCfg.SessionPath, o.sessionTTL, o.cookieCfg,
		))

	case msgSessionDelete:
		if o.cookieCfg.SessionName != "" {
			resp.Cookies = append(resp.Cookies, clearSessionCookie(
				o.cookieCfg.SessionName, o.cookieCfg.SessionPath, o.cookieCfg,
			))
		}
	}
	return resp
}

func (o *Interface) attachCookieClearingResponse(resp Response, name string) Response {
	if name == msgSessionCreate || name == msgSessionDelete {
		if o.cookieCfg.SessionName != "" {
			resp.Cookies = append(resp.Cookies, clearSessionCookie(
				o.cookieCfg.SessionName, o.cookieCfg.SessionPath, o.cookieCfg,
			))
		}
	}
	return resp
}
