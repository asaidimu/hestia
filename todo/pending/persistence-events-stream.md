# Persistence events stream

Status: **planned** — the implementation guide is written; the feature is not.

## Goal

Expose go-anansi persistence events (`document:create:success`,
`document:update:success`, `document:delete:success`, and optionally
`*:failed`) over SSE so clients can react to data changes without polling.

## Context

Read [`docs/guide/streaming.md`](../../docs/guide/streaming.md) first. It
documents the output-stream contract, the readiness barrier, `streamsink`,
backpressure, the persistence-event payload shapes, replay, and a complete
worked handler.

Existing patterns to copy:

- `core/system/notifications/service.go` — per-user filtered stream,
  subscribe-before-return, `streamsink`, readiness barrier + `ctx.Done()`.
- `core/system/audit/service.go` — collection-scoped persistence
  subscription (`DocumentCreateSuccess`).
- `core/system/logs/service.go` — cursor-based polling stream.
- `core/system/workflows/service.go` — subscribe-before-replay + timestamp
  dedupe.

Persistence events require an event bus. Hestia's
`core/persistence/sqlite.Default` always constructs one
(`core/persistence/sqlite/sqlite.go`); a custom `PersistenceFactory` must do
the same, otherwise `Subscribe` is a silent no-op.

## Design decisions to settle

- [ ] **Scope.** One collection per stream (query param `collection`) or all
      collections? Recommend collection-scoped first; a global feed needs a
      stronger authorization story.
- [ ] **Event selection.** Accept a list of event types (comma-separated query
      param); default to create/update/delete success. Reject unknown types
      with a `SystemError`.
- [ ] **Ordering.** One subscription per event type gives no cross-type
      ordering. If consumers need a single ordered feed, subscribe once to
      `document:*` and filter on `event.Type`.
- [ ] **Authorization.** `rule="root"` for the first cut, or
      `rule="authenticated"` plus a per-collection read check. The policy
      engine cannot express "may read collection X" generically today; decide
      before exposing this to non-root users.
- [ ] **Payload shape.** Emit the event envelope (`type`, `collection`,
      `operation`, `timestamp`, `output`, `error`, ...) and let clients pull
      `output.data`. Do not emit raw Go structs — the bus JSON round-trips.
- [ ] **Sensitive fields.** Add a `SanitizationRules()` entry for the events
      scope; event `input`/`output` can contain full documents.
- [ ] **Replay.** Off by default. If enabled, require a bounded
      `ReplayCursor`; unbounded replay from the start of the bus log is a
      denial-of-service risk.
- [ ] **Backpressure.** `TrySend` with a bounded buffer and a dropped counter;
      persistence events must not stall writers.

## Tasks

- [ ] Add `core/system/events/` (or extend `collections`) with
      `model/inputs.go`: `PersistenceStreamInput{Collection, Events, Replay}`.
      Bind query params with `input:"modifiers.<name>"` (not `payload.*` —
      `openStream` sends a bodyless `GET`); modifier values are strings, so
      keep `Events`/`Replay` as strings and parse them in the handler.
- [ ] Implement `StreamPersistence` per the guide, using
      `persistence.Subscribe` (collection filter in the callback) or
      `persist.Collection(ctx, name)` (returns `(Collection, error)`) then
      `.Subscribe` for the collection-scoped variant.
- [ ] Add `@hestia.register(intent="stream", output="model.PersistenceStreamOutput", ...)`
      and run `hestia service generate system events`; commit the generated
      `registrations.go` / `policies.go`.
- [ ] Add `sanitization.go` rules for the events scope and regenerate
      `core/system/gen_sanitization.go` so the `events` scope is registered.
- [ ] Wire the service into the module (`RegisterService` /
      `CollectServiceRegistrations`).
- [ ] Add a client store (`client/packages/core/system/events/`) using
      `openStream` + `createAsyncStream`; export it from the package index.
- [ ] Tests: service test (write → assert event), transport SSE test
      (framing + disconnect), client end-to-end test (login → stream → write
      → assert → abort).
- [ ] Document the endpoint in `docs/api/` and link the guide.

## Verification

```sh
go test ./core/system/events/ -race -count=1
go test ./core/interface/http/ -run TestSSE -race -count=1
cd client && bunx vitest run packages/core
```
