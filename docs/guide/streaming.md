# Streaming

Hestia has two independent streaming axes. They solve different problems and
can be combined, but most features only need one.

| Axis | Declared by | Request | Response |
|---|---|---|---|
| **Streamed input** | a `<-chan dispatch.Item[TIn]` handler parameter | NDJSON body, one document per line | normal result |
| **Streamed output** | `intent="stream"` in `@hestia.register` | normal request | SSE (`text/event-stream`) fed by `abstract.Result.DocumentChannel` |

This guide covers the output path in detail because that is what a
persistence-event feed needs. It includes a worked example that streams
**go-anansi persistence events** (`document:create:success`,
`document:update:success`, `document:delete:success`, ...).

## Output streams: the contract

A stream handler is an ordinary message handler with `intent="stream"`:

```go
// @hestia.register(
//   name="system:events:persistence:stream",
//   intent="stream",
//   rule="root",
//   description="Stream persistence events",
//   output="model.PersistenceStreamOutput",
// )
func (s *EventsService) StreamPersistence(
    ctx context.Context,
    msg abstract.Message,
    input *model.PersistenceStreamInput,
) (*abstract.Result, error) {
    // ...
}
```

`hestia service generate <module> <service>` turns that annotation into a
`MessageRegistration` with `Intent: abstract.Stream` and
`Handler: dispatch.Handle[TIn](s.StreamPersistence)`. The input type comes
from the `*TIn` parameter. Declare `output=` as well: when the method returns
`*abstract.Result` the signature alone cannot express the wire shape, and the
docs listing (`GET /api/system/core/docs/list`) reads the registration's output
schema.
The HTTP transport derives `GET /system/events/persistence/stream` from the
message name and answers with SSE.

The handler must return quickly with a result whose `DocumentChannel` (or
`BlobChannel`) is still open:

```go
return &abstract.Result{DocumentChannel: sink.Channel()}, nil
```

The transport then owns the response: it writes each document as

```text
data: {"data": <sanitized document>}
```

followed by a blank line, and emits `: keepalive` comments while idle. Each
document is sanitized at the egress edge with the sanitizer registered for the
feature scope, which the dispatcher derives from the message name's second
segment (`system:events:persistence:stream` → `events`). Do not pre-render
JSON yourself; return a `*document.Document`.

### The readiness barrier

For `intent="stream"`, `dispatch.Dispatch` creates a buffered channel and,
**after the handler returns**, feeds it one `abstract.StreamItem{}` and closes
it. `msg.InputChannel()` exposes that channel.

Two consequences matter:

1. Subscribe to your event source **inside the handler, before returning**.
   Waiting for the barrier first leaves a window where events are lost.
2. The barrier is not "the stream is connected"; it only means the handler
   finished. Use it to detect a stream that never went live (for example the
   transport failed after the handler returned), then park on `ctx.Done()`
   for the real lifetime.

The `ctx` handed to a stream handler is transport-owned. It is cancelled when
the SSE writer stops — client disconnect, flush error, or the stream drained.
Do **not** rely on `fasthttp.RequestCtx.Done()`: fasthttp only fires that on
server shutdown.

## Lifecycle rules

These rules are not optional; violating them leaks goroutines, loses events,
or crashes the process.

### 1. Subscribe before returning

```go
sink := streamsink.New[*document.Document](64)

subID := source.Subscribe(ctx, persistence.SubscriptionOptions{
    Event: persistence.DocumentCreateSuccess,
    Callback: func(_ context.Context, event persistence.PersistenceEvent) error {
        sink.TrySend(toDocument(event))
        return nil
    },
})
```

Subscribe synchronously in the handler. Do not start a goroutine that
subscribes later.

### 2. Use `streamsink`, never a bare channel

`go-events` cancels a subscriber's drain context on `Unsubscribe` but does
not wait for an in-flight callback. Closing a bare channel while a callback
is sending panics the process.

`core/runtime/streamsink.Sink[T]` serializes `Send`/`TrySend` against an
idempotent `Close`. Use it for every stream producer.

### 3. Tear down in the right order

```go
go func() {
    cleanup := func() {
        source.Unsubscribe(ctx, subID) // stop the source first
        sink.Close()                   // then release the channel
    }

    select {
    case <-msg.InputChannel(): // stream went live
    case <-ctx.Done():         // never went live
        cleanup()
        return
    }

    <-ctx.Done() // client disconnect
    cleanup()
}()
```

Always `Unsubscribe` before `Close`. `Close` is idempotent, so cleanup is
safe even if both branches race.

