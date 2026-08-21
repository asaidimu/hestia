# Hestia — Second-Pass Architecture Review

**Date:** 2026-08-21
**Scope:** Follow-up to the first-pass devnotes review (84 notes, `#arch-*`, `#review-*`, `#cruft-*`, etc.). This pass verifies the first-pass findings, adds notes in areas it didn't cover, and answers a specific set of architectural questions about production-readiness and P2P node deployment.
**Companion devnotes:** `#review2-20260821-001` through `#review2-20260821-005` (see `devnotes list --tags=review2`)

**A scoping note up front, because it matters for how to read this document:** Hestia is ~232 Go files plus a TypeScript client. A genuinely exhaustive line-by-line audit of all of it is a multi-week effort, not something done in one sitting. What follows is a *risk-prioritized* second pass — I went deep on auth/session/api-key security, the dispatch/runtime architecture, the update mechanism, and the HTTP interface, spot-checked the first pass for staleness, and read broadly (not deeply) across the remaining ~150 files that had zero devnotes coverage. I did not find anything in that broad pass severe enough to call out beyond what's below, but "did not find" is not the same as "verified clean" for files I only skimmed. Treat the sections below as load-bearing; treat the absence of a finding elsewhere as "not yet reviewed," not "reviewed and fine."

Also: `go vet`, `go build`, and `gofmt` could not be run against this codebase in this environment — `go.mod` declares `go 1.27rc1`, and the toolchain fetch is blocked by network egress rules here. Everything below is from reading, not from compiler/linter output. That includes the `core/runtime/di/container.go` generic-method syntax the tooling flagged — under go1.24's grammar `func (c *Container) Register[T any](...)` is illegal (methods can't have their own type parameters), but this may be valid under go1.27's language surface, which I can't check. **Verify this compiles under the actual toolchain before treating it as a real bug.**

---

## 1. Is it actually SOLID?

Short answer: **partially, and unevenly.** The *shape* of the codebase — small handler functions dispatched through a registry, policy bindings as data rather than code, a DI container, interfaces at package boundaries (`abstract.Dispatcher`, `abstract.CredentialsProvider`, `abstract.Notifier`) — is a genuine attempt at SOLID and mostly succeeds at the macro level. The violations are concentrated in a few specific, fixable places rather than being systemic.

### Where it holds up well
- **DIP is taken seriously at the top level.** Services are constructed via `abstract.MustResolve[T](rt)` against interfaces (`persistence.Persistence`, `*zap.Logger`), not concrete types. `NewCredentialsProvider` returns an `abstract.CredentialsProvider` interface, not a struct.
- **ISP is respected in the abstract package.** `abstract.Dispatcher` is one method. `abstract.Registry` is five focused methods. `abstract.BindingPolicyStore` is three methods for one job. Nobody's forced to implement a fat interface with no-ops.
- **OCP shows up in the dispatcher chain design** (`core/runtime/chain.go`) — new cross-cutting concerns (auth, rate-limiting, audit) are added as `DispatcherLink`s wrapping the base dispatcher, not as modifications to existing dispatch logic. This is the right pattern, undermined only by the `InsertBefore`/`InsertAfter` naming bug noted below (`#review2-20260821-003`).
- **Policy-as-data is a good SRP move.** `PolicyBindings()` functions return declarative `[]policies.Binding` rather than embedding authorization logic in handlers.

### Where it breaks down (concrete, not vibes)
- **SRP violation, systemic: the `inputs.go`/`outputs.go` vs `model/inputs.go`/`model/outputs.go` duplication.** The first pass already flagged this per-package (`#cruft-20260821-001` through `-024`) but it's worth naming as a *pattern*, not eight separate bugs: nearly every `core/system/*` package has two parallel definitions of its request/response shapes — one hand-written, one generated into `model/`. That's the same responsibility (define the wire contract for a domain) implemented twice, and it's the single largest source of "which one is current" risk in the codebase. This is the highest-leverage cleanup available before a production push, precisely because it's mechanical (delete the stale copy, repoint imports) rather than a redesign.
- **LSP is untestable in the one place it matters most for P2P: `abstract.Dispatcher`.** There is exactly one implementation family (the synchronous chain), so LSP compliance can't really be assessed — there's nothing to substitute against yet. This becomes the real question the moment you add a second implementation (a remote/P2P dispatcher): does it honor the *same contract* the local one does (ordering, error semantics, exactly-once vs at-least-once delivery)? See §3.
- **DIP violation: `core/internal/boot/database.go` hardcodes `sqlite3` and constructs the executor/query factory concretely** — there's no `Persistence` construction seam for a different backend. Fine for a single-node monolith; a real constraint for P2P (§4) where you may want per-node storage that isn't SQLite, or at minimum want to swap it in tests without spinning up a real file.
- **SRP violation, concrete instance: `UpdatesService.Check`** does version-check, staging (download + prepare), and admin notification as one atomic operation with no way to invoke a subset. See `#review2-20260821-004`.

