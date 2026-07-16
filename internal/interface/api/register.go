package api

// 05/07/2026 TODO: DO SOMETHING WITH SANITIZATION ERRORS
import (
	"context"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/abstract"
)

const (
	statusOK         = 200
	statusCreated    = 201
	statusNoContent  = 204
	statusNotFound   = 404
	statusTooMany    = 429
)
const (
	msgSessionCreate = "system:auth:session:create"
	msgSessionRefresh = "system:auth:session:refresh"
	msgSessionDelete  = "system:auth:session:delete"
)

type transportMessage struct {
	id      string
	name    string
	ctx     context.Context
	input   *data.Document
	inputCh <-chan *data.Document
	blobCh  <-chan registration.Blob
}

func (m *transportMessage) ID() string                         { return m.id }
func (m *transportMessage) Name() string                       { return m.name }
func (m *transportMessage) Context() context.Context            { return m.ctx }
func (m *transportMessage) Input() *data.Document               { return m.input }
func (m *transportMessage) InputChannel() <-chan *data.Document   { return m.inputCh }
func (m *transportMessage) BlobInputChannel() <-chan registration.Blob { return m.blobCh }

func (o *Interface) installDispatcherRegistrations() {
	o.installRegistrations(o.regs, false)
}

func (o *Interface) installBootstrapSafeRegistrations() {
	o.installRegistrations(o.regs, true)
}

func (o *Interface) installRegistrations(regs []abstract.MessageRegistration, bootstrapOnly bool) {
	for _, reg := range regs {
		if bootstrapOnly && !reg.BootstrapSafe {
			continue
		}
		if reg.Internal {
			continue
		}

		httpMethod := IntentToHTTPMethod(reg.Intent)
		httpPath := DeriveRoute(reg.Name, reg.Input.Arguments)
		pattern := httpMethod + " " + IntentToHTTPPath(reg.Intent, httpPath)

		o.trans.Handle(pattern, o.wrap(func(ctx context.Context, req Request) (Response, error) {
			req = o.injectCookieRefreshToken(req, reg.Name)
			doc := buildDoc(ctx, req, reg.Input)

			msg := &transportMessage{
				id:    abstract.MustNewID(),
				name:  reg.Name,
				ctx:   ctx,
				input: doc,
			}

			var inputCh chan *data.Document
			if reg.Intent == registration.Stream {
				inputCh = make(chan *data.Document)
				msg.inputCh = inputCh
			}

			result, err := o.disp.Send(msg)
			if err != nil {
				resp := Response{}
				resp = o.attachCookieClearingResponse(resp, reg.Name)
				return resp, err
			}

			if reg.Intent == registration.Stream {
				inputCh <- data.MustNewDocument(map[string]any{}, ctx)
			}

			resp := serializeResponse(result, reg.Output, reg.Intent, httpPath)
			resp = o.attachCookieToResponse(resp, result, reg.Name)
			return resp, nil
		}))
	}
}

func buildDoc(ctx context.Context, req Request, input core.Input) *data.Document {
	doc := data.MustNewDocument(map[string]any{}, ctx)

	args := make(map[string]any)
	for name := range input.Arguments {
		if v, ok := req.PathParams[name]; ok {
			args[name] = v
		}
	}
	doc.Set("arguments", args)

	modifiers := make(map[string]any)
	for name := range input.Modifiers {
		if vals, ok := req.Query[name]; ok && len(vals) > 0 {
			modifiers[name] = vals[0]
		}
	}
	doc.Set("modifiers", modifiers)

	if input.Payload != 0 {
		switch input.Payload {
		case definition.FieldTypeBytes:
			doc.Set("payload", req.Body)
		default:
			var body map[string]any
			if len(req.Body) > 0 {
				if err := json.Unmarshal(req.Body, &body); err != nil {
					body = nil
				}
			}
			if body != nil {
				doc.Set("payload", body)
			}
		}
	}

	return doc
}