### 4. Choose a backpressure policy

- `TrySend` drops the newest value when the buffer is full and counts it in
  `Sink.Dropped()`. Use it for notifications, audit, and logs, where a slow
  consumer must not stall the writer.
- `Send(ctx, v)` blocks until the value is delivered, the context is done, or
  the sink closes. Use it when gaps are worse than backpressure (workflow
  timelines).

## go-anansi persistence events

The persistence layer emits an event for every operation at start, success,
and failure. Two subscription levels are available:

| Level | Call | Scope |
|---|---|---|
| Collection | `col.Subscribe(ctx, opts)` | events for one collection; the collection name is filtered automatically |
| Persistence | `persist.Subscribe(ctx, opts)` | every collection; filter in your callback |

`col` comes from `persist.Collection(ctx, name)`, which returns
`(Collection, error)`; generated model collections
(`*model.SystemNotificationss`) embed one, so `s.model.Subscribe(...)` is the
same API.

### Event types

`persistence.PersistenceEventType` is a string. The document-level types are:

```text
document:create:start    document:create:success    document:create:failed
document:read:start      document:read:success      document:read:failed
document:update:start    document:update:success    document:update:failed
document:delete:start    document:delete:success    document:delete:failed
```

`persistence.PersistenceEvents` (`"*"`) matches all of them. Other families
are also available: `collection:*`, `persistence:read:*`, `migrate:*`,
`rollback:*`, `transaction:*`, `telemetry`, and `persistence:lifecycle:*`.

`SubscriptionOptions.Event` accepts an exact type or a glob/regex pattern
(`document:*`, `*:success`). Subscribe once per exact type to avoid filter
overhead; a pattern plus a callback filter is the alternative.

### What the callback receives

`persistence.SubscriptionOptions.Callback` is
`func(ctx context.Context, event persistence.PersistenceEvent) error`.
`PersistenceEvent` carries:

```go
Type          PersistenceEventType // "document:create:success"
Timestamp     int64                // Unix ms, captured at operation start
Operation     string               // "createOne", "createMany", "update", ...
Collection    *string              // nil for non-collection events
Input         any
Output        any
Error         *string
Issues        []common.Issue
Query         any
TransactionID *string
Duration      *int64               // ms
Context       map[string]any
```

**The event bus JSON round-trips events.** `Emit` marshals the event and the
adapter unmarshals it before calling the handler, so `Input` and `Output`
arrive as generic JSON values (`map[string]any`, `[]any`, `float64`, ...),
not Go structs. Plan for that in the callback:

| Event | `Output` shape |
|---|---|
| `document:create:success` | `createOne`: `{"status": "CREATED", "data": {<created document>}}` → read `output.data`. `createMany`: an array of those objects. |
| `document:create:failed` | `nil`; use `Error` / `Issues` |
| `document:update:success` | `{"data": [<documents>], "count": n, "total": n}` (`count`/`total` may be omitted) |
| `document:read:success` | `{"data": [<documents>], "count": n, "total": n}` (`count`/`total` may be omitted) |
| `document:delete:success` | a number (`float64`) — the deleted count |

`*:start` events fire before the operation and carry the input but no output;
`*:failed` events carry `Error` and a nil `Output`. (`Issues` exists on the
struct but the current emitter never populates it.)

### Replay

By default a subscription is **live-only**: it sees events published after
`Subscribe` returns. Set `Replay: true` to receive history first, then live
events:

```go
subID := persist.Subscribe(ctx, persistence.SubscriptionOptions{
    Event:       persistence.DocumentUpdateSuccess,
    Replay:      true,
    ReplayCursor: time.Now().Add(-5 * time.Minute), // optional bound
    Callback:    cb,
})
```

`ReplayCursor` accepts `time.Time`, `uuid.UUID`, or a UUID string. A nil
cursor replays from the beginning of the bus's retained log, which can be
large. For a user-facing feed, bound it or keep replay off and seed the
client from a normal query.

### Authorization and noise

A persistence-level subscription sees **every** collection, including
internal ones (`_audit_log_`, `_notifications_`, ...). Prefer a
collection-scoped subscription, or require an explicit collection name and
filter in the callback. The policy engine has no generic "may read collection
X" check today, so start with `rule="root"` (as in the example) or add an
explicit authorization check before exposing the feed to other roles.

## Worked example: a persistence-event feed

Goal: `GET /system/events/persistence/stream?collection=_user_` streams
create/update/delete events for one collection until the client disconnects.

### Input DTO