**Bottom line:** the architecture *intends* SOLID and gets the macro decisions right. The debt is in execution — duplicated types, a couple of interfaces (`Dispatcher`) that are one level too coarse for where the project wants to go, and inconsistent error handling that the first pass already caught. None of it requires a rewrite; it requires the cleanup pass the first-pass notes already scoped out, plus the async/P2P seam discussed below.

---

## 2. Command decomposition: the updates example, and the pattern behind it

You asked specifically: *is the check-for-update command tied to staging?* Yes, confirmed — `#review2-20260821-004`. `UpdatesService.Check` (`core/system/updates/service.go:101`) always calls `checkAndStage`, which calls `updater.CheckForUpdate` **and** `updater.PrepareUpdate` (a download + disk write) in the same call, with no way to invoke just the first half. The registered message is even named `system:updates:check:create` with `Intent: abstract.Create` — the intent tells you it was never actually a read operation, despite the doc comment calling it "the read-only check."

This matters more than it looks like in isolation, because it's the same shape of problem you'll hit at fleet scale generally: **any handler that bundles "check state" with "mutate state" becomes unpollable.** A coordinator that wants a cheap, frequent, side-effect-free health signal across N nodes (update available? disk space ok? peer count healthy?) needs that to be a `Read` intent with zero writes. Right now the update-check is the one place this is visibly broken, but it's worth an explicit codebase-wide pass — before the P2P work — for any other handler where "get status" secretly does work. I didn't find another instance in the areas I covered, but I'd flag this as a specific thing to check for as you touch each `system/*` package, not just take my word that updates is the only one.

