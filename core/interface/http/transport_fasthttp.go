package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type TransportOptions struct {
	Addr           string
	Logger         Logger
	APIPrefix      string
	StaticFS       fs.FS
	AllowedOrigins []string
}

// CORSAllowlist manages allowed origins for CORS, safe for concurrent use.
type CORSAllowlist struct {
	mu      sync.RWMutex
	origins []string
}

func NewCORSAllowlist(origins []string) *CORSAllowlist {
	cp := make([]string, len(origins))
	copy(cp, origins)
	return &CORSAllowlist{origins: cp}
}

func (c *CORSAllowlist) Allowed(origin string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, o := range c.origins {
		if o == origin {
			return true
		}
	}
	return false
}

func (c *CORSAllowlist) Add(origin string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.origins = append(c.origins, origin)
}

func (c *CORSAllowlist) Remove(origin string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, o := range c.origins {
		if o == origin {
			c.origins = append(c.origins[:i], c.origins[i+1:]...)
			return
		}
	}
}

func (c *CORSAllowlist) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := make([]string, len(c.origins))
	copy(cp, c.origins)
	return cp
}

type HTTPTransport struct {
	addr      string
	logger    Logger
	apiPrefix string
	staticFS  fs.FS
	server    *fasthttp.Server
	router    *pathTrie
	corsList  *CORSAllowlist
}

func NewTransport(opts TransportOptions) *HTTPTransport {
	ac := opts.AllowedOrigins
	if ac == nil {
		ac = defaultAllowedOrigins()
	}
	return &HTTPTransport{
		addr:      opts.Addr,
		logger:    opts.Logger,
		apiPrefix: opts.APIPrefix,
		staticFS:  opts.StaticFS,
		router:    newPathTrie(),
		corsList:  NewCORSAllowlist(ac),
	}
}

func defaultAllowedOrigins() []string {
	return []string{}
}

func (t *HTTPTransport) SetAllowedOrigins(origins []string) {
	t.corsList = NewCORSAllowlist(origins)
}

func (t *HTTPTransport) CORSAllowlist() *CORSAllowlist {
	return t.corsList
}

func (t *HTTPTransport) Handle(pattern string, handler abstract.Handler) {
	method, path := splitPattern(pattern)
	t.router.insert(method, path, handler)
}

const (
	// maxRequestBodySize caps a single request body (10 GiB). fasthttp's
	// default is only 4 MiB, which would break resumable-upload chunks.
	maxRequestBodySize = 10 << 30
)