The input schema is what the transport binds. Keep the fields small and
explicit; the handler validates them.

```go
// core/system/events/model/inputs.go
type PersistenceStreamInput struct {
    Collection string `json:"collection" input:"modifiers.collection"`
    Events     string `json:"events" input:"modifiers.events"` // comma-separated
    Replay     string `json:"replay" input:"modifiers.replay"` // "true" / "false"
}

// PersistenceStreamOutput documents the wire shape for the docs listing. The
// stream actually emits the event envelope below, one document per SSE frame.
type PersistenceStreamOutput struct {
    Type       string `anansi:"type"`
    Collection string `anansi:"collection"`
    Operation  string `anansi:"operation"`
    Timestamp  int64  `anansi:"timestamp"`
}
```

`hestia service generate system events` reads the struct through
`dispatch.SchemaFromTypeWithTag[model.PersistenceStreamInput]("input")` and
registers the route. `Events` defaults to create/update/delete when empty.

> **Query params bind through `modifiers.*`, not `payload.*`.** `payload.*`
> is filled from the request body, and `openStream` sends a bodyless `GET`.
> Use `modifiers.<name>` for query-string inputs (as the blob routes do for
> `overwrite`), or `arguments.<name>` for path segments. Modifier values are
> strings, so booleans and lists are parsed in the handler.

### Handler

Subscribe to each requested type. The callback is called on the event bus's
drain goroutine; keep it fast and non-blocking (`TrySend`). Filter by
collection even when using `persist.Subscribe`, because a persistence-level
subscription sees every collection.

Each type is a separate subscription, so **cross-type ordering is not
guaranteed**. If the consumer needs one ordered feed, subscribe once to
`document:*` and filter by `e.Type` in the callback instead of looping.

```go
func (s *EventsService) StreamPersistence(
    ctx context.Context,
    msg abstract.Message,
    input *model.PersistenceStreamInput,
) (*abstract.Result, error) {
    if input.Collection == "" {
        return nil, common.NewSystemError(
            "EVENTS_COLLECTION_REQUIRED", "collection is required")
    }

    replay, err := strconv.ParseBool(input.Replay)
    if input.Replay != "" && err != nil {
        return nil, common.NewSystemError(
            "EVENTS_REPLAY_INVALID", "replay must be true or false")
    }

    var events []string
    for _, e := range strings.Split(input.Events, ",") {
        if e = strings.TrimSpace(e); e != "" {
            events = append(events, e)
        }
    }
    if len(events) == 0 {
        events = []string{
            string(persistence.DocumentCreateSuccess),
            string(persistence.DocumentUpdateSuccess),
            string(persistence.DocumentDeleteSuccess),
        }
    }

    sink := streamsink.New[*document.Document](128)
    subIDs := make([]string, 0, len(events))

    for _, eventType := range events {
        eventType := eventType
        subIDs = append(subIDs, s.persist.Subscribe(ctx,
            persistence.SubscriptionOptions{
                Event:  persistence.PersistenceEventType(eventType),
                Replay: replay,
                Callback: func(_ context.Context, e persistence.PersistenceEvent) error {
                    if e.Collection == nil || *e.Collection != input.Collection {
                        return nil
                    }
                    sink.TrySend(persistenceEventDocument(ctx, e))
                    return nil
                },
            },
        ))
    }

    go func() {
        cleanup := func() {
            for _, id := range subIDs {
                s.persist.Unsubscribe(ctx, id)
            }
            sink.Close()
        }

        select {
        case <-msg.InputChannel(): // handler returned, stream is live
        case <-ctx.Done():         // never went live
            cleanup()
            return
        }

        <-ctx.Done() // client disconnected or stream ended
        cleanup()
    }()

    return &abstract.Result{DocumentChannel: sink.Channel()}, nil
}
```

### Event → document

```go
func persistenceEventDocument(ctx context.Context, e persistence.PersistenceEvent) *document.Document {
    m := map[string]any{
        "type":      string(e.Type),
        "timestamp": e.Timestamp,
        "operation": e.Operation,
    }
    if e.Collection != nil {
        m["collection"] = *e.Collection
    }
    if e.Input != nil {
        m["input"] = e.Input
    }
    if e.Output != nil {
        m["output"] = e.Output
    }
    if e.Error != nil {
        m["error"] = *e.Error
    }
    if e.Duration != nil {
        m["duration"] = *e.Duration
    }
    return document.NewRecordView(m, ctx)
}
```

