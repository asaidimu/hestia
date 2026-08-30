package wails

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/abstract"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/runtime/route"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/audit"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type Request struct {
	Name      string              `json:"name"`
	Arguments map[string]string   `json:"arguments"`
	Modifiers map[string][]string `json:"modifiers,omitempty"`
	Payload   any                 `json:"payload,omitempty"`
}

type Response struct {
	Data   any `json:"data"`
	Status int `json:"status"`
}

type Options struct {
	Dispatcher    abstract.Dispatcher
	Internal      abstract.Dispatcher
	CredProvider  abstract.CredentialsProvider
	Registrations []abstract.MessageRegistration

	// SourceIP is the audit source IP label for in-process Dispatch calls.
	// Default: "wails".
	SourceIP string
	// UserAgent is the audit user-agent label for in-process Dispatch calls.
	// Default: "hestia-desktop".
	UserAgent string
	// APIPrefix is the URL prefix for the HTTP handler.
	// Default: "/api".
	APIPrefix string
}

type routeEntry struct {
	method string
	path   string
	name   string
	intent abstract.Verb
	input  abstract.Input
	output *definition.Schema
}

type Adapter struct {
	opts Options

	mu           sync.RWMutex
	sessionToken string
	userID       string
	claims       *abstract.Claims

	sourceIP string
	agent    string
	prefix   string
	routes   []routeEntry
}

func New(opts Options) *Adapter {
	a := &Adapter{opts: opts}
	a.sourceIP = opts.SourceIP
	if a.sourceIP == "" {
		a.sourceIP = "wails"
	}
	a.agent = opts.UserAgent
	if a.agent == "" {
		a.agent = "hestia-desktop"
	}
	a.prefix = opts.APIPrefix
	if a.prefix == "" {
		a.prefix = "/api"
	}
	a.buildRoutes()
	return a
}

func (a *Adapter) Start(bootstrapped bool) {
	a.buildRoutes()
}

func (a *Adapter) Restart(bootstrapped bool) {
	a.routes = nil
	a.buildRoutes()
}

func (a *Adapter) Login(email, password string) (map[string]any, error) {
	ctx := context.Background()
	doc := data.MustNewDocument(map[string]any{}, ctx)
	doc.Set("payload", map[string]any{"email": email, "password": password})

	msg := dispatch.NewMessage("system:auth:session:create", ctx, doc)
	result, err := dispatch.Await(ctx, a.opts.Internal, msg)
	if err != nil {
		return nil, err
	}

	token := result.SessionToken
	if token == "" {
		return nil, fmt.Errorf("authentication succeeded but no session token was returned")
	}

	info, err := a.opts.CredProvider.ValidateSession(token)
	if err != nil {
		return nil, fmt.Errorf("failed to validate new session: %w", err)
	}

	claims := a.resolveClaims(ctx, info.UserID)

	a.mu.Lock()
	a.sessionToken = token
	a.userID = info.UserID
	a.claims = claims
	a.mu.Unlock()

	if result.Document != nil {
		sane, _ := result.Document.Sanitize()
		if sane != nil {
			return sane.ToMap(), nil
		}
	}
	return map[string]any{}, nil
}

func (a *Adapter) Logout() error {
	a.mu.Lock()
	a.sessionToken = ""
	a.userID = ""
	a.claims = nil
	a.mu.Unlock()
	return nil
}

func (a *Adapter) IsAuthenticated() bool {
	a.mu.RLock()
	token := a.sessionToken
	claims := a.claims
	a.mu.RUnlock()

	if token == "" || claims == nil {
		return false
	}

	info, err := a.opts.CredProvider.ValidateSession(token)
	if err != nil {
		return false
	}
	return info.ExpiresAt > time.Now().Unix()
}

