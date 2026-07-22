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
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/identity"
	"github.com/asaidimu/hestia/core/interface/api"
	httpserver "github.com/asaidimu/hestia/core/interface/api/http"
	"github.com/asaidimu/hestia/core/registration"
	"github.com/asaidimu/hestia/core/runtime"
)



type Request struct {
	Name      string              `json:"name"`
	Arguments map[string]string   `json:"arguments"`
	Modifiers map[string][]string `json:"modifiers,omitempty"`
	Payload   any                 `json:"payload,omitempty"`
}

type Response struct {
	Data   map[string]any `json:"data"`
	Status int            `json:"status"`
}

type Options struct {
	Dispatcher    runtime.Dispatcher
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
	intent registration.Verb
	input  abstract.Input
	output *definition.Schema
}

type Adapter struct {
	opts Options

	mu           sync.RWMutex
	sessionToken string
	userID       string
	claims       *identity.Claims

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

	msg := abstract.NewMessage("system:auth:session:create", ctx, doc)
	result, err := a.opts.Internal.Send(msg)
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

	if m := api.SanitizeToMap(result.Document); m != nil {
		return m, nil
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

	var claims *identity.Claims
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
	traceID := abstract.MustNewID()
	ctx = runtime.ContextWithAuditTransport(ctx, a.sourceIP, a.agent, traceID)

	// Trace ID
	ctx = runtime.ContextWithTraceID(ctx, traceID)

	ctx = a.authenticatedContext(ctx, claims)

	msg := abstract.NewMessage(req.Name, ctx, doc)
	result, err := a.opts.Dispatcher.Send(msg)
	if err != nil {
		status := 500
		code := "INTERNAL_ERROR"
		var sysErr *common.SystemError
		if errors.As(err, &sysErr) {
			status = api.SystemErrorToStatus(sysErr)
			code = sysErr.Code
		}
		return Response{
			Data:   map[string]any{"data": nil, "metadata": map[string]any{}, "error": map[string]any{"code": code, "message": err.Error()}},
			Status: status,
		}, err
	}

	a.mu.Lock()
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

		httpMethod := api.IntentToHTTPMethod(reg.Intent)
		httpPath := api.DeriveRoute(reg.Name, reg.Input.Arguments)
		httpPath = a.prefix + httpPath

		a.routes = append(a.routes, routeEntry{
			method: httpMethod,
			path:   api.IntentToHTTPPath(reg.Intent, httpPath),
			name:   reg.Name,
			intent: reg.Intent,
			input:  reg.Input,
			output: reg.Output,
		})
	}
}

func (a *Adapter) resolveClaims(ctx context.Context, userID string) *identity.Claims {
	if userID == "" {
		return &identity.Claims{}
	}

	doc := data.MustNewDocument(map[string]any{}, ctx)
	doc.Set("arguments", map[string]any{"user_id": userID})

	msg := abstract.NewMessage("system:users:user:get", ctx, doc)
	result, err := a.opts.Internal.Send(msg)
	if err != nil || result == nil || result.Document == nil {
		return &identity.Claims{UserID: userID}
	}

	email, _ := result.Document.GetString("email")
	perms, _ := result.Document.GetStringArray("permissions")
	if perms == nil {
		perms = []string{}
	}

	return &identity.Claims{
		UserID:    userID,
		Email:     email,
		Scopes:    perms,
		TokenType: "session",
	}
}

func (a *Adapter) authenticatedContext(ctx context.Context, claims *identity.Claims) context.Context {
	if claims == nil {
		claims = &identity.Claims{}
	}
	ctx = identity.ContextWithClaims(ctx, claims)

	if claims.UserID != "" {
		ctx = runtime.ContextWithAuditIdentity(ctx, claims.UserID, runtime.ActorTypeUser, runtime.AuthMethodPassword)
	}

	a.mu.RLock()
	token := a.sessionToken
	a.mu.RUnlock()
	if token != "" {
		ctx = runtime.ContextWithAuditSessionID(ctx, token)
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
		params := httpserver.ExtractPathParams(route.path, r.URL.Path)
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
			traceID = abstract.MustNewID()
		}
		ctx = runtime.ContextWithTraceID(ctx, traceID)

		// Resource ID from route definition
		if route.input.ResourceIDField != "" {
			if rid, ok := params[route.input.ResourceIDField]; ok && rid != "" {
				ctx = runtime.ContextWithAuditResourceID(ctx, rid)
			}
		}

		var claims *identity.Claims
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

		doc := buildDoc(ctx, r, params, route.input)

		msg := abstract.NewMessage(route.name, ctx, doc)
		result, err := a.opts.Dispatcher.Send(msg)
		if err != nil {
			writeError(w, err)
			return
		}

		writeResult(w, result, route.intent)
		return
	}

	http.NotFound(w, r)
}

func buildResponse(result *registration.Result) Response {
	meta := map[string]any{}
	resp := Response{Status: 200}

	if result == nil {
		resp.Data = map[string]any{"data": nil, "metadata": meta}
		return resp
	}

	switch {
	case result.Document != nil:
		resp.Data = map[string]any{"data": api.SanitizeToMap(result.Document), "metadata": meta}

	case result.Documents != nil:
		resp.Data = map[string]any{"data": api.SanitizeAll(result.Documents), "metadata": meta}

	case result.Page != nil:
		if p := result.Page.Pagination; p != nil {
			meta["page"] = p
		}
		resp.Data = map[string]any{"data": api.SanitizeAll(result.Page.Documents), "metadata": meta}

	default:
		resp.Data = map[string]any{"data": nil, "metadata": meta}
	}

	return resp
}

func buildDoc(ctx context.Context, r *http.Request, pathParams map[string]string, input abstract.Input) *data.Document {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	return api.BuildInputDocument(ctx, input, pathParams, r.URL.Query(), body)
}

func writeResult(w http.ResponseWriter, result *registration.Result, _ registration.Verb) {
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
		json.NewEncoder(w).Encode(map[string]any{"data": api.SanitizeToMap(result.Document), "metadata": map[string]any{}})

	case result.Documents != nil:
		json.NewEncoder(w).Encode(map[string]any{"data": api.SanitizeAll(result.Documents), "metadata": map[string]any{}})

	case result.Page != nil:
		meta := map[string]any{}
		if p := result.Page.Pagination; p != nil {
			meta["page"] = p
		}
		json.NewEncoder(w).Encode(map[string]any{"data": api.SanitizeAll(result.Page.Documents), "metadata": meta})

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
		status = api.SystemErrorToStatus(sysErr)
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