The transport sanitizes this document before writing it. Because the scope is
`events`, add `core/system/events/sanitization.go` with a
`SanitizationRules()` function and register it in
`core/system/gen_sanitization.go`. Event `input`/`output` can contain full
documents, so mask the same fields the owning feature masks. `Input`/`Output`
are already generic JSON values.

## Consuming the stream from TypeScript

`Transport.openStream` is the single SSE entry point. It applies the
transport's default headers (including `X-Api-Key`) and defaults
`credentials: "same-origin"`.

```ts
const controller = new AbortController()

await client.openStream(
  "/system/events/persistence/stream?collection=_user_",
  {
    onOpen: () => console.log("connected"),
    onMessage: (data) => {
      const event = JSON.parse(data) // { data: { type, collection, ... } }
      console.log(event.data)
    },
    onError: (err) => console.error(err),
    onClose: () => console.log("closed"),
  },
  { signal: controller.signal },
)

// later
controller.abort()
```

For pull-style consumption, wrap it with `createAsyncStream`:

```ts
import { createAsyncStream } from "@asaidimu/hestia"

interface PersistenceEvent {
  type: string
  collection?: string
  operation: string
  timestamp: number
  output?: unknown
  error?: string
}

const source = createAsyncStream<PersistenceEvent>()
const controller = new AbortController()

void client.openStream(
  "/system/events/persistence/stream?collection=_user_",
  {
    onMessage: (data) => source.push(JSON.parse(data).data),
    onError: () => source.end(),
    onClose: () => source.end(),
  },
  { signal: controller.signal },
)

for await (const event of source.stream()) {
  console.log(event.type)
}
```

Cross-origin cookie sessions need `credentials: "include"` **and** the
origin must be on the server's allowlist; the CORS layer then returns
`Access-Control-Allow-Credentials: true`.

## Testing

Three layers are worth covering:

1. **Service test** — dispatch the stream with `testutil.NewPersistence(t)`,
   perform a write for the same collection, and assert the document arrives
   on `result.DocumentChannel`. See
   `core/system/notifications/stream_test.go`.
2. **Transport test** — install the route and assert SSE framing, disconnect
   cancellation, and idle keepalives. See
   `core/interface/http/sse_disconnect_test.go`.
3. **Client end-to-end** — boot `cmd/test-server`, open the stream, trigger a
   write, assert delivery, then abort. See
   `client/packages/core/system/notifications/notifications.infra.test.ts`.

Use `t.Cleanup` to release the result and cancel the context. Prefer polling
with a deadline over fixed sleeps.

## Streamed input (NDJSON)

The other axis is for bulk operations where the request body is a stream of
documents. Declare the handler with a receive-only channel of
`dispatch.Item[TIn]`:

```go
// @hestia.register(
//   name="system:users:user:import",
//   intent="create",
//   rule="root",
//   description="Bulk-import users from an NDJSON stream",
// )
func (s *UsersService) ImportUsers(
    ctx context.Context,
    msg abstract.Message,
    items <-chan dispatch.Item[model.ImportUserInput],
) (*abstract.Result, error) {
    for item := range items {
        if item.Err != nil {
            // per-item policy: abort or collect
            continue
        }
        // item.Value is a bound, validated model.ImportUserInput
    }
    return &abstract.Result{}, nil
}
```

The transport decodes one JSON document per line and validates each against
the input schema; `dispatch.HandleInputStream` binds each document into
`dispatch.Item{Value: ...}` (or `Item{Err: ...}` when framing, validation, or
binding fails). The channel is producer-owned; the transport closes it when
the body ends or framing fails. The generator sets `Input.Streaming: true`
from the channel parameter and emits
`dispatch.HandleInputStream[TIn](s.Method)`.

Streamed input and streamed output are orthogonal: a handler can consume
`items` and return a `DocumentChannel` if it needs to report progress.

## Implementation checklist

- [ ] Annotation has `intent="stream"`; the method returns `(*abstract.Result, error)`.
- [ ] The handler subscribes synchronously before returning.
- [ ] The producer writes through `streamsink.Sink[T]`.
- [ ] Cleanup calls `Unsubscribe` before `Close`, and `Close` is idempotent.
- [ ] The goroutine waits on `msg.InputChannel()` and `ctx.Done()`.
- [ ] Backpressure policy is explicit (`TrySend` vs `Send`).
- [ ] The document is a `*document.Document`; no manual JSON.
- [ ] A sanitization rule exists for any sensitive event payload.
- [ ] The route is generated (`hestia service generate <module> <service>`).
- [ ] A service test, a transport test, and a client end-to-end test exist.
