package httpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/abstract"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type TransportOptions struct {
	Addr      string
	Logger    Logger
	APIPrefix string
	StaticFS  fs.FS
}

type HTTPTransport struct {
	addr      string
	logger    Logger
	apiPrefix string
	staticFS  fs.FS
	server    *fasthttp.Server
	routes    []routeEntry
}

type routeEntry struct {
	method  string
	prefix  string
	handler abstract.Handler
}

func NewTransport(opts TransportOptions) *HTTPTransport {
	return &HTTPTransport{
		addr:      opts.Addr,
		logger:    opts.Logger,
		apiPrefix: opts.APIPrefix,
		staticFS:  opts.StaticFS,
	}
}

func (t *HTTPTransport) Handle(pattern string, handler abstract.Handler) {
	method, path := splitPattern(pattern)
	t.routes = append(t.routes, routeEntry{method: method, prefix: path, handler: handler})
}

func (t *HTTPTransport) Start() error {
	t.server = &fasthttp.Server{
		Handler: t.serveHTTP,
	}
	return t.server.ListenAndServe(t.addr)
}

func (t *HTTPTransport) Shutdown(ctx context.Context) error {
	if t.server == nil {
		return nil
	}
	return t.server.Shutdown()
}

func (t *HTTPTransport) serveHTTP(ctx *fasthttp.RequestCtx) {
	cors(ctx)
	if string(ctx.Method()) == "OPTIONS" {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	correlationID(ctx)

	method := string(ctx.Method())
	path := string(ctx.Path())

	for _, route := range t.routes {
		if route.method != "" && route.method != method {
			continue
		}
		params := extractPathParams(route.prefix, path)
		if params == nil && route.prefix != path {
			continue
		}

		cookies := make(map[string]string)
		ctx.Request.Header.VisitAllCookie(func(k, v []byte) {
			cookies[string(k)] = string(v)
		})

		req := abstract.Request{
			Operation:  route.method + " " + route.prefix,
			Body:       ctx.Request.Body(),
			PathParams: params,
			Query:      queryArgsToMap(ctx.QueryArgs()),
			Headers:    headersToMap(&ctx.Request.Header),
			Cookies:    cookies,
			ClientIP:   clientIP(ctx),
			UserAgent:  string(ctx.UserAgent()),
			RequestID:  string(ctx.Request.Header.Peek("X-Request-ID")),
		}
		resp, err := route.handler(ctx, req)
		if err != nil {
			t.writeError(ctx, err, resp.Cookies)
			return
		}
		t.writeSuccess(ctx, resp)
		return
	}

	if t.staticFS != nil {
		if t.apiPrefix != "" && strings.HasPrefix(path, t.apiPrefix) {
			t.writeJSON(ctx, fasthttp.StatusNotFound, map[string]any{
				"error": map[string]any{"code": "NOT_FOUND", "message": "no matching route"},
			})
			return
		}
		t.serveStatic(ctx, path)
		return
	}

	t.writeJSON(ctx, fasthttp.StatusNotFound, map[string]any{
		"error": map[string]any{"code": "NOT_FOUND", "message": "no matching route"},
	})
}

func cors(ctx *fasthttp.RequestCtx) {
	origin := ctx.Request.Header.Peek("Origin")
	if len(origin) == 0 {
		origin = []byte("*")
	}
	ctx.Response.Header.Set("Access-Control-Allow-Origin", string(origin))
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, x-api-key")
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.Set("Vary", "Origin")
}

func correlationID(ctx *fasthttp.RequestCtx) {
	id := ctx.Request.Header.Peek("X-Request-ID")
	if len(id) == 0 {
		id = ctx.Request.Header.Peek("X-Correlation-ID")
	}
	if len(id) == 0 {
		id = []byte(randomID())
	}
	ctx.Response.Header.Set("X-Request-ID", string(id))
	ctx.Request.Header.Set("X-Request-ID", string(id))
}

// ── Response writing ───────────────────────────────────────────────────────

func (t *HTTPTransport) writeSuccess(ctx *fasthttp.RequestCtx, resp abstract.Response) {
	if resp.Status == 0 {
		resp.Status = fasthttp.StatusOK
	}

	for _, c := range resp.Cookies {
		fc := fasthttp.Cookie{}
		fc.SetKey(c.Name)
		fc.SetValue(c.Value)
		fc.SetPath(c.Path)
		fc.SetDomain(c.Domain)
		fc.SetMaxAge(c.MaxAge)
		fc.SetSecure(c.Secure)
		fc.SetHTTPOnly(c.HTTPOnly)
		fc.SetSameSite(mapSameSite(c.SameSite))
		ctx.Response.Header.SetCookie(&fc)
	}

	for k, vals := range resp.Headers {
		for _, v := range vals {
			ctx.Response.Header.Add(k, v)
		}
	}

	if raw, ok := resp.Body.([]byte); ok {
		if len(ctx.Response.Header.ContentType()) == 0 {
			ctx.SetContentType("application/octet-stream")
		}
		ctx.SetStatusCode(resp.Status)
		ctx.Write(raw)
		return
	}

	if stream, ok := resp.Body.(abstract.StreamBody); ok {
		if resp.Status == 0 {
			resp.Status = fasthttp.StatusOK
		}
		ctx.Response.Header.SetContentType("text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.SetStatusCode(resp.Status)
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			for data := range stream {
				jsonBytes, err := json.Marshal(map[string]any{"data": data})
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
				if err := w.Flush(); err != nil {
					return
				}
			}
		})
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(resp.Status)

	if resp.Status == fasthttp.StatusNoContent {
		return
	}

	meta := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"request":   ctx.Request.Header.Peek("X-Request-ID"),
	}
	if resp.Page != nil {
		meta["page"] = resp.Page
	}

	json.NewEncoder(ctx).Encode(map[string]any{
		"data":     resp.Body,
		"metadata": meta,
	})
}

