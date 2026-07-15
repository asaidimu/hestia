package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/abstract"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type TransportOptions struct {
	Addr   string
	Logger Logger
}

type HTTPTransport struct {
	addr   string
	logger Logger
	server *http.Server
	mux    *http.ServeMux
}

func NewTransport(opts TransportOptions) *HTTPTransport {
	return &HTTPTransport{
		addr:   opts.Addr,
		logger: opts.Logger,
		mux:    http.NewServeMux(),
	}
}

func (t *HTTPTransport) Handle(pattern string, handler abstract.Handler) {
	t.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		cookies := make(map[string]string, len(r.Cookies()))
		for _, c := range r.Cookies() {
			cookies[c.Name] = c.Value
		}

		req := abstract.Request{
			Operation:  pattern,
			Body:       body,
			PathParams: extractPathParams(r),
			Query:      r.URL.Query(),
			Headers:    r.Header,
			Cookies:    cookies,
			ClientIP:   clientIP(r),
			UserAgent:  r.UserAgent(),
			RequestID:  r.Header.Get("X-Request-ID"),
		}
		resp, err := handler(r.Context(), req)
		if err != nil {
			t.writeError(w, r, err)
			return
		}
		t.writeSuccess(w, r, resp)
	})
}

func (t *HTTPTransport) Start() error {
	handler := corsMiddleware(
		correlationIDMiddleware(t.mux),
	)
	t.server = &http.Server{
		Addr:    t.addr,
		Handler: handler,
	}
	if err := t.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (t *HTTPTransport) Shutdown(ctx context.Context) error {
	if t.server == nil {
		return nil
	}
	return t.server.Shutdown(ctx)
}

// ── Response writing ───────────────────────────────────────────────────────

func (t *HTTPTransport) writeSuccess(w http.ResponseWriter, r *http.Request, resp abstract.Response) {
	if resp.Status == 0 {
		resp.Status = http.StatusOK
	}

	for _, c := range resp.Cookies {
		http.SetCookie(w, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			MaxAge:   c.MaxAge,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
			SameSite: c.SameSite,
		})
	}

	for k, vals := range resp.Headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	if raw, ok := resp.Body.([]byte); ok {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.WriteHeader(resp.Status)
		_, _ = w.Write(raw)
		return
	}

	if stream, ok := resp.Body.(abstract.StreamBody); ok {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(resp.Status)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
		for data := range stream {
			jsonBytes, err := json.Marshal(map[string]any{"data": data})
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
			flusher.Flush()
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)

	if resp.Status == http.StatusNoContent {
		return
	}

	meta := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"request":   r.Header.Get("X-Request-ID"),
	}
	if resp.Page != nil {
		meta["page"] = resp.Page
	}

	json.NewEncoder(w).Encode(map[string]any{
		"data":     resp.Body,
		"metadata": meta,
	})
}

func (t *HTTPTransport) writeError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")

	status := http.StatusInternalServerError
	var sysErr *common.SystemError

	if errors.As(err, &sysErr) {
		status = systemErrorToStatus(sysErr)
	} else {
		sysErr = common.NewSystemError("INTERNAL_ERROR", err.Error())
	}

	issue := sysErr.ToIssue()

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    issue.Code,
			"message": issue.Message,
			"details": issue.Cause,
		},
		"metadata": map[string]any{
			"timestamp": time.Now().Format(time.RFC3339),
			"request":   r.Header.Get("X-Request-ID"),
		},
	})
}

// ── helpers ────────────────────────────────────────────────────────────────

var pathParamRe = regexp.MustCompile(`\{(\w+)\}`)

func extractPathParams(r *http.Request) map[string]string {
	m := make(map[string]string)
	for _, match := range pathParamRe.FindAllStringSubmatch(r.Pattern, -1) {
		key := match[1]
		if v := r.PathValue(key); v != "" {
			m[key] = v
		}
	}
	return m
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

// ── Middleware ──────────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, x-api-key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = r.Header.Get("X-Correlation-ID")
		}
		if id == "" {
			id = randomID()
		}
		w.Header().Set("X-Request-ID", id)
		r.Header.Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

func randomID() string {
	return uuid.Must(uuid.NewV7()).String()
}

var codeToStatus = map[string]int{
	"ERR_ACCESS_DENIED":     403,
	"NOT_FOUND":             404,
	"ALREADY_EXISTS":        409,
	"VALIDATION_ERROR":      400,
	"INVALID_REQUEST":       400,
	"UNAUTHORIZED":          401,
	"INVALID_CREDENTIALS":   401,
	"EMAIL_EXISTS":          409,
	"USER_DELETED":          410,
	"FORBIDDEN":             403,
	"MISSING_PARAM":         400,
	"INVALID_QDSL":          400,
	"DOCUMENT_REQUIRED":     400,
	"PARSE_DOCUMENT":        400,
	"SCHEMA_REQUIRED":       400,
	"SCHEMA_MISSING_NAME":   400,
	"COLLECTION_EXISTS":     409,
	"RESERVED_NAME":         409,
	"AUTH_REQUIRED":         401,
	"DOCUMENT_NOT_FOUND":    404,
	"NOT_IMPLEMENTED":       501,
	"SERVICE_UNAVAILABLE":   503,
}

func codeToStatusFn(code string) int {
	if s, ok := codeToStatus[code]; ok {
		return s
	}
	return 500
}

func systemErrorToStatus(err *common.SystemError) int {
	return codeToStatusFn(err.Code)
}