**Concrete split**, spelled out in the note: `system:updates:check:get` (Read, `CheckForUpdate` only, no side effects) and `system:updates:stage:create` (Create, today's `PrepareUpdate` + notify behavior). `RunScheduledCheck` (the cron path) keeps calling both back-to-back — a scheduled job doing check-then-stage in one shot is correct; an operator or another node polling status should not have to pay for staging just to ask a question.

---

## 3. Making the dispatchers async (and why this is *the* P2P question)

Confirmed, concretely: `abstract.Dispatcher` (`core/abstract/dispatcher.go`) is:

```go
type Dispatcher interface {
    Send(msg Message) (*Result, error)
}
```

One method, fully synchronous, no context parameter (context comes from `msg.Context()` instead — itself a slight smell, since it means the caller's cancellation and the message's identity are entangled). Every link in the chain — `LocalDispatcher`, `SecureDispatcher`, `RateLimitDispatcher`, the tenant/audit/recovery/bootstrap dispatchers — wraps the next one and calls `Send` as a direct blocking Go call. `LocalDispatcher.Send` (`core/runtime/local-dispatcher.go:52`) is the base case: `return entry.fn(msg.Context(), msg)`. No queue, no channel handoff, nothing that would survive a network hop.

This is *fine* for what the codebase is today — a single-process HTTP/CLI/Wails-desktop monolith, where `net/http` already gives you per-request concurrency for free and a blocking in-process call is genuinely the simplest correct thing. It is **the** architectural blocker for the P2P goal, because there's currently no seam where "route this message" can mean "route this message to a different node" instead of "call this local function." Full writeup with a concrete phased plan is in `#review2-20260821-005`; summarized:

1. **Add `ctx` as an explicit `Send` parameter** (`Send(ctx context.Context, msg Message) (*Result, error)`) instead of relying on `msg.Context()`. Small breaking change, but it's the prerequisite for everything after — a remote dispatcher needs the *caller's* cancellation/deadline, not just whatever was baked into the message at construction time.
2. **Add `AsyncDispatcher` alongside `Dispatcher`, not instead of it:**
   ```go
   type AsyncDispatcher interface {
       SendAsync(ctx context.Context, msg Message) (<-chan Result, error)
   }
   ```
   `LocalDispatcher` implements this trivially (run `entry.fn` in a goroutine, write to a buffered channel of size 1). A future `RemoteDispatcher`/`P2PDispatcher` implements the same interface by serializing `msg`, publishing over the network transport, and resolving the channel when a correlated response arrives or the context deadline fires. This is additive — existing synchronous call sites in HTTP/CLI/Wails don't need to change on day one; they can keep calling `Send`, and a `SyncFromAsync` adapter (blocks on the channel) lets `Send` be implemented *in terms of* `SendAsync` once you're ready to unify.
3. **Every `DispatcherLink.Wrap` needs an async-aware equivalent** so cross-cutting concerns (auth, rate-limiting, audit, tenant scoping) still apply uniformly regardless of whether the terminal handler resolves locally or over the wire. This is mechanical once #2 exists, but it's real work across every link in `core/runtime/`.
4. **The concept that's genuinely missing, not just deferred: locality.** Nothing in `abstract.Message` today says "this message's handler lives on node X" vs "this node." `TenantID` is the closest existing field and it answers a different question (whose data is this) not (where does this run). Before committing to the exact shape of `AsyncDispatcher`, you need to settle: how does a node know which peers can handle `system:collections:documents:create` for a given tenant/shard? Is that a gossip-based handler-advertisement protocol, a static routing table, a DHT? That answer shapes what `SendAsync` needs to accept (a target hint? does it resolve locality internally?) — I'd treat this as a design spike that happens *before* #2 is finalized, not after.

---

## 4. What else needs to change for P2P node deployment, beyond dispatch

The dispatcher is the biggest single blocker, but it's not the only thing standing between "single-node monolith" and "node in a mesh." Everything below is state that is currently node-local and in-memory or node-local on disk, with no cross-node story:

- **Persistence is hardwired to a single local SQLite file** (`core/internal/boot/database.go`) — `sql.Open("sqlite3", dsn)` with no abstraction seam for a different backend. For P2P this is arguably *correct* as a default (each node wants local storage) but it means there's currently zero replication/consensus/CRDT layer for state that needs to agree across nodes (e.g. policy bindings, tenant metadata, anything that isn't purely node-local like audit logs). Worth deciding early: which collections are per-node-authoritative (this node's own audit log, its own rate-limit counters) vs which need cluster-wide agreement (user accounts, policies, tenant config) — because those need fundamentally different mechanisms, and conflating them will hurt later.
- **Rate limiting (`core/runtime/ratestore/inmemory.go`) is in-process memory, per-node.** That's actually the right default for a mesh (each node protects itself), but if you ever want a *cluster-wide* rate limit (e.g. per-tenant API quota enforced across all nodes a tenant's traffic might land on), the current design can't express that at all — there's no shared-state or gossip-aggregation path.
- **The scheduler (`core/runtime/scheduler/scheduler.go`) runs cron jobs node-local with no coordination.** In a multi-node deployment, every node running the same cron job (e.g. `RunScheduledCheck` for updates) either needs to be idempotent-safe under N-way concurrent execution, or you need leader election / a "only one node runs this job" mechanism. Worth an explicit decision per scheduled job rather than assuming.
- **The audit buffer (`core/runtime/audit_buffer.go`) writes to node-local storage.** Fine if audit trails are meant to be per-node artifacts aggregated later by something else; a problem if the product story is "one queryable audit log for the whole mesh."
- **Sessions are HMAC tokens signed with a single shared secret** (`core/system/auth/session_service.go`) — this actually *helps* P2P, since any node holding the secret can validate a session issued by any other node without a round-trip. Worth keeping this property when you extend it — don't accidentally make session validation require calling back to an "auth node."

None of this needs to be solved before the async dispatcher work — but I'd sequence it as: (1) settle the async dispatcher interface + locality model (§3), since it's the foundation everything else routes through, (2) do an explicit per-subsystem pass (persistence, rate-limit, scheduler, audit) deciding node-local vs cluster-wide for each, (3) only then design the actual peer transport/discovery layer, because its requirements depend on what #1 and #2 decided needs to cross the wire.

---

## 5. Documentation endpoint and MCP auto-generation