func (a *Adapter) Dispatch(req Request) (Response, error) {
	a.mu.RLock()
	sessionToken := a.sessionToken
	a.mu.RUnlock()

	ctx := context.Background()

	var claims *abstract.Claims
	if sessionToken != "" {
		info, err := a.opts.CredProvider.ValidateSession(sessionToken)
		if err == nil && info.ExpiresAt > time.Now().Unix() {
			claims = a.resolveClaims(ctx, info.UserID)
		}
	}

	doc := data.MustNewDocument(map[string]any{}, ctx)
	args := make(map[string]any)
	for k, v := range req.Arguments {
		args[k] = v
	}
	doc.Set("arguments", args)

	mods := make(map[string]any)
	for k, vals := range req.Modifiers {
		if len(vals) > 0 {
			mods[k] = vals[0]
		}
	}
	doc.Set("modifiers", mods)
	if ct, ok := mods["content_type"]; ok {
		doc.Set("content_type", ct)
	}

	if req.Payload != nil {
		payload := req.Payload
		if s, ok := payload.(string); ok {
			if decoded, err := base64.StdEncoding.DecodeString(s); err == nil && len(decoded) > 0 {
				payload = decoded
			}
		}
		doc.Set("payload", payload)
	}

	// Transport context (in-process, so use static descriptors)
	traceID := dispatch.MustNewID()
	ctx = runtime.ContextWithAuditTransport(ctx, a.sourceIP, a.agent, traceID)

	// Trace ID
	ctx = runtimecontext.ContextWithTraceID(ctx, traceID)

	ctx = a.authenticatedContext(ctx, claims)

	if schema := a.lookupSchema(req.Name); schema != nil {
		if issues, ok := dispatch.ValidateInputDocument(schema, doc); !ok {
			return Response{}, common.NewSystemError("VALIDATION_ERROR", "input validation failed").WithIssues(issues)
		}
	}

	msg := dispatch.NewMessage(req.Name, ctx, doc)
	result, err := dispatch.Await(ctx, a.opts.Dispatcher, msg)
	if err != nil {
		status := 500
		code := "INTERNAL_ERROR"
		var sysErr *common.SystemError
		if errors.As(err, &sysErr) {
			status = httpapi.SystemErrorToStatus(sysErr)
			code = sysErr.Code
		}
		return Response{
			Data:   map[string]any{"data": nil, "metadata": map[string]any{}, "error": map[string]any{"code": code, "message": err.Error()}},
			Status: status,
		}, err
	}

	a.mu.Lock()
	// @note #review2-20260821-002 issue resolved P2 #security,#auth,#review : ValidateSession result used without checking ExpiresAt
	// @see #review2-20260821-001
	// Resolved by fix to #review2-20260821-001: SessionService.Validate now rejects expired tokens, so all call sites are protected.
	//
	// Two call sites in this file (around line 122, in the post-login flow, and here around line 245-251, in dispatch response handling) consume the ValidateSession result and set a.userID/a.claims without checking info.ExpiresAt. Contrast with the correctly-guarded call sites at lines 163-167, 179-182, and 382-395 in this same file, which all check info.ExpiresAt > time.Now().Unix() before trusting the claims. The line-122 case is low risk in practice (it runs immediately after a fresh login, so the token cannot yet be expired), but the line-245 case handles an arbitrary dispatch response and has no such guarantee. This is a symptom of the root cause at #review2-20260821-001: SessionService.Validate does not enforce its own expiry, so every caller must remember to, and this file shows that as inconsistent in practice.
	if token := result.SessionToken; token != "" {
		a.sessionToken = token
		info, err := a.opts.CredProvider.ValidateSession(token)
		if err == nil {
			a.userID = info.UserID
			a.claims = a.resolveClaims(ctx, info.UserID)
		}
	}
	if req.Name == "system:auth:session:delete" {
		a.sessionToken = ""
		a.userID = ""
		a.claims = nil
	}
	a.mu.Unlock()

	return buildResponse(result), nil
}

