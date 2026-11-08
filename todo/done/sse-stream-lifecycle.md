# SSE stream lifecycle (S-19 follow-up)

Status: **landed** — transport fix, panic-safe cleanup, client migration, and follow-up tests complete.

## Symptom

The Playwright probe opened an `EventSource`/`fetch` SSE stream and got
`200 text/event-stream` followed immediately by end-of-stream with
`eventCount: 0`. Server logs showed events delivered into
`DocumentChannel` and subscriptions never cleaned up. Affected every SSE
surface that goes through `managedStream`: notifications, audit, logs,
workflows.

## Root cause

`writeSuccess` wrapped the `managedStream` path in `defer ms.Close()` and then
called `writeEventStream`. fasthttp's `SetBodyStreamWriter` only *registers* the
callback (`NewStreamReader` starts it asynchronously), so the deferred
`Close()` ran when the handler returned — before the writer consumed anything.
`Close` closed `ms.done`, the producer exited and closed `streamCh`, and the
writer ranged an already-drained channel. The response ended immediately.

## Landed

- `core/interface/http/transport_fasthttp.go`
  - `managedStream.Close` now runs **inside** the `SetBodyStreamWriter`
    callback (`defer onDone()`), not in `writeSuccess`. Flush error or drained
    stream releases the producer at the correct time.
  - `Response.ImmediateHeaderFlush = true` for SSE, so the `200` and headers
    are visible before the first event (and idle streams are not mistaken for
    hung requests).
  - SSE keepalive comment (`: keepalive`) every `writeTimeout/2` (capped at
    15s). This keeps idle streams alive through proxies and turns a dead
    client into a flush error so teardown runs.
  - Per-request write timeout via `HTTPTransport.headerReceived`: fasthttp sets
    `Server.WriteTimeout` as an **absolute** deadline for the whole response,
    which killed every SSE stream at 60s once delivery worked. The server now
    runs with `WriteTimeout: 0` and `headerReceived` re-applies the 60s bound to
    non-SSE requests only, so stream responses never get a write deadline.
    (An earlier attempt pushed the deadline forward from the writer goroutine;
    it was removed because fasthttp's `perIPConn` wrapper nils its embedded
    `net.Conn` on close, making `ctx.Conn().SetWriteDeadline` panic the whole
    process after a client disconnected.)
- `core/interface/http/register.go`
  - Stream-intent routes are wrapped by `withStreamLifetime`. It gives the
    handler a transport-owned `streamContext` whose `Done()` closes when the
    body writer exits; `managedStream.Close` cancels it. Services parked on
    `ctx.Done()` (notification/audit/log/workflow) now unsubscribe on
    disconnect instead of leaking zombie subscriptions.
  - `streamContext` deliberately avoids `context.WithCancel` on the
    `*fasthttp.RequestCtx`: `cancelCtx.cancel` walks the parent
    (`parentCancelCtx` → `Value`) after fasthttp has recycled the RequestCtx,
    which is a data race. The server-shutdown channel is captured while the
    request is alive instead.
  - `managedStream` gained `finished`, closed by the producer goroutine, so
    tests can assert producer exit deterministically instead of racing a
    channel send.
- `core/interface/http/sse_disconnect_test.go`
  - `TestSSEEventsReachClient` — events produced after the handler returns
    reach the client (fails on the old `defer Close`).
  - `TestSSEDisconnectCancelsHandlerContext` — disconnect cancels the handler
    context (no zombie subscription).
  - `TestSSESurvivesWriteTimeoutAndEmitsKeepalive` — idle stream survives past
    `Server.WriteTimeout` and emits keepalives.
  - `TestHeaderReceivedExemptsOnlySSE` — the per-request timeout policy.
  - `TestSSEWriterSurvivesDisconnectWithPerIPConn` — regression test for the
    `perIPConn` crash: the writer must not touch the connection object.
  - Existing `TestSSEProducerReleasedOnClientDisconnect` now waits on
    `managedStream.finished`.