func (t *HTTPTransport) Start() error {
	t.server = &fasthttp.Server{
		Handler: t.serveHTTP,
		// fasthttp treats MaxRequestBodySize <= 0 as its 4 MB default, which
		// would reject every resumable-upload chunk. Use a generous explicit
		// ceiling; individual blob handlers enforce their own per-op limits.
		MaxRequestBodySize: maxRequestBodySize,
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
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			n := runtime.Stack(stack, false)
			if t.logger != nil {
				t.logger.Error("panic recovered in HTTP handler",
					zap.Any("panic", r),
					zap.ByteString("stack", stack[:n]),
					zap.String("method", string(ctx.Method())),
					zap.String("path", string(ctx.Path())),
					zap.String("request_id", string(ctx.Request.Header.Peek("X-Request-ID"))),
				)
			}
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.Response.Header.Set("Content-Type", "application/json")
			_, _ = ctx.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
		}
	}()

	t.cors(ctx)
	if string(ctx.Method()) == "OPTIONS" {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	correlationID(ctx)

	method := string(ctx.Method())
	path := string(ctx.Path())

	handler, params, ok := t.router.lookup(method, path)
	if ok {
		cookies := make(map[string]string)
		ctx.Request.Header.VisitAllCookie(func(k, v []byte) {
			cookies[string(k)] = string(v)
		})

		req := abstract.Request{
			Operation:  method + " " + path,
			Body:       ctx.Request.Body(),
			PathParams: params,
			Query:      queryArgsToMap(ctx.QueryArgs()),
			Headers:    headersToMap(&ctx.Request.Header),
			Cookies:    cookies,
			ClientIP:   clientIP(ctx),
			UserAgent:  string(ctx.UserAgent()),
			RequestID:  string(ctx.Request.Header.Peek("X-Request-ID")),
		}
		resp, err := handler(ctx, req)
		if err != nil {
			t.writeError(ctx, err, resp)
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

func (t *HTTPTransport) cors(ctx *fasthttp.RequestCtx) {
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		return
	}

	host := string(ctx.Request.Header.Peek("Host"))
	sameOrigin := host != "" && (origin == "http://"+host || origin == "https://"+host)

	allowed := sameOrigin || t.corsList.Allowed(origin)
	if !allowed {
		return
	}

	ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, x-api-key")
	ctx.Response.Header.Set("Vary", "Origin")
	if sameOrigin {
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	}
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

// ── Response envelope types ─────────────────────────────────────────────────

type responseMeta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request,omitempty"`
	Page      any    `json:"page,omitempty"`
}

type responseErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ── Response writing ───────────────────────────────────────────────────────

func (t *HTTPTransport) writeSuccess(ctx *fasthttp.RequestCtx, resp abstract.Response) {
	if resp.Status == 0 {
		resp.Status = fasthttp.StatusOK
	}

	for _, c := range resp.Cookies {
		ctx.Response.Header.SetCookie(toFasthttpCookie(c))
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

	meta := buildResponseMeta(resp, ctx)
	json.NewEncoder(ctx).Encode(map[string]any{
		"data":     resp.Body,
		"metadata": meta,
	})
}

func buildResponseMeta(resp abstract.Response, ctx *fasthttp.RequestCtx) map[string]any {
	meta := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"request":   string(ctx.Request.Header.Peek("X-Request-ID")),
	}
	if resp.Page != nil {
		meta["page"] = resp.Page
	}
	for k, v := range resp.Metadata {
		meta[k] = v
	}
	return meta
}

func (t *HTTPTransport) writeError(ctx *fasthttp.RequestCtx, err error, resp abstract.Response) {
	ctx.SetContentType("application/json")

	for k, vals := range resp.Headers {
		for _, v := range vals {
			ctx.Response.Header.Add(k, v)
		}
	}

	for _, c := range resp.Cookies {
		ctx.Response.Header.SetCookie(toFasthttpCookie(c))
	}

	status := fasthttp.StatusInternalServerError
	var sysErr *common.SystemError

	if errors.As(err, &sysErr) {
		status = systemErrorToStatus(sysErr)
	} else {
		sysErr = common.NewSystemError("INTERNAL_ERROR", err.Error())
	}

	issue := sysErr.ToIssue()

	meta := buildResponseMeta(resp, ctx)

	ctx.SetStatusCode(status)
	json.NewEncoder(ctx).Encode(map[string]any{
		"error": responseErrorBody{
			Code:    issue.Code,
			Message: issue.Message,
			Details: issue.Cause,
		},
		"metadata": meta,
	})

	if t.logger != nil {
		t.logger.Warn("request error",
			zap.String("code", issue.Code),
			zap.Int("status", status),
			zap.String("error", issue.Message),
			zap.String("request_id", string(ctx.Request.Header.Peek("X-Request-ID"))),
		)
	}
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

func ExtractPathParams(pattern, path string) map[string]string {
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
	"ERR_ACCESS_DENIED":   fasthttp.StatusForbidden,
	"NOT_FOUND":           fasthttp.StatusNotFound,
	"ALREADY_EXISTS":      fasthttp.StatusConflict,
	"VALIDATION_ERROR":    fasthttp.StatusBadRequest,
	"INVALID_REQUEST":     fasthttp.StatusBadRequest,
	"UNAUTHORIZED":        fasthttp.StatusUnauthorized,
	"INVALID_CREDENTIALS": fasthttp.StatusUnauthorized,
	"EMAIL_EXISTS":        fasthttp.StatusConflict,
	"USER_DISABLED":       fasthttp.StatusForbidden,
	"FORBIDDEN":           fasthttp.StatusForbidden,
	"MISSING_PARAM":       fasthttp.StatusBadRequest,
	"INVALID_QDSL":        fasthttp.StatusBadRequest,
	"DOCUMENT_REQUIRED":   fasthttp.StatusBadRequest,
	"PARSE_DOCUMENT":      fasthttp.StatusBadRequest,
	"SCHEMA_REQUIRED":     fasthttp.StatusBadRequest,
	"SCHEMA_MISSING_NAME": fasthttp.StatusBadRequest,
	"COLLECTION_EXISTS":   fasthttp.StatusConflict,
	"RESERVED_NAME":       fasthttp.StatusConflict,
	"AUTH_REQUIRED":       fasthttp.StatusUnauthorized,
	"DOCUMENT_NOT_FOUND":  fasthttp.StatusNotFound,
	"NOT_IMPLEMENTED":     fasthttp.StatusNotImplemented,
	"SERVICE_UNAVAILABLE": fasthttp.StatusServiceUnavailable,
	"RATE_LIMITED":        fasthttp.StatusTooManyRequests,
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

func SystemErrorToStatus(err *common.SystemError) int {
	return systemErrorToStatus(err)
}
