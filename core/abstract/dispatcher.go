package abstract

import "context"

// @note #review2-20260821-005 issue resolved P0 #review,#arch,#p2p : Dispatcher.Send is synchronous end-to-end with no async/remote-capable variant
// Resolved by the async-native dispatcher refactor. Dispatcher.Send is now Send(ctx, msg, onComplete CompletionFunc) error: acceptance returns synchronously (non-nil error = pre-check rejection, callback never fires); execution runs on a dispatcher-owned goroutine and the completion callback fires exactly once — seconds or days later — with (*Result, error). msg.Context() remains the request-metadata source of truth; explicit ctx governs lifecycle/cancellation before terminal start. Panic recovery was absorbed into LocalDispatcher (a link's recover() can never catch a panic in the terminal's goroutine), so RecoveryDispatcher was deleted from the chain and codebase. Blocking sugar lives in runtime/dispatch (Await/Dispatch). The remote/P2P seam is now this interface: a future RemoteDispatcher implements Send by publishing over the network and invoking the stored completion on correlated reply — no parked waiter goroutine per in-flight call. Durable execution via go-events v2 planned as follow-up (todo/async_dispatcher_refactor.md Epic 2).
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

// CompletionFunc receives the final outcome of an accepted dispatch exactly
// once, on a goroutine owned by the dispatcher — possibly long after Send
// returned (remote or durable execution). A nil res with non-nil err signals
// failure; panics are recovered by the dispatcher and delivered as
// *runtime.PanicError. onComplete == nil means fire-and-forget.
type CompletionFunc func(ctx context.Context, res *Result, err error)

// Complete delivers an outcome to onComplete unless it is nil
// (fire-and-forget).
func Complete(onComplete CompletionFunc, ctx context.Context, res *Result, err error) {
	if onComplete == nil {
		return
	}
	onComplete(ctx, res, err)
}

// Dispatcher accepts messages for execution. Send arranges for the message's
// handler to run asynchronously; onComplete (if non-nil) is invoked exactly
// once when execution finishes.
//
// Contract:
//   - A non-nil return value is a synchronous pre-check rejection (handler
//     missing/disabled, unauthorized, rate-limited, not bootstrapped). No
//     goroutine is started and onComplete never fires.
//   - A nil return means the message was accepted: onComplete fires exactly
//     once, even on panic or internal error.
//   - ctx governs acceptance and cancellation before the terminal handler
//     starts; msg.Context() remains the request-metadata source of truth.
type Dispatcher interface {
	Send(ctx context.Context, msg Message, onComplete CompletionFunc) error
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