- `client/packages/core/core/client.ts`
  - `HttpTransport.openStream` now applies the transport's `defaultHeaders`
    (notably `X-Api-Key`) and sends `credentials: "same-origin"` by default.
    `StreamOptions.credentials` allows opting into `"include"` for credentialed
    cross-origin streams. Streams previously bypassed the network client
    entirely, so API-key auth never reached the server.
  - Tests in `client/packages/core/system/blobs/blobs.test.ts` cover both.

## Verification

```sh
go build ./...
go test ./core/... -count=1 -timeout 900s
go test ./core/runtime/streamsink/ ./core/runtime/logsink/ \
  ./core/system/notifications/ ./core/system/audit/ ./core/system/logs/ \
  ./core/system/workflows/ -race -count=1
cd client && bunx vitest run packages/core
```

All pass (Go full suite; TS 300 passed, 4 skipped). The positive transport
test fails against the pre-fix transport (reproduced).

End-to-end probe (TS client against `cmd/test-server` on `:8070`): log in,
`notifications.stream(...)`, `notifications.create(...)`; observed
`onOpen` → `onMessage` with the created document → `abort()` → `onClose`, and
the server stayed healthy (no panic) after the disconnect.

## Follow-ups landed

- [x] **Panic-safe stream cleanup.** Added `core/runtime/streamsink` — a
      bounded `Sink[T]` whose `Send`/`TrySend` are serialized against an
      idempotent `Close`, so a callback still in flight when a stream tears down
      can never send on a closed channel. Notifications, audit, logs and
      workflows now use it; `Unsubscribe` runs before `Close`. Covered by
      `sink_test.go` (including `-race` close/send stress tests).
- [x] **Logs stream stall.** The poller compared `ring.Len()` (capped at
      capacity) against the previous length, so once the 1000-entry ring filled
      live streaming stopped forever. `RingBuffer` now keeps a monotonic
      sequence per slot and exposes `Snapshot(n)` / `Since(cursor)`; the log
      stream captures the cursor atomically with the replay. Tests in
      `core/runtime/logsink/ring_test.go`.
- [x] **Subscribe ordering.** Audit now subscribes before the readiness
      barrier, matching notifications, so creates between the handler returning
      and the stream going live are not lost. Notifications keeps its existing
      (correct) ordering. Both tear down when the client disconnects before the
      stream goes live.
- [x] **Audit/logs client transport.** Added
      `client/packages/core/core/async-stream.ts` (`createAsyncStream`) and
      migrated `system/audit/store.ts` and `system/logs/store.ts` off the native
      `EventSource` global onto `Transport.openStream`. All four SSE surfaces
      now share one auth path (API key + credentials) and work outside the
      browser.
- [x] **Workflow replay/live gap.** `WorkflowsService.Stream` subscribes to the
      bus before replaying, buffers events emitted during the replay, then
      drains them de-duplicated by timestamp. Live events use `sink.Send` so
      timeline events are not dropped under backpressure.
- [x] **Stale diagnostic tests.** Replaced
      `core/system/notifications/stream_repro_test.go` with
      `stream_test.go`: `TestNotificationStreamStaysOpenAfterDispatch` and
      `TestNotificationStreamDeliversCreatedNotification`.
- [x] **In-repo end-to-end coverage.** Added
      `client/packages/core/system/notifications/notifications.infra.test.ts`
      (login → open stream → create for `auth_disabled` → assert delivery →
      abort), replacing the uncommitted external Playwright probe.
- [x] **Cross-origin credentials.** `cors()` now sets
      `Access-Control-Allow-Credentials: true` for allowlisted origins too, so
      cookie-authenticated cross-origin SSE works from trusted dashboards;
      `TestCORSAllowlist` asserts it. Callers opt in with
      `credentials: "include"`.

## Remaining notes

- Drop policy is still "drop newest" for notifications/audit/logs
  (`TrySend`, counted by `Sink.Dropped()`; logs warn on teardown). Workflows use
  blocking `Send` because timeline gaps are worse than backpressure.
- The external Playwright harness remains uncommitted; the in-repo integration
  test above is the maintained coverage.