func serializeResponse(result *registration.Result, output *definition.Schema, intent registration.Verb, httpPath string) Response {
	if result == nil {
		status := statusOK
		if intent == registration.Delete {
			status = statusNoContent
		}
		return Response{Status: status}
	}

	if intent == registration.Stream {
		if result.DocumentChannel != nil {
			streamCh := make(chan any, 64)
			go func() {
				defer close(streamCh)
				for d := range result.DocumentChannel {
					sane, _ := d.Sanitize()
					streamCh <- sane.ToMap()
				}
			}()
			return Response{Status: statusOK, Body: StreamBody(streamCh)}
		}
		if result.BlobChannel != nil {
			streamCh := make(chan any, 64)
			go func() {
				defer close(streamCh)
				for b := range result.BlobChannel {
					streamCh <- map[string]any{"data": b.Data, "content_type": b.ContentType}
				}
			}()
			return Response{Status: statusOK, Body: StreamBody(streamCh)}
		}
	}

	if result.Blob.Data != nil {
		return Response{
			Status:  statusOK,
			Body:    result.Blob.Data,
			Headers: map[string][]string{"Content-Type": {result.Blob.ContentType}},
		}
	}

	if result.DocumentChannel != nil {
		var docs []any
		for d := range result.DocumentChannel {
			sane, _ := d.Sanitize()
			docs = append(docs, sane.ToMap())
		}
		if docs == nil {
			docs = []any{}
		}
		return Response{Status: statusOK, Body: map[string]any{"items": docs}}
	}

	if result.BlobChannel != nil {
		return Response{Status: statusOK}
	}

	if output == nil || len(output.Fields) == 0 {
		status := statusOK
		if intent == registration.Create {
			status = statusCreated
		}
		if intent == registration.Delete {
			status = statusNoContent
		}
		return Response{Status: status}
	}

	for fieldName := range output.Fields {
		switch fieldName {
		case "document":
			if result.Document != nil {
				status := statusOK
				if intent == registration.Create {
					status = statusCreated
				}
				sane, _ := result.Document.Sanitize()
				resp := Response{Status: status, Body: sane}
				if intent == registration.Create {
					if id := result.Document.ID(); id != "" {
						resp.Headers = map[string][]string{
							"Location": {httpPath + "/" + id},
						}
					}
				}
				return resp
			}
		case "documents":
			if result.Documents != nil {
				items := make([]any, 0, len(result.Documents))
				for _, d := range result.Documents {
					sane, _ := d.Sanitize()
					items = append(items, sane.ToMap())
				}
				return Response{Status: statusOK, Body: items}
			}
		case "page":
			if result.Page != nil {
				items := make([]any, 0, len(result.Page.Documents))
				for _, d := range result.Page.Documents {
					sane, _ := d.Sanitize()
					items = append(items, sane.ToMap())
				}
				return Response{
					Status: statusOK,
					Body:   items,
					Page:   result.Page.Pagination,
				}
			}
		}
	}

	status := statusOK
	if intent == registration.Create {
		status = statusCreated
	}
	if intent == registration.Delete {
		status = statusNoContent
	}
	return Response{Status: status}
}

// injectCookieRefreshToken reads the refresh token from the cookie and
// injects it into the request body for refresh and logout endpoints,
// so the handler can find it as a regular payload field.
func (o *Interface) injectCookieRefreshToken(req Request, name string) Request {
	if o.cookieCfg.RefreshName == "" {
		return req
	}
	if name != msgSessionRefresh && name != msgSessionDelete {
		return req
	}
	rt, ok := req.Cookies[o.cookieCfg.RefreshName]
	if !ok || rt == "" {
		return req
	}
	var bodyMap map[string]any
	if len(req.Body) > 0 {
		json.Unmarshal(req.Body, &bodyMap)
	}
	if bodyMap == nil {
		bodyMap = make(map[string]any)
	}
	if _, exists := bodyMap["refresh_token"]; !exists {
		bodyMap["refresh_token"] = rt
		req.Body, _ = json.Marshal(bodyMap)
	}
	return req
}