Looked at `cmd/docs-server/main.go` and the `core/interface/http` doc/route machinery. What exists today: `cmd/docs-server` boots a full in-memory `hestia.Application` (with a **hardcoded admin password**, `"password123"` — fine for a docs server that never persists, but worth a comment saying so explicitly since it'll look alarming to anyone grepping for hardcoded credentials) purely to serve pre-rendered static HTML pages under `static/api/*.html`, generated at build time. That's a static-site generator bolted onto a live server process — heavier than it needs to be, and the generated docs are a snapshot, not live-introspectable.

**The good news: you're closer to auto-generated MCP than the docs situation suggests**, because the actual data model for it already exists and is richer than what the static docs use. `abstract.MessageRegistration` (`core/abstract/module.go:183`) carries `Name`, `Description`, `Input.Schema` (a full `definition.Schema`), and `Output *definition.Schema` for every registered message — this is essentially an MCP tool manifest already sitting in memory (`Interface.regs` in `core/interface/http/server.go`). An MCP endpoint is a thin translation layer:

- MCP `tools/list` ⟵ map over `regs`, skip `Internal: true` entries (same filter `installDispatcherRegistrations` already applies), translate `Name`/`Description`/`Input.Schema` into MCP's tool-schema JSON shape.
- MCP `tools/call` ⟵ same path `installRegistration` already builds (`buildDoc` → `ValidateInputDocument` → `dispatch.ValidateInputDocument` → `Dispatcher.Send`), just invoked from an MCP transport instead of the fasthttp trie.

Practically: implement it as a new `Transport` (the interface already exists — `core/abstract/transport.go:106`, one method, `Handle(pattern string, handler Handler)`) that speaks MCP's JSON-RPC framing instead of HTTP, sitting alongside `HTTPTransport` in `core/interface/http/transport_fasthttp.go` (or as a sibling package, `core/interface/mcp`). It can reuse `installRegistration`'s validation/dispatch pipeline almost entirely — the only new code is the MCP-shaped request/response translation and the manifest generation. This is a genuinely good fit for the existing registration model; the static docs generator, by contrast, is disconnected from it and duplicating information that's already structured elsewhere. I'd consider replacing the static-HTML docs server with something that serves the *same* schema-to-manifest translation as human-readable docs (OpenAPI-ish) rather than maintaining two separate "describe the API" code paths.

## 6. Adding manual aliases to the HTTP interface

Checked this directly — **there currently is no supported way to do this.** Every HTTP route is derived purely from the message name and its declared arguments:

```go
// core/interface/http/derive.go
func DeriveRoute(name string, arguments []abstract.ArgumentDefinition) string {
    parts := strings.Split(name, ":")
    path := "/" + strings.Join(parts, "/")
    for _, arg := range arguments {
        path += fmt.Sprintf("/{%s}", arg.Name)
    }
    return path
}
```

`system:auth:session:create` always becomes `/system/auth/session/create`, with no override point. `abstract.MessageRegistration` has no `Aliases []string` field, and `installRegistration` (`core/interface/http/register.go:63`) is the only call site that invokes `o.trans.Handle(pattern, ...)` — that method exists on the `Transport` interface and *would* let you register an arbitrary extra path, but `trans` is an unexported field on `Interface`, and `Config`/`Options` expose no hook to reach it. Concretely, today you cannot add `/login` as an alias for `/system/auth/session/create` without forking `installRegistration`.

**Two ways to fix it, different tradeoffs:**
- **Minimal, non-invasive:** add `Aliases []string` to `MessageRegistration`, and in `installRegistration`, after installing the derived route, loop over `Aliases` and call `o.trans.Handle(httpMethod+" "+alias, sameHandler)` for each. A few lines, fully backward compatible, no API surface change for existing registrations.
- **More general, matches the MCP-transport direction above:** expose a public `Interface.HandleRaw(pattern string, handler Handler)` (or similar) so alias registration doesn't require a `MessageRegistration` at all — useful for routes that don't map to a dispatcher message (health checks, custom webhooks). I'd do both: the `Aliases` field for the common case (same handler, different path), and the raw hook for the escape hatch.

---

## 7. Notes index for this pass

| ID | Priority | File | Summary |
|---|---|---|---|
| `#review2-20260821-001` | P1 | `core/system/auth/session_service.go:66` | `SessionService.Validate` doesn't check `ExpiresAt` itself |
| `#review2-20260821-002` | P2 | `utils/wails/dispatch.go:245` | Two call sites trust `ValidateSession` without checking expiry (symptom of #001) |
| `#review2-20260821-003` | P2 | `core/runtime/chain.go:34` | `InsertBefore`/`InsertAfter` never set `Name`, inserted links become unaddressable |
| `#review2-20260821-004` | P1 | `core/system/updates/service.go:101` | `check:create` conflates checking with staging; unpollable at fleet scale |
| `#review2-20260821-005` | P0 | `core/abstract/dispatcher.go:5` | `Dispatcher.Send` is fully synchronous — the core blocker for P2P dispatch |

All validated with `devnotes check` and synced with `devnotes index update`. Query with:
```
devnotes list --tags=review2
devnotes trace review2-20260821-005 --direction=out
```

Existing first-pass notes spot-checked for staleness (`#8uuufn`, `#ra2yqz`, plus the `#arch-*` and `#review-*` series read in full while researching this document) — all still accurately placed and described. No first-pass note appears stale or already fixed.
