package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net"
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
	hestiaruntime "github.com/asaidimu/hestia/core/runtime"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type TransportOptions struct {
	Addr             string
	Logger           Logger
	APIPrefix        string
	StaticFS         fs.FS
	AllowedOrigins   []string
	TrustedProxyHops int
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
	addr             string
	logger           Logger
	apiPrefix        string
	staticFS         fs.FS
	server           *fasthttp.Server
	router           *pathTrie
	corsList         *CORSAllowlist
	trustedProxyHops int
	bodyLimits       map[string]int64
}

func NewTransport(opts TransportOptions) *HTTPTransport {
	ac := opts.AllowedOrigins
	if ac == nil {
		ac = defaultAllowedOrigins()
	}
	return &HTTPTransport{
		addr:             opts.Addr,
		logger:           opts.Logger,
		apiPrefix:        opts.APIPrefix,
		staticFS:         opts.StaticFS,
		router:           newPathTrie(),
		corsList:         NewCORSAllowlist(ac),
		trustedProxyHops: opts.TrustedProxyHops,
		bodyLimits:       make(map[string]int64),
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
	// maxRequestBodySize caps a single request body at the largest per-op
	// blob ceiling (maxUploadChunkBytes = 256 MiB). fasthttp's default is
	// only 4 MiB, which would break resumable-upload chunks. With
	// StreamRequestBody enabled the server never buffers the whole body;
	// every non-blob route is additionally capped at defaultBodyLimit by
	// the transport (S-8).
	maxRequestBodySize = 256 << 20

	// defaultBodyLimit caps request bodies for every route without an
	// explicit override; the large allowance is reserved for blob upload
	// routes via SetBodyLimit.
	defaultBodyLimit = 8 << 20
)

// Server timeouts and connection caps (S-8): fasthttp ships with none
// of these set, which exposes slowloris-style connection exhaustion
// and unbounded goroutine fan-out.
const (
	serverReadTimeout   = 30 * time.Second
	serverWriteTimeout  = 60 * time.Second
	serverIdleTimeout   = 120 * time.Second
	serverMaxConnsPerIP = 500
	serverConcurrency   = 1024
)

func (t *HTTPTransport) Start() error {
	t.server = &fasthttp.Server{
		Handler: t.serveHTTP,
		// S-8: bound memory and connection lifetime. StreamRequestBody keeps
		// request bodies off the heap until a handler asks for them, and the
		// global ceiling matches the largest per-op blob limit.
		MaxRequestBodySize: maxRequestBodySize,
		StreamRequestBody:  true,
		ReadTimeout:        serverReadTimeout,
		WriteTimeout:       serverWriteTimeout,
		IdleTimeout:        serverIdleTimeout,
		MaxConnsPerIP:      serverMaxConnsPerIP,
		Concurrency:        serverConcurrency,
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
		// S-8: per-route body ceiling. fasthttp enforces the global cap
		// while streaming; this check keeps every non-blob route to the
		// small default without ever reading the body.
		limit := int64(defaultBodyLimit)
		if custom, has := t.bodyLimits[method+" "+path]; has {
			limit = custom
		}
		if cl := int64(ctx.Request.Header.ContentLength()); cl > limit {
			t.writeJSON(ctx, fasthttp.StatusRequestEntityTooLarge, map[string]any{
				"error": map[string]any{"code": "PAYLOAD_TOO_LARGE", "message": "request body exceeds the route limit"},
			})
			return
		}

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
			ClientIP:   t.clientIP(ctx),
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

	if ms, ok := resp.Body.(*managedStream); ok {
		// S-19: the writer owns the done channel — close it when this writer
		// stops consuming (flush error on client disconnect, or the stream
		// drained) so the producer and its upstream subscription are
		// released. fasthttp's RequestCtx.Done() only fires on server
		// shutdown and can never release the producer on its own.
		defer ms.Close()
		t.writeEventStream(ctx, resp, ms.ch)
		return
	}
	if stream, ok := resp.Body.(abstract.StreamBody); ok {
		t.writeEventStream(ctx, resp, stream)
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

// writeEventStream renders a stream body as an SSE response.
func (t *HTTPTransport) writeEventStream(ctx *fasthttp.RequestCtx, resp abstract.Response, stream abstract.StreamBody) {
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

	switch {
	case errors.Is(err, hestiaruntime.ErrRestartRequired):
		// A-15: the operation succeeded but the change activates on restart.
		// Return an honest 503 instead of an opaque 500; no error logging —
		// this is an expected outcome, and the host's restart hook (wired
		// outermost in the chain) handles process termination.
		sysErr = common.NewSystemError("RESTART_REQUIRED", "the change requires a server restart; retry after the server comes back up")
		status = fasthttp.StatusServiceUnavailable
	case errors.As(err, &sysErr):
		status = systemErrorToStatus(sysErr)
	default:
		// S-16: raw internal error strings (SQL errors, filesystem paths)
		// must not reach clients. Log the cause server-side, return an
		// opaque message.
		if t.logger != nil {
			t.logger.Error("internal error",
				zap.Error(err),
				zap.String("request_id", string(ctx.Request.Header.Peek("X-Request-ID"))),
			)
		}
		sysErr = common.NewSystemError("INTERNAL_ERROR", "internal server error")
	}

	issue := sysErr.ToIssue()

	// S-16: 5xx causes leak internals to anonymous clients. The cause is
	// logged server-side above; 4xx details (validation issues, policy
	// denials) are legitimate client feedback and stay.
	details := issue.Cause
	if status >= fasthttp.StatusInternalServerError {
		details = nil
	}

	meta := buildResponseMeta(resp, ctx)

	ctx.SetStatusCode(status)
	json.NewEncoder(ctx).Encode(map[string]any{
		"error": responseErrorBody{
			Code:    issue.Code,
			Message: issue.Message,
			Details: details,
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

// SetBodyLimit overrides the default per-route body ceiling. Called at
// registration time for blob routes, whose payloads are the only ones
// legitimately larger than defaultBodyLimit (S-8).
func (t *HTTPTransport) SetBodyLimit(pattern string, maxBytes int64) {
	if t.bodyLimits == nil {
		t.bodyLimits = make(map[string]int64)
	}
	t.bodyLimits[pattern] = maxBytes
}

// clientIP resolves the client address for rate limiting, audit and
// access logs (S-7). Proxy-supplied headers are honored only when the
// deployment declares how many reverse proxies sit in front of hestia
// (TrustedProxyHops, env TRUSTED_PROXY_HOPS): direct-to-server clients
// can otherwise rotate X-Forwarded-For per request to mint fresh
// rate-limit buckets or forge audit source IPs. With N trusted hops,
// the N right-most X-Forwarded-For entries were contributed by trusted
// proxies, so entry len-N is the best client estimate; a header
// shorter than the configured chain is not trustworthy and falls back
// to RemoteAddr.
func (t *HTTPTransport) clientIP(ctx *fasthttp.RequestCtx) string {
	remote := stripPort(ctx.RemoteAddr().String())
	if t.trustedProxyHops <= 0 {
		return remote
	}
	if fwd := ctx.Request.Header.Peek("X-Forwarded-For"); len(fwd) > 0 {
		parts := strings.Split(string(fwd), ",")
		idx := len(parts) - t.trustedProxyHops
		if idx < 0 {
			return remote
		}
		return stripPort(strings.TrimSpace(parts[idx]))
	}
	if t.trustedProxyHops == 1 {
		if realIP := ctx.Request.Header.Peek("X-Real-IP"); len(realIP) > 0 {
			return stripPort(strings.TrimSpace(string(realIP)))
		}
	}
	return remote
}

// stripPort removes the :port suffix so RemoteAddr and proxy headers
// are keyed identically by rate limiting and audit.
func stripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil && host != "" {
		return host
	}
	return addr
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
	"RESTART_REQUIRED":    fasthttp.StatusServiceUnavailable,
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
