package abstract

import "context"

// @note #review2-20260821-005 issue P0 #review,#arch,#p2p : Dispatcher.Send is synchronous end-to-end with no async/remote-capable variant
//
// Every implementation in the chain -- LocalDispatcher, SecureDispatcher,
// RateLimitDispatcher, AuditLog/TenantDispatcher, RecoveryDispatcher,
// BootstrapDispatcher -- wraps the next Dispatcher and calls Send() as a
// direct, blocking Go function call that returns (*Result, error) on the
// same goroutine. LocalDispatcher.Send (core/runtime/local-dispatcher.go:52)
// is the base case: it does `return entry.fn(msg.Context(), msg)` with no
// queue, no channel handoff, and no notion of "this might take a network
// round-trip." Every transport (HTTP, CLI, Wails) calls Dispatcher.Send and
// blocks the calling goroutine until it returns.
//
// This is fine for a single-process monolith (net/http already gives you
// per-request concurrency via goroutines, so throughput isn't the problem)
// but it is the central architectural blocker for the P2P node deployment
// goal: there is currently no seam where a message could be routed to a
// *different* node instead of a local handler. Adding that seam later
// without an async story means either (a) blocking a goroutine per in-flight
// remote call indefinitely, or (b) a breaking change to this interface.
//
// Resolution -- concretely, in order of increasing scope:
//  1. Add a context.Context parameter to Send itself: Send(ctx, msg) instead
//     of relying on msg.Context(). This is a prerequisite for #2 and #3
//     because a remote dispatcher needs the caller's ctx for cancellation/
//     deadline propagation independent of what's embedded in the message.
//  2. Introduce a second interface alongside Dispatcher, not a replacement:
//     type AsyncDispatcher interface {
//     SendAsync(ctx context.Context, msg Message) (<-chan Result, error)
//     }
//     LocalDispatcher can implement this trivially by running entry.fn in a
//     goroutine and writing to a buffered chan of size 1. A future
//     RemoteDispatcher (or P2PDispatcher) implements the same interface by
//     serializing msg, publishing it over the network transport, and
//     resolving the channel when a correlated response arrives (or the ctx
//     deadline fires). Existing synchronous callers are unaffected; new
//     transports (or existing ones once ready) adopt SendAsync incrementally.
//  3. Every DispatcherLink.Wrap in the chain needs an Async-aware equivalent
//     so cross-cutting concerns (auth, rate-limit, audit, tenant scoping)
//     still apply uniformly whether the terminal handler is local or remote.
//     Concretely: change DispatcherLink.Wrap to operate on the AsyncDispatcher
//     interface, and provide a `SyncFromAsync` adapter (blocks and unwraps
//     the channel) so the existing Dispatcher call sites in HTTP/CLI/Wails
//     don't all need to change on day one.
//  4. Message routing needs an explicit locality concept before any of this
//     is useful: something has to decide "is this message's target handler
//     local to this node, or does it need to go over the wire to node X".
//     abstract.Message today has no notion of a target node/peer -- TenantID
//     is the closest thing, and tenant != node. This is a bigger design
//     question (service discovery / handler advertisement across the mesh)
//     that should be settled before committing to the exact shape of #2.
type MessageHandler func(context.Context, Message) (*Result, error)

type Dispatcher interface {
	Send(msg Message) (*Result, error)
}

type IntentType string

const (
	IntentTypeCommand IntentType = "COMMAND"
	IntentTypeQuery   IntentType = "QUERY"
)

type HandlerInfo struct {
	Name          string     `json:"name"`
	IntentType    IntentType `json:"intent_type"`
	Description   string     `json:"description"`
	Enabled       bool       `json:"enabled"`
	BootstrapSafe bool       `json:"bootstrap_safe"`
}

type Registry interface {
	RegisterHandler(name string, handler MessageHandler, info HandlerInfo) error
	GetHandler(name string) (MessageHandler, error)
	DeleteHandler(name string) error
	ListHandlers() []HandlerInfo
	SetHandlerEnabled(name string, enabled bool) error
}

type ResourceContextExtractor interface {
	ResourceContext() any
}