func (t *HTTPTransport) writeError(ctx *fasthttp.RequestCtx, err error, cookies []abstract.Cookie) {
	ctx.SetContentType("application/json")

	for _, c := range cookies {
		fc := fasthttp.Cookie{}
		fc.SetKey(c.Name)
		fc.SetValue(c.Value)
		fc.SetPath(c.Path)
		fc.SetDomain(c.Domain)
		fc.SetMaxAge(c.MaxAge)
		fc.SetSecure(c.Secure)
		fc.SetHTTPOnly(c.HTTPOnly)
		fc.SetSameSite(mapSameSite(c.SameSite))
		ctx.Response.Header.SetCookie(&fc)
	}

	status := fasthttp.StatusInternalServerError
	var sysErr *common.SystemError

	if errors.As(err, &sysErr) {
		status = systemErrorToStatus(sysErr)
	} else {
		sysErr = common.NewSystemError("INTERNAL_ERROR", err.Error())
	}

	issue := sysErr.ToIssue()

	ctx.SetStatusCode(status)
	json.NewEncoder(ctx).Encode(map[string]any{
		"error": map[string]any{
			"code":    issue.Code,
			"message": issue.Message,
			"details": issue.Cause,
		},
		"metadata": map[string]any{
			"timestamp": time.Now().Format(time.RFC3339),
			"request":   ctx.Request.Header.Peek("X-Request-ID"),
		},
	})
}

// ── Static file serving ──────────────────────────────────────────────────────

func (t *HTTPTransport) serveStatic(ctx *fasthttp.RequestCtx, path string) {
	clean := strings.TrimPrefix(path, "/")
	if clean == "" {
		clean = "index.html"
	}

	data, err := fs.ReadFile(t.staticFS, clean)
	if err != nil {
		// SPA fallback — serve index.html
		index, err := fs.ReadFile(t.staticFS, "index.html")
		if err != nil {
			t.writeJSON(ctx, fasthttp.StatusNotFound, map[string]any{
				"error": map[string]any{"code": "NOT_FOUND", "message": "not found"},
			})
			return
		}
		ct := mime.TypeByExtension(filepath.Ext(clean))
		if ct == "" {
			ct = "text/html"
		}
		ctx.SetContentType(ct)
		ctx.Write(index)
		return
	}

	ctx.SetContentType(mimeType(clean))
	ctx.Write(data)
}

func (t *HTTPTransport) writeJSON(ctx *fasthttp.RequestCtx, status int, v any) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(v)
}

func mimeType(name string) string {
	ct := mime.TypeByExtension(filepath.Ext(name))
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func splitPattern(pattern string) (string, string) {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ' ' {
			return pattern[:i], pattern[i+1:]
		}
	}
	return "", pattern
}

