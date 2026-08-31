package http

import "github.com/asaidimu/hestia/core/abstract"

type Request = abstract.Request
type Response = abstract.Response
type Cookie = abstract.Cookie
type Handler = abstract.Handler
type Transport = abstract.Transport
type StreamBody = abstract.StreamBody
type RouteOptions = abstract.RouteOptions
type RouteOption = abstract.RouteOption

// WithStreamingBody opts a route into streamed-body delivery: the handler
// receives Request.BodyStream instead of a buffered Request.Body.
var WithStreamingBody = abstract.WithStreamingBody
