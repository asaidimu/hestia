# Streaming + full codegen coverage — working todo

Companion to `HandlerGenerics.md` (which carries the full design, §3.2–3.6 and
§4). This doc is the **execution checklist**: what blocks auto-generated
registrations, policies, and sanitization policies for feature services, what
was decided, and what landed. Update the checkboxes as work completes.

## Goal

Feature services get all three generated artifacts from `@hestia.register`
annotations alone:

1. `registrations.go` — including streaming ops (input / output / both)
2. `policies.go` — bindings for every op, streaming included
3. `sanitization.go` — field-mask rules emitted from a declared source
   (blocked on a format decision; see §D below)

## Blocker inventory (verified against `97d9695`)

| # | Blocker | Where | Status |
|---|---|---|---|
| ~~B1~~ | `abstract.Input` has no `Streaming` field — a registration cannot declare streamed input | `core/abstract/module.go` | **landed** |
| ~~B2~~ | `Message.InputChannel()` carries `data.Documenter`; no error slot; never populated with real items | `core/abstract/message.go` | **landed** (re-typed to `<-chan StreamItem`) |
| ~~B3~~ | `DispatchInput` has no `DocumentStream`; Stream-intent channel is a readiness barrier only | `core/runtime/dispatch/dispatch.go` | **landed** |
| ~~B4~~ | Transport contract is buffered-only: `abstract.Request` has no `BodyStream`; `Transport.Handle` has no per-route options | `core/abstract/transport.go` | **landed** |
| ~~B5~~ | fasthttp transport has no streaming route path (`routeEntry.streaming`, `ctx.RequestBodyStream()`) | `core/interface/http/transport_fasthttp.go` | **landed** |
| ~~B6~~ | No NDJSON per-item document producer (`streamDocuments`), no `BuildInputDocumentFromPayload`, no `installRegistration` streaming branch | `core/interface/http/{doc,register}.go` | **landed** |
| ~~B7~~ | `dispatch.Item[TIn]` / `StreamHandler[TIn]` / `HandleInputStream` / `HandleOutputStream` missing | `core/runtime/dispatch/bind.go` | **landed** |
| ~~B8~~ | Generator refuses streaming annotations (`render.go` hard error, `gen.go` skip) — no registration, no policy binding | `cmd/hestia/core/gen/{gen,render}.go` | **landed** |
| ~~B9~~ | `SanitizationDispatcher` never sanitizes `DocumentChannel` (SSE) or `Blob` results | `core/runtime/sanitization-dispatcher.go` | **landed** (channel; blob N/A — binary) |
| B10 | Sanitization rules are hand-written per feature + hand-maintained aggregator (`gen_sanitization.go` is not generated) | `core/system/*/sanitization.go`, `core/system/gen_sanitization.go` | **open** — needs format decision (§D) | B9 landed |
| B11 | Duplex (`HandleStream`) needs a persistent-connection transport (WebSocket/gRPC/HTTP2) — fasthttp recycles `RequestCtx` when `serveHTTP` returns | spec §3.6 | **deferred by design** |
| B12 | Codegen contract erosion: hand-edits to DO-NOT-EDIT files (`97d9695` touched `schedules/registrations.go`); audit/logs hand-registered | workflow | **mitigated** by B8 landing (audit stream reg can now be generated) |
| B13 | Per-item error policy / backpressure undecided | spec §8 | **decided** — see §C |
| B14 | Client SDK + `routes.gen.ts` + docs have no streaming-op awareness | `client/`, `cmd/gen-routes` | open, follow-up |

## A. Contract (§3.2–3.3) — LANDED

- `abstract.Input` gains `Streaming bool` (orthogonal axis to `Verb`; a bulk
  import is `Create` + `Streaming`).
- `abstract.StreamItem{Doc *document.Document; Err error}` — concrete type, per
  §1 of HandlerGenerics.md.
- `Message.InputChannel() <-chan StreamItem` (was `<-chan data.Documenter`).
  The Stream-intent readiness barrier is re-typed and its historical
  post-completion ordering preserved.
- `DispatchInput.DocumentStream <-chan StreamItem` — producer-owned (the
  transport owns close semantics); mutually exclusive with `Document`.

## B. Transport (§3.4) — LANDED

- `abstract.Request.BodyStream io.Reader` — set **instead of** `Body` for
  streaming routes; `ctx.Request.Body()` must never be touched on that path.
- `RouteOptions{StreamingBody bool}` + `WithStreamingBody()`; `Handle` grows
  variadic options (backward compatible). `routeEntry{handler, streaming}`.
