package http

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

const (
	statusOK        = 200
	statusCreated   = 201
	statusNoContent = 204
	statusNotFound  = 404
	statusTooMany   = 429
)

const (
	msgSessionCreate = "system:auth:session:create"
	msgSessionDelete = "system:auth:session:delete"
)

func (o *Interface) installDispatcherRegistrations() {
	for _, reg := range o.regs {
		if reg.Internal {
			continue
		}
		o.installRegistration(reg)
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
		o.installRegistration(reg)
	}
}

func (o *Interface) installRegistration(reg abstract.MessageRegistration) {
	httpMethod := IntentToHTTPMethod(reg.Intent)
	httpPath := DeriveRoute(reg.Name, reg.Input.Arguments)
	if o.opts.APIPrefix != "" {
		httpPath = o.opts.APIPrefix + httpPath
	}
	pattern := httpMethod + " " + IntentToHTTPPath(reg.Intent, httpPath)

	var pool *document.DocumentPool
	if reg.Input.Schema != nil {
		p, err := document.NewDocumentPool(reg.Input.Schema)
		if err != nil {
			panic(fmt.Sprintf("install %s: input pool: %v", reg.Name, err))
		}
		pool = p
	}

	if _, ok := o.noRefreshCommands[reg.Name]; ok {
		o.noRefreshOps[pattern] = struct{}{}
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

		result, err := dispatch.Dispatch(o.disp, dispatch.DispatchInput{
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

		resp := serializeResponse(result, reg.Output, reg.Intent, httpPath)
		resp = o.attachCookieToResponse(resp, result, reg.Name)
		return resp, nil
	}))
}

func buildDoc(ctx context.Context, req Request, input runtime.Input, pool *document.DocumentPool) (data.Documenter, error) {
	if pool == nil {
		return nil, nil
	}
	return BuildInputDocument(pool, input, req)
}

func serializeResponse(result *abstract.Result, output *definition.Schema, intent abstract.Verb, httpPath string) Response {
	if result == nil {
		return emptyResponse(intent)
	}

	if streamResp, ok := serializeStreamResult(result); ok {
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

func streamChannel[T any](src <-chan T, transform func(T) any) (Response, bool) {
	if src == nil {
		return Response{}, false
	}
	streamCh := make(chan any, 64)
	go func() {
		defer close(streamCh)
		for v := range src {
			streamCh <- transform(v)
		}
	}()
	return Response{Status: statusOK, Body: StreamBody(streamCh)}, true
}

func serializeStreamResult(result *abstract.Result) (Response, bool) {
	if resp, ok := streamChannel(result.DocumentChannel, func(d data.Documenter) any {
		sane, _ := d.Sanitize()
		return sane
	}); ok {
		return resp, true
	}
	return streamChannel(result.BlobChannel, func(b abstract.Blob) any {
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

func drainChannelResponse(docCh <-chan data.Documenter) Response {
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

func fieldExists(output *definition.Schema, name string) bool {
	for _, f := range output.Fields {
		if string(f.Name) == name {
			return true
		}
	}
	return false
}

func serializeOutputField(result *abstract.Result, output *definition.Schema, intent abstract.Verb, httpPath string) (Response, bool) {
	status := createStatus(intent)

	if result.Document != nil && fieldExists(output, "document") {
		sane, _ := result.Document.Sanitize()
		return Response{
			Status:  status,
			Body:    sane,
			Headers: locationHeader(intent, result.Document, httpPath),
		}, true
	}
	if result.Documents != nil && fieldExists(output, "documents") {
		items := make([]data.Documenter, 0, len(result.Documents))
		for _, d := range result.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		return Response{Status: statusOK, Body: items}, true
	}
	if result.Page != nil && fieldExists(output, "page") {
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
