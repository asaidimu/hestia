package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/valyala/fasthttp"

	"github.com/asaidimu/hestia/app/abstract"
)

func newCtx(method, path string) *fasthttp.RequestCtx {
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	return ctx
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    map[string]string
	}{
		{"single param", "/users/{id}", "/users/123", map[string]string{"id": "123"}},
		{"multiple params", "/{a}/x/{b}", "/foo/x/bar", map[string]string{"a": "foo", "b": "bar"}},
		{"no params", "/static", "/static", map[string]string{}},
		{"no match", "/users/{id}", "/items/123", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathParams(tt.pattern, tt.path)
			if tt.want == nil {
				if got != nil {
					t.Errorf("want nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d: got %v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("params[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestCORS(t *testing.T) {
	transport := NewTransport(TransportOptions{Addr: ":0"})
	transport.Handle("GET /test", func(ctx2 context.Context, req abstract.Request) (abstract.Response, error) {
		return abstract.Response{Status: 200, Body: "ok"}, nil
	})

	// Direct CORS header test using the transport's serveHTTP
	ctx := newCtx("OPTIONS", "/test")
	transport.serveHTTP(ctx)

	if string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")) == "" {
		t.Error("CORS headers not set on OPTIONS")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Errorf("status = %d, want 204", ctx.Response.StatusCode())
	}
}

func TestCorrelationID(t *testing.T) {
	t.Run("generates ID when none provided", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		transport.Handle("GET /ping", func(ctx2 context.Context, req abstract.Request) (abstract.Response, error) {
			return abstract.Response{Status: 200, Body: "ok"}, nil
		})

		ctx := newCtx("GET", "/ping")
		transport.serveHTTP(ctx)

		id := ctx.Response.Header.Peek("X-Request-ID")
		if len(id) == 0 {
			t.Error("expected X-Request-ID to be set")
		}
	})

	t.Run("preserves existing X-Request-ID", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		transport.Handle("GET /ping", func(ctx2 context.Context, req abstract.Request) (abstract.Response, error) {
			return abstract.Response{Status: 200, Body: "ok"}, nil
		})

		ctx := newCtx("GET", "/ping")
		ctx.Request.Header.Set("X-Request-ID", "my-id")
		transport.serveHTTP(ctx)

		if string(ctx.Response.Header.Peek("X-Request-ID")) != "my-id" {
			t.Errorf("got %q, want my-id", ctx.Response.Header.Peek("X-Request-ID"))
		}
	})
}

func TestWriteSuccess(t *testing.T) {
	t.Run("writes status and JSON body", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		ctx := newCtx("GET", "/")
		transport.writeSuccess(ctx, abstract.Response{Status: 201, Body: map[string]string{"foo": "bar"}})

		if ctx.Response.StatusCode() != 201 {
			t.Errorf("status = %d, want 201", ctx.Response.StatusCode())
		}
		var body map[string]any
		if err := json.Unmarshal(ctx.Response.Body(), &body); err != nil {
			t.Fatalf("json error: %v", err)
		}
		if _, ok := body["data"]; !ok {
			t.Error("missing 'data' key")
		}
	})

	t.Run("no content returns 204", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		ctx := newCtx("GET", "/")
		transport.writeSuccess(ctx, abstract.Response{Status: 204})

		if ctx.Response.StatusCode() != 204 {
			t.Errorf("status = %d, want 204", ctx.Response.StatusCode())
		}
		if len(ctx.Response.Body()) != 0 {
			t.Error("expected empty body")
		}
	})

	t.Run("raw bytes body", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		ctx := newCtx("GET", "/")
		transport.writeSuccess(ctx, abstract.Response{
			Status: 200,
			Body:   []byte("raw data"),
			Headers: map[string][]string{
				"Content-Type": {"application/octet-stream"},
			},
		})

		if string(ctx.Response.Body()) != "raw data" {
			t.Errorf("body = %q, want raw data", ctx.Response.Body())
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("known system error", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		ctx := newCtx("GET", "/")
		sysErr := common.NewSystemError("NOT_FOUND", "test not found")
		transport.writeError(ctx, sysErr, nil)

		if ctx.Response.StatusCode() != 404 {
			t.Errorf("status = %d, want 404", ctx.Response.StatusCode())
		}
	})

	t.Run("plain error maps to 500", func(t *testing.T) {
		transport := NewTransport(TransportOptions{Addr: ":0"})
		ctx := newCtx("GET", "/")
		transport.writeError(ctx, errors.New("something broke"), nil)

		if ctx.Response.StatusCode() != 500 {
			t.Errorf("status = %d, want 500", ctx.Response.StatusCode())
		}
	})
}

func TestCodeToStatus(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{"NOT_FOUND", 404},
		{"ALREADY_EXISTS", 409},
		{"VALIDATION_ERROR", 400},
		{"UNAUTHORIZED", 401},
		{"FORBIDDEN", 403},
		{"NOT_IMPLEMENTED", 501},
		{"SERVICE_UNAVAILABLE", 503},
		{"", 500},
		{"UNKNOWN_CODE", 500},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := codeToStatusFn(tt.code)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSystemErrorToStatus(t *testing.T) {
	tests := []struct {
		name string
		code string
		want int
	}{
		{"NOT_FOUND", "NOT_FOUND", 404},
		{"ALREADY_EXISTS", "ALREADY_EXISTS", 409},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysErr := common.NewSystemError(tt.code, tt.name)
			got := systemErrorToStatus(sysErr)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	t.Run("X-Forwarded-For", func(t *testing.T) {
		ctx := newCtx("GET", "/")
		ctx.Request.Header.Set("X-Forwarded-For", "1.2.3.4")
		if ip := clientIP(ctx); ip != "1.2.3.4" {
			t.Errorf("got %q, want 1.2.3.4", ip)
		}
	})

	t.Run("X-Real-IP fallback", func(t *testing.T) {
		ctx := newCtx("GET", "/")
		ctx.Request.Header.Set("X-Real-IP", "5.6.7.8")
		if ip := clientIP(ctx); ip != "5.6.7.8" {
			t.Errorf("got %q, want 5.6.7.8", ip)
		}
	})

	t.Run("RemoteAddr fallback", func(t *testing.T) {
		ctx := newCtx("GET", "/")
		if ip := clientIP(ctx); ip == "" {
			t.Error("expected non-empty RemoteAddr")
		}
	})
}

func TestSplitPattern(t *testing.T) {
	method, path := splitPattern("GET /api/users")
	if method != "GET" || path != "/api/users" {
		t.Errorf("got %q %q, want GET /api/users", method, path)
	}
	method, path = splitPattern("/no-method")
	if method != "" || path != "/no-method" {
		t.Errorf("got %q %q, want \"\" /no-method", method, path)
	}
}

func TestMapSameSite(t *testing.T) {
	if got := mapSameSite(abstract.SameSiteStrictMode); got != fasthttp.CookieSameSiteStrictMode {
		t.Errorf("strict: got %d", got)
	}
	if got := mapSameSite(abstract.SameSiteLaxMode); got != fasthttp.CookieSameSiteLaxMode {
		t.Errorf("lax: got %d", got)
	}
	if got := mapSameSite(abstract.SameSiteNoneMode); got != fasthttp.CookieSameSiteNoneMode {
		t.Errorf("none: got %d", got)
	}
	if got := mapSameSite(0); got != fasthttp.CookieSameSiteStrictMode {
		t.Errorf("default: got %d", got)
	}
}