func (a *Adapter) Handler() http.Handler {
	return http.HandlerFunc(a.serveHTTP)
}

func (a *Adapter) buildRoutes() {
	for _, reg := range a.opts.Registrations {
		if reg.Internal {
			continue
		}

		httpMethod := route.IntentToHTTPMethod(reg.Intent)
		httpPath := route.DeriveRoute(reg.Name, reg.Input.Arguments())
		httpPath = a.prefix + httpPath

		a.routes = append(a.routes, routeEntry{
			method: httpMethod,
			path:   route.IntentToHTTPPath(reg.Intent, httpPath),
			name:   reg.Name,
			intent: reg.Intent,
			input:  reg.Input,
			output: reg.Output,
		})
	}
}

func (a *Adapter) lookupSchema(name string) *definition.Schema {
	for _, reg := range a.opts.Registrations {
		if reg.Name == name {
			return reg.Input.Schema
		}
	}
	return nil
}

func (a *Adapter) resolveClaims(ctx context.Context, userID string) *abstract.Claims {
	if userID == "" {
		return &abstract.Claims{}
	}

	doc := data.MustNewDocument(map[string]any{}, ctx)
	doc.Set("arguments", map[string]any{"user_id": userID})

	msg := dispatch.NewMessage("system:users:user:get", ctx, doc)
	result, err := dispatch.Await(ctx, a.opts.Internal, msg)
	if err != nil || result == nil || result.Document == nil {
		return &abstract.Claims{UserID: userID}
	}

	email, _ := result.Document.GetString("email")
	perms, _ := result.Document.GetStringArray("permissions")
	if perms == nil {
		perms = []string{}
	}

	return &abstract.Claims{
		UserID:    userID,
		Email:     email,
		Scopes:    perms,
		TokenType: "session",
	}
}

func (a *Adapter) authenticatedContext(ctx context.Context, claims *abstract.Claims) context.Context {
	if claims == nil {
		claims = &abstract.Claims{}
	}
	ctx = runtimecontext.ContextWithClaims(ctx, claims)

	if claims.UserID != "" {
		ctx = runtime.ContextWithAuditIdentity(ctx, claims.UserID, audit.ActorTypeUser, audit.AuthMethodPassword)
	}

	a.mu.RLock()
	token := a.sessionToken
	a.mu.RUnlock()
	if token != "" {
		ctx = runtimecontext.ContextWithSessionID(ctx, token)
	}

	return ctx
}

func (a *Adapter) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, a.prefix) {
		http.NotFound(w, r)
		return
	}

	for _, route := range a.routes {
		if route.method != "" && route.method != r.Method {
			continue
		}
		params := httpapi.ExtractPathParams(route.path, r.URL.Path)
		if params == nil && route.path != r.URL.Path {
			continue
		}

		ctx := r.Context()

		// Transport context
		reqID := r.Header.Get("X-Request-ID")
		ctx = runtime.ContextWithAuditTransport(ctx, r.RemoteAddr, r.UserAgent(), reqID)

		// Trace ID
		traceID := reqID
		if traceID == "" {
			traceID = dispatch.MustNewID()
		}
		ctx = runtimecontext.ContextWithTraceID(ctx, traceID)

		// Resource ID from route definition
		if route.input.ResourceIDField != "" {
			if rid, ok := params[route.input.ResourceIDField]; ok && rid != "" {
				ctx = runtimecontext.ContextWithResourceID(ctx, rid)
			}
		}

		var claims *abstract.Claims
		if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
			info, err := a.opts.CredProvider.ValidateSession(cookie.Value)
			if err == nil && info.ExpiresAt > time.Now().Unix() {
				claims = a.resolveClaims(ctx, info.UserID)
			}
		}
		if claims == nil {
			a.mu.RLock()
			token := a.sessionToken
			a.mu.RUnlock()
			if token != "" {
				info, err := a.opts.CredProvider.ValidateSession(token)
				if err == nil && info.ExpiresAt > time.Now().Unix() {
					claims = a.resolveClaims(ctx, info.UserID)
				}
			}
		}
		ctx = a.authenticatedContext(ctx, claims)

		doc, err := buildDoc(ctx, r, params, route.input)
		if err != nil {
			writeError(w, common.NewSystemError("VALIDATION_ERROR", "input validation failed").WithIssues([]common.Issue{{Code: "INVALID_INPUT", Message: err.Error()}}))
			return
		}

		if issues, ok := dispatch.ValidateInputDocument(route.input.Schema, doc); !ok {
			writeError(w, common.NewSystemError("VALIDATION_ERROR", "input validation failed").WithIssues(issues))
			return
		}

		msg := dispatch.NewMessage(route.name, ctx, doc)
		result, err := dispatch.Await(ctx, a.opts.Dispatcher, msg)
		if err != nil {
			writeError(w, err)
			return
		}

		writeResult(w, result, route.intent)
		return
	}

	http.NotFound(w, r)
}

