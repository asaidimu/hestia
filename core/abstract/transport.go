package abstract

import (
        "context"
        "io"

        "github.com/asaidimu/go-anansi/v8/core/query"
)

// @note #perf-20260821-001 issue status=open priority=P2 tags=#performance,#struct-packing : Request struct has poor field ordering
//
// Request struct (line 9) has suboptimal field ordering for memory alignment:
//
// Current layout (64-bit):
// - Operation: string (16 bytes)
// - Body: []byte (24 bytes)
// - PathParams: map (8 bytes)
// - Query: map (8 bytes)
// - Headers: map (8 bytes)
// - Cookies: map (8 bytes)
// - ClientIP: string (16 bytes)
// - UserAgent: string (16 bytes)
// - RequestID: string (16 bytes)
//
// Total: ~120 bytes with possible padding
//
// Better ordering (group by size):
// - Body: []byte (24 bytes)
// - Operation: string (16 bytes)
// - ClientIP: string (16 bytes)
// - UserAgent: string (16 bytes)
// - RequestID: string (16 bytes)
// - PathParams: map (8 bytes)
// - Query: map (8 bytes)
// - Headers: map (8 bytes)
// - Cookies: map (8 bytes)
//
// For IoT/HFT: Every byte matters on memory-constrained devices.
// This struct is created for every HTTP request.
//
// Resolution: Reorder fields to minimize padding. Consider using a pool
// for Request objects to reduce allocations.
type Request struct {
        Operation string
        // Body is the fully-buffered request body, set for non-streaming routes.
        Body []byte
        // BodyStream is set INSTEAD of Body for streaming routes: the transport
        // must not touch ctx.Request.Body() on that path, or fasthttp buffers the
        // whole body and defeats streaming. Owned by the handler closure for the
        // lifetime of exactly one serveHTTP invocation.
        BodyStream io.Reader
        PathParams map[string]string
        Query      map[string][]string
        Headers    map[string][]string
        Cookies    map[string]string
        ClientIP   string
        UserAgent  string
        RequestID  string
}

// RouteOptions carries per-route transport behavior. RouteOption mutators are
// passed to Transport.Handle as variadic options so the transport can decide
// before touching the body whether a route streams.
type RouteOptions struct {
        StreamingBody bool
}

type RouteOption func(*RouteOptions)

// WithStreamingBody marks a route as expecting a streamed request body. The
// transport then hands the handler BodyStream instead of buffering Body.
func WithStreamingBody() RouteOption { return func(o *RouteOptions) { o.StreamingBody = true } }

type SameSite int

const (
        SameSiteStrictMode SameSite = iota + 1
        SameSiteLaxMode
        SameSiteNoneMode
)

type Cookie struct {
        Name     string
        Value    string
        Path     string
        Domain   string
        MaxAge   int
        Secure   bool
        HTTPOnly bool
        SameSite SameSite
}

type StreamBody <-chan any

// @note #perf-20260821-002 issue status=open priority=P2 tags=#performance,#struct-packing : Response struct has mixed pointer/value types
//
// Response struct (line 42) mixes pointer and value types inefficiently:
//
// - Status: int (8 bytes)
// - Headers: map (8 bytes)
// - Body: any (16 bytes - interface)
// - Cookies: []Cookie (24 bytes - slice)
// - Page: *query.PaginationInfo (8 bytes - pointer)
// - Metadata: map (8 bytes)
//
// The Body field is an interface, which means it can hold any type.
// This causes boxing/unboxing overhead and prevents compiler optimizations.
//
// For IoT/HFT: Interface types cause GC pressure and prevent inlining.
//
// Resolution:
// 1. Consider using concrete types instead of interface{} for Body
// 2. Group pointer types together to minimize padding
// 3. Consider using a pool for Response objects
type Response struct {
        Status   int
        Headers  map[string][]string
        Body     any
        Cookies  []Cookie
        Page     *query.PaginationInfo
        Metadata map[string]any
}

type Handler func(ctx context.Context, req Request) (Response, error)

type Transport interface {
        // Handle registers a route. Options are appended (backward compatible);
        // WithStreamingBody opts the route into streamed-body delivery.
        Handle(pattern string, handler Handler, opts ...RouteOption)
        Start() error
        Shutdown(ctx context.Context) error
}