// attachCookieToResponse sets or clears both access and refresh token cookies
// based on the auth operation.
func (o *Interface) attachCookieToResponse(resp Response, result *registration.Result, name string) Response {
	switch name {
	case msgSessionCreate, msgSessionRefresh:
		at := extractAccessToken(result)
		rt := extractRefreshToken(result)
		if at == "" && rt == "" {
			return resp
		}
		if at != "" && o.cookieCfg.AccessName != "" {
			resp.Cookies = append(resp.Cookies, Cookie{
				Name:     o.cookieCfg.AccessName,
				Value:    at,
				Path:     o.cookieCfg.AccessPath,
				Domain:   o.cookieCfg.Domain,
				Secure:   o.cookieCfg.Secure,
				HTTPOnly: o.cookieCfg.HTTPOnly,
				SameSite: o.cookieCfg.SameSite,
				MaxAge:   int(o.accessTokenTTL.Seconds()),
			})
		}
		if rt != "" && o.cookieCfg.RefreshName != "" {
			resp.Cookies = append(resp.Cookies, Cookie{
				Name:     o.cookieCfg.RefreshName,
				Value:    rt,
				Path:     o.cookieCfg.RefreshPath,
				Domain:   o.cookieCfg.Domain,
				Secure:   o.cookieCfg.Secure,
				HTTPOnly: o.cookieCfg.HTTPOnly,
				SameSite: o.cookieCfg.SameSite,
				MaxAge:   int(o.refreshTokenTTL.Seconds()),
			})
		}

	case msgSessionDelete:
		if o.cookieCfg.AccessName != "" {
			resp.Cookies = append(resp.Cookies, Cookie{
				Name:     o.cookieCfg.AccessName,
				Value:    "",
				Path:     o.cookieCfg.AccessPath,
				Domain:   o.cookieCfg.Domain,
				Secure:   o.cookieCfg.Secure,
				HTTPOnly: o.cookieCfg.HTTPOnly,
				SameSite: o.cookieCfg.SameSite,
				MaxAge:   -1,
			})
		}
		if o.cookieCfg.RefreshName != "" {
			resp.Cookies = append(resp.Cookies, Cookie{
				Name:     o.cookieCfg.RefreshName,
				Value:    "",
				Path:     o.cookieCfg.RefreshPath,
				Domain:   o.cookieCfg.Domain,
				Secure:   o.cookieCfg.Secure,
				HTTPOnly: o.cookieCfg.HTTPOnly,
				SameSite: o.cookieCfg.SameSite,
				MaxAge:   -1,
			})
		}
	}
	return resp
}

// attachCookieClearingResponse clears both cookies for auth operations when
// the handler returns an error (e.g. expired/revoked token).
func (o *Interface) attachCookieClearingResponse(resp Response, name string) Response {
	switch name {
	case msgSessionCreate, msgSessionRefresh, msgSessionDelete:
		if o.cookieCfg.AccessName != "" {
			resp.Cookies = append(resp.Cookies, Cookie{
				Name:   o.cookieCfg.AccessName,
				Value:  "",
				Path:   o.cookieCfg.AccessPath,
				Domain: o.cookieCfg.Domain,
				MaxAge: -1,
			})
		}
		if o.cookieCfg.RefreshName != "" {
			resp.Cookies = append(resp.Cookies, Cookie{
				Name:   o.cookieCfg.RefreshName,
				Value:  "",
				Path:   o.cookieCfg.RefreshPath,
				Domain: o.cookieCfg.Domain,
				MaxAge: -1,
			})
		}
	}
	return resp
}

// extractAccessToken pulls the access token from a registration.Result.
func extractAccessToken(result *registration.Result) string {
	if result == nil || result.Document == nil {
		return ""
	}
	tokenIf := result.Document.GetOr("token", nil)
	tokenMap, ok := tokenIf.(map[string]any)
	if !ok {
		return ""
	}
	at, _ := tokenMap["access"].(string)
	return at
}

// extractRefreshToken pulls the refresh token from a registration.Result.
func extractRefreshToken(result *registration.Result) string {
	if result == nil || result.Document == nil {
		return ""
	}
	tokenIf := result.Document.GetOr("token", nil)
	tokenMap, ok := tokenIf.(map[string]any)
	if !ok {
		return ""
	}
	rt, _ := tokenMap["refresh"].(string)
	return rt
}
