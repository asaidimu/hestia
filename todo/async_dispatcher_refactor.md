# Async-Native Dispatcher Refactor

**Source:** devnotes `#review2-20260821-005` (P0) — Dispatcher.Send is synchronous end-to-end.
**Goal:** callback-native async dispatch as the default path, with the seam needed for
future remote/P2P dispatch and a follow-up durable-execution lane (go-events v2).

## Design decisions (settled)

- Single interface; async is the default. No sync/async prefixes anywhere.
  ```go
  type CompletionFunc func(ctx context.Context, res *Result, err error)
  type Dispatcher interface {
      Send(ctx context.Context, msg Message, onComplete CompletionFunc) error
  }
  ```
- Signature mirrors `MessageHandler` exactly: handler produces `(*Result, error)`,
  completion consumes it. No Outcome struct — rejection paths stay allocation-free
  (`nil, err`) and the whole `errors.As` ecosystem keeps working.
- Contract:
  - Exactly-once completion for ephemeral sends, including panics (recovered,
    delivered as `*PanicError`) and ctx-cancel-before-start (`err = ctx.Err()`).
  - `onComplete == nil` → fire-and-forget (explicit nil at call sites).
  - Non-nil return from Send = synchronous pre-check rejection; callback never fires.
  - Callbacks run on dispatcher-owned goroutines; no ordering guarantee between
    concurrent completions; slow callbacks never block the pipeline.
- `msg.Context()` remains the request-metadata source of truth (identity, tenant,
  trace). Explicit ctx governs lifecycle only.
- Blocking sugar lives in `runtime/dispatch` (abstract stays pure):
  `Await(ctx, d, msg)` and `Dispatch(ctx, disp, in)` rebuilt on Await.
- Panic recovery absorbed into `LocalDispatcher.Send`; `RecoveryDispatcher` deleted
  (a link's recover() cannot catch panics in the terminal's goroutine).
- Stream intent: feed input channel at dispatch time with ctx-aware select
  (fixes `#mem-20260821-003` goroutine leak in http streamChannel).
- Semver-major: external `DispatcherLink.Wrap` implementers break.

## Status

- [*] Epic 1 — async-native dispatch (this changeset)
  - [*] Phase 1: unify dispatchMessage/genericMessage (`#arch-20260821-006`)
  - [*] Phase 2: abstract types (`CompletionFunc`, new Send signature)
  - [*] Phase 3: migrate impls + call sites (~13 impls, ~100 sites incl. tests)
  - [*] Phase 4: LocalDispatcher goroutine/call + panic recovery + exactly-once
  - [*] Phase 5: links tenant→blob→bootstrap→secure→ratelimit→throttle→audit;
        delete RecoveryDispatcher
  - [*] Phase 6: transports — HTTP/Wails via Await; CLI/identity-provider blocking
  - [*] Phase 7: stream intent re-spec + ctx-aware producer
  - [*] Phase 8: cleanup, doc.go contract, resolve notes, go test -race ./core/...
- [*] Epic 1.5 — fire-and-forget transports (landed standalone)
  - [*] MessageRegistration.FireAndForget flag (json: fire_and_forget)
  - [*] dispatch.Enqueue(ctx, disp, in) (id, error) — Send(ctx,msg,nil) sibling
        of Dispatch; ID = Idempotency-Key when present else fresh UUIDv7
  - [*] HTTP installRegistration branch → 202 {"data":{"id","status":"accepted"}}
  - [*] Annotation vocabulary: fire_and_forget="true" attr (parser + emitter +
        both testdata fixtures; unknown attrs already ignored → back-compat safe)
  - Note: until Epic 2 lands, accepted work is best-effort — a crash between
        ack and execution loses the callback with no retry signal. FireAndForget
        registrations are the natural WithDurable() defaults once Epic 2 lands.
- [ ] Epic 2 — durable lane via github.com/asaidimu/go-events/v2 (follow-up changeset)  - [ ] Phase 9: `WithDurable()` option through Send (no-op without config)
  - [ ] Phase 10: runtime/durable package — envelope codec (document payloads only),
        dispatch:intent / dispatch:outcome topics, executor w/ stable subscriber ID,
        EventTimeout disabled, DLQ→audit, ArchiveCompactor on
  - [ ] Phase 11: SetupConfig.DurableExec wiring; shutdown ordering
        (stop intake → drain checkpoints → close bus)
  - [ ] Phase 12: convert throttle actions, notification sends, scheduler jobs,
        updates checks; kill-and-restart resume test
  - Note: durability guarantees accepted work completes, NOT that missed cron
        ticks fire while down. Catch-up scheduling is out of scope.

## Key files

- core/abstract/dispatcher.go, system_options.go
- core/runtime/{local-dispatcher,bootstrap-dispatcher,secure-dispatcher,rate-limit,
  throttle,tenant-dispatcher,namespaced-dispatcher,recovery-dispatcher,
  access-log-dispatcher}.go
- core/runtime/dispatch/{dispatch,message}.go (+ new await.go)
- core/system/module.go (chain assembly), core/internal/boot/builder.go
- core/interface/http/{register,identity_provider}.go, core/interface/cli/orchestrator.go
- utils/wails/dispatch.go

## Verification

- Existing suite updated in place = regression harness (identical assertions).
- New units: exactly-once under panic/cancel/error; sync rejection spawns no
  goroutine; days-long callback simulation (no parked waiter).
- `go test -race ./core/runtime/...`; full-boot tests stay skipped per AGENTS.md.