func buildResponse(result *abstract.Result) Response {
	meta := map[string]any{}
	resp := Response{Status: 200}

	if result == nil {
		resp.Data = map[string]any{"data": nil, "metadata": meta}
		return resp
	}

	switch {
	case result.Document != nil:
		sane, _ := result.Document.Sanitize()
		resp.Data = map[string]any{"data": sane, "metadata": meta}

	case result.Documents != nil:
		items := make([]data.Documenter, 0, len(result.Documents))
		for _, d := range result.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		resp.Data = map[string]any{"data": items, "metadata": meta}

	case result.Page != nil:
		if p := result.Page.Pagination; p != nil {
			meta["page"] = p
		}
		items := make([]data.Documenter, 0, len(result.Page.Documents))
		for _, d := range result.Page.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		resp.Data = map[string]any{"data": items, "metadata": meta}

	default:
		resp.Data = map[string]any{"data": nil, "metadata": meta}
	}

	return resp
}

func buildDoc(ctx context.Context, r *http.Request, pathParams map[string]string, input abstract.Input) (data.Documenter, error) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	pool, err := document.NewDocumentPool(input.Schema)
	if err != nil {
		return nil, err
	}
	req := httpapi.Request{
		Body:       body,
		PathParams: pathParams,
		Query:      r.URL.Query(),
		Headers:    map[string][]string{"Content-Type": {r.Header.Get("Content-Type")}},
	}
	return httpapi.BuildInputDocument(pool, input, req)
}

func writeResult(w http.ResponseWriter, result *abstract.Result, _ abstract.Verb) {
	if result == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "metadata": map[string]any{}})
		return
	}

	if result.Blob.Data != nil {
		ct := result.Blob.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		w.Write(result.Blob.Data)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	switch {
	case result.Document != nil:
		sane, _ := result.Document.Sanitize()
		json.NewEncoder(w).Encode(map[string]any{"data": sane, "metadata": map[string]any{}})

	case result.Documents != nil:
		items := make([]data.Documenter, 0, len(result.Documents))
		for _, d := range result.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"data": items, "metadata": map[string]any{}})

	case result.Page != nil:
		meta := map[string]any{}
		if p := result.Page.Pagination; p != nil {
			meta["page"] = p
		}
		items := make([]data.Documenter, 0, len(result.Page.Documents))
		for _, d := range result.Page.Documents {
			if sane, _ := d.Sanitize(); sane != nil {
				items = append(items, sane)
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"data": items, "metadata": meta})

	default:
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "metadata": map[string]any{}})
	}
}

func writeError(w http.ResponseWriter, err error) {
	message := err.Error()
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"

	var sysErr *common.SystemError
	if errors.As(err, &sysErr) {
		code = sysErr.Code
		status = httpapi.SystemErrorToStatus(sysErr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"metadata": map[string]any{},
	})
}