- `streamDocuments(ctx, r, input, pool)`: NDJSON decode loop; framing errors
  (non-EOF decode failures) emit one `StreamItem{Err}` and end the stream —
  the byte stream is unrecoverable at that point; per-item envelope built once
  via `BuildInputDocumentFromPayload` (constant prefix, substituted payload).

## C. Handler adapters (§3.5) + decided semantics — LANDED

- `Item[TIn]{Value TIn; Err error}`; `StreamHandler[TIn]`;
  `HandleInputStream[TIn]` (bind loop over `msg.InputChannel()`, per-item
  errors surfaced as `Item.Err`); `HandleOutputStream[TIn] = Handle[TIn]`
  (naming only).
- **Per-item validation decision**: schema validation happens per item in the
  producer (`streamDocuments`) via `ValidateInputDocument` — fail-closed per
  S-14 (validator init failure aborts the stream, not silently passes).
  Validation failures become `StreamItem{Err}`, so the handler owns the
  abort-vs-collect policy (the spec's open question). Framing/decode errors
  end the stream (unrecoverable byte position).
- **Backpressure decision**: unbuffered `streamDocuments` channel — the HTTP
  body reader blocks on a slow consumer (natural backpressure), per spec §8.

## D. Sanitization policies (B10) — needs a format decision before codegen

Current: each feature hand-writes `SanitizationRules() *sanitize.FieldMaskConfig`;
`core/system/gen_sanitization.go` is a hand-maintained aggregator. Candidate
formats: (a) an annotation attribute (`@hestia.sanitize(...)` blocks on input
and output model types), (b) a JSON sidecar next to the schema, (c) struct
tags on the model. Until picked, sanitization codegen stays open; the runtime
gap (B9) is fixed independently so generated stream ops are safe the moment
they land.

## E. Generator (§4) — LANDED

- Emission targets unchanged (`registrations.go`, `policies.go`). Streaming
  annotations now emit: `Handler: dispatch.HandleInputStream[TIn](s.Method)`,
  `Input.Streaming: true`, and a normal policy binding. The
  "HandleInputStream not landed" error and the skip counter are deleted.
- Output-only streaming registrations keep `dispatch.Handle[TIn]` (the
  existing `raw`-result path audit uses today is expressible as annotations).
- Golden testdata gains a streaming case.

## F. Rollout order (this change)

1. Contract: `StreamItem`, `Input.Streaming`, `InputChannel` re-type,
   `DispatchInput.DocumentStream` (+ ~10 mechanical test-mock updates)
2. `bind.go`: `Item`, `StreamHandler`, `HandleInputStream`, `HandleOutputStream`
3. Transport: `BodyStream`, `RouteOptions`, fasthttp streaming path,
   `streamDocuments`, `installRegistration` branch
4. Generator: streaming emission + goldens
5. Sanitization: `DocumentChannel` pass-through sanitizer
6. Regression gate: build / vet / test all-green before push

## G. Explicitly out of scope (this change)

- `HandleStream` (duplex) — needs WebSocket/gRPC/HTTP2 transport first; spec
  §3.6 recommendation stands. Annotations for duplex signatures fail loudly.
- Sanitization-rule codegen (B10) — format decision pending (§D).
- Client SDK streaming ergonomics (B14).

---

## H. Landing notes (2026-08-31)

- All layers landed and gated: build / vet / test green; new suites —
  `dispatch/stream_test.go` (bind + forwarding + abandonment), 
  `http/streaming_test.go` (end-to-end NDJSON over the installed route,
  framing termination, per-item validation continuation, missing-body guard),
  `runtime/sanitization_stream_test.go` (scope stash + consume), and the
  generator golden now includes a streaming registration that is proven to
  compile via TestGenerateCompiles.
- **Envelope convention pinned by tests**: each NDJSON item plays the role of
  the request body — item fields bind through the `payload` section
  (`input:"payload.<field>"` tags), identical to buffered bodies. The fixture
  and golden document this.
- The HTTP-layer SSE transform previously called `d.Sanitize()` with **no
  context** — a silent no-op (scope lives on the message context). Fixed via
  the metadata stash (`StreamSanitizeArgs`), which also fixes dispatcher-side
  output sanitization running against the caller's scope-less ctx.
- Known residual (documented, bounded): notifications/audit service-side
  producers park on `ctx.Done()` (server shutdown) after writer abandonment —
  pre-existing, service-side; the dispatcher-side stash introduces no new
  goroutine.