func extractPathParams(pattern, path string) map[string]string {
	var patternParts, pathParts []string
	start := 0
	for i := 0; i <= len(pattern); i++ {
		if i == len(pattern) || pattern[i] == '/' {
			if i > start {
				patternParts = append(patternParts, pattern[start:i])
			}
			start = i + 1
		}
	}
	start = 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				pathParts = append(pathParts, path[start:i])
			}
			start = i + 1
		}
	}
	if len(patternParts) != len(pathParts) {
		return nil
	}
	params := make(map[string]string)
	for i, pp := range patternParts {
		if len(pp) > 2 && pp[0] == '{' && pp[len(pp)-1] == '}' {
			params[pp[1:len(pp)-1]] = pathParts[i]
		} else if pp != pathParts[i] {
			return nil
		}
	}
	return params
}

func clientIP(ctx *fasthttp.RequestCtx) string {
	if fwd := ctx.Request.Header.Peek("X-Forwarded-For"); len(fwd) > 0 {
		return string(fwd)
	}
	if realIP := ctx.Request.Header.Peek("X-Real-IP"); len(realIP) > 0 {
		return string(realIP)
	}
	return ctx.RemoteAddr().String()
}

func queryArgsToMap(qa *fasthttp.Args) map[string][]string {
	m := make(map[string][]string)
	qa.VisitAll(func(k, v []byte) {
		key := string(k)
		m[key] = append(m[key], string(v))
	})
	return m
}

func headersToMap(h *fasthttp.RequestHeader) map[string][]string {
	m := make(map[string][]string)
	h.VisitAll(func(k, v []byte) {
		key := string(k)
		m[key] = append(m[key], string(v))
	})
	return m
}

func mapSameSite(s abstract.SameSite) fasthttp.CookieSameSite {
	switch s {
	case abstract.SameSiteLaxMode:
		return fasthttp.CookieSameSiteLaxMode
	case abstract.SameSiteNoneMode:
		return fasthttp.CookieSameSiteNoneMode
	default:
		return fasthttp.CookieSameSiteStrictMode
	}
}

func randomID() string {
	return uuid.Must(uuid.NewV7()).String()
}

var codeToStatus = map[string]int{
	"ERR_ACCESS_DENIED":    fasthttp.StatusForbidden,
	"NOT_FOUND":            fasthttp.StatusNotFound,
	"ALREADY_EXISTS":       fasthttp.StatusConflict,
	"VALIDATION_ERROR":     fasthttp.StatusBadRequest,
	"INVALID_REQUEST":      fasthttp.StatusBadRequest,
	"UNAUTHORIZED":         fasthttp.StatusUnauthorized,
	"INVALID_CREDENTIALS":  fasthttp.StatusUnauthorized,
	"EMAIL_EXISTS":         fasthttp.StatusConflict,
	"USER_DELETED":         fasthttp.StatusGone,
	"FORBIDDEN":            fasthttp.StatusForbidden,
	"MISSING_PARAM":        fasthttp.StatusBadRequest,
	"INVALID_QDSL":         fasthttp.StatusBadRequest,
	"DOCUMENT_REQUIRED":    fasthttp.StatusBadRequest,
	"PARSE_DOCUMENT":       fasthttp.StatusBadRequest,
	"SCHEMA_REQUIRED":      fasthttp.StatusBadRequest,
	"SCHEMA_MISSING_NAME":  fasthttp.StatusBadRequest,
	"COLLECTION_EXISTS":    fasthttp.StatusConflict,
	"RESERVED_NAME":        fasthttp.StatusConflict,
	"AUTH_REQUIRED":        fasthttp.StatusUnauthorized,
	"DOCUMENT_NOT_FOUND":   fasthttp.StatusNotFound,
	"NOT_IMPLEMENTED":      fasthttp.StatusNotImplemented,
	"SERVICE_UNAVAILABLE":  fasthttp.StatusServiceUnavailable,
}

func codeToStatusFn(code string) int {
	if s, ok := codeToStatus[code]; ok {
		return s
	}
	return fasthttp.StatusInternalServerError
}

func systemErrorToStatus(err *common.SystemError) int {
	return codeToStatusFn(err.Code)
}
