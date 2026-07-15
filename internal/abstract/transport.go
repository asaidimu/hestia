package abstract

import (
	"context"
	"net/http"

	"github.com/asaidimu/go-anansi/v8/core/query"
)

type Request struct {
	Operation  string
	Body       []byte
	PathParams map[string]string
	Query      map[string][]string
	Headers    map[string][]string
	Cookies    map[string]string
	ClientIP   string
	UserAgent  string
	RequestID  string
}

type Cookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	MaxAge   int
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

type StreamBody <-chan any

type Response struct {
	Status  int
	Headers map[string][]string
	Body    any
	Cookies []Cookie
	Page    *query.PaginationInfo
}

type Handler func(ctx context.Context, req Request) (Response, error)

type Transport interface {
	Handle(pattern string, handler Handler)
	Start() error
	Shutdown(ctx context.Context) error
}
