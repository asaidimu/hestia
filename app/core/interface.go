package core

import "context"

// Interface is a transport layer that receives external input (HTTP requests,
// CLI args, WebSocket messages, gRPC calls), translates them to Messages,
// dispatches them through the system's Dispatcher, and returns the Result
// to the external caller.
//
// Built-in implementations:
//   - api.Interface  — HTTP/RPC transport (routes → messages → HTTP responses)
//   - cli.Interface  — CLI flag parser (flags → bootstrap messages → stdout)
//
// Custom implementations follow the same pattern: receive external input,
// call disp.Send(msg) to dispatch, return the result to the caller.
type Interface interface {
	Start(bootstrapped bool)
	Restart(bootstrapped bool)
	Shutdown(ctx context.Context) error
}
