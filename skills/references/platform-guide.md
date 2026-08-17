# Platform decision guide — verified answers

Answers to the go/no-go questions a platform team must ask before building on
hestia. Each answer was verified against the code in this repo (paths cited).
Where the README or docs contradict the code, the code wins and the discrepancy
is flagged.

**How to use:** read the "Verdict" summary first. If a bolded item is a
showstopper for your use case, that's your decision. Then dive into sections.

---

## Verdicts on the go/no-go questions

| # | Question | Answer |
|---|---|---|
| 11 | Two apps in one process? | **No.** `boot.ProjectName` is a package-level var (`core/internal/boot/config.go`), and generated models are package singletons (`InitSystemUsersModel`). One hestia app per process. |
| 13 | Defense-in-depth or single gate? | **Single gate.** Authz is one `SecureDispatcher` chain link. Inserting a link *before* it (via `DispatcherChainFunc` + `InsertBefore("secure", ...)`) bypasses authorization for that path. |
| 15 | Can a module shadow built-ins? | **No.** `LocalDispatcher.RegisterHandler` errors on duplicate names (`core/runtime/local-dispatcher.go`). |
| 19 | Authorization granularity? | **Operation-level by default.** `RuleKey` → CEL rule evaluates identity (e.g. `administrator`). Resource context exists (`ResourceContextExtractor`, `ResourceIDField`) and collections pass collection name, but ownership-scoping is done manually in domain methods, not in the policy. |
| 29 | Consistency model? | SQLite WAL, single writer, ACID. `DatabaseInteractor` + `PersistenceFactory` can swap backends (only SQLite ships). |
| 38 | Scale ceiling? | Single-process, embedded SQLite + in-process dispatcher + bbolt blob index. No sharding, no multi-node routing. Fine for a single server; not horizontally scalable as shipped. |
| 41 | Custom decorators? | **Yes** — dispatcher chain links are the decorator mechanism (`DispatcherChainFunc`, `InsertBefore/InsertAfter/Remove`). No per-handler middleware, but chain links can rewrite messages (`NamespacedDispatcher` pattern). |
| 44 | Custom HTTP endpoints (webhook/graphql/ws)? | **No first-class mechanism.** Build a custom `runtime.Interface`, or add `httpapi.Middleware`/`StaticFS`. Raw custom routes aren't exposed through `SetupConfig`. |
| 71 | Event bus durable? | The persistence event bus is Pebble-backed (`go-events`), but **schedules** are in-memory cron restored from the DB at boot (`LiveSchedule.Init`) — missed ticks during downtime are not replayed. |
| 75 | Upgrade/rollback contract? | **None.** Alpha; "frequent breaking changes" (README). Migrations are forward-only, applied at boot, no rollback path. |
| 92 | License compat? | MIT (repo) with MIT/Apache-2.0 deps (go-anansi, go-iam, go-events, pebble, wails, go-sqlite3, zap). Commercially usable, but see lock-in (#101). |
| 101 | Lock-in? | **High.** Message-driven + anansi documents + policies + sanitization are all hestia-specific. Migrating off later means rewriting handlers, persistence, and authz. |

---

## 1. Architecture & message processing

- **Panic safety:** `RecoveryDispatcher` catches panics, returns a wrapped
  `PanicError` (survives `errors.Is`/`errors.As`), logs the stack — the server
  keeps running (`core/runtime/recovery-dispatcher.go`).
- **Processing guarantee:** at-most-once. `LocalDispatcher.Send` invokes the
  handler directly; no retries, no redelivery. Idempotency-Key header sets the
  message ID + trace ID (`core/interface/http/register.go`) but nothing dedupes
  on it.
- **Message name:** hard 4-segment validation at registration
  (`validateMessageName`, `core/internal/boot/app.go`); warnings, not fatal.
- **Long-running work:** request-scoped. Streaming exists
  (`ResultKindDocumentChannel` / `ResultKindBlobChannel`); no background
  worker abstraction beyond the scheduler.
- **Cross-node:** none. In-process dispatcher only; no horizontal message
  routing.
- **Module lifecycle:** `RegisterModules` topo-sorts by `Dependencies()`
  (Kahn, cycle/missing → error) and aborts `Setup` on the first failure.
  `Start`/`Stop` run in dependency / reverse-dependency order
  (`core/internal/boot/app.go`).
- **Chain order** is deliberate and fixed, but mutable by name:
  `bootstrap → secure → ratelimit → throttle → tenant → blob → recovery → audit`
  (`core/system/module.go`). Custom links can be inserted anywhere — including
  *before* `secure` (authz bypass risk).
- **Result pooling:** documents/results are pooled; `result.Release()` returns
  them. The HTTP layer auto-releases. Non-HTTP dispatch paths must release
  manually or leak pooled memory.
- **Two apps per process:** unsupported (see verdict #11).

## 2. Security, authn & authz

- **Identity resolution (HTTP):** session cookie → API key → anonymous, in
  that order; an `X-Api-Key` header with a session elevates it (audited as
  on-behalf-of) (`core/interface/http/middleware.go`).
- **Session tokens are NOT JWT.** They are HMAC-SHA256-signed base64url
  `payload.signature` with a 16-byte truncated signature
  (`core/system/auth/session_service.go`). README's "JWT" claim is wrong.
  Contains sid/uid/iat/exp/crt/tv.
- **Revocation:** `token_version` in the token is checked against the user's
  current version; password change bumps it and invalidates old sessions
  (`middleware.go`). Stateless — no server-side session store.
- **CSRF:** SameSite=strict + HTTPOnly + Secure cookies by default
  (`core/runtime/config.go`). No CSRF token; if you set SameSite=none, you own
  the risk.
- **Authz model:** `SecureDispatcher` → `PermissionManager.Resolve(msg)` maps
  message→ruleKey, `AccessController.Can` evaluates CEL
  (`core/runtime/secure-dispatcher.go`). Rules are **not cached**
  (`CacheTTL: 0`, `core/system/module.go`) and live in a LiveRepository that
  auto-syncs on DB writes — policy changes apply immediately.
- **Default rule if a binding has no RuleKey:** `administrator`
  (`core/system/gen_features.go`). So an unbound new message defaults to
  admin-only — secure-by-default.
- **System identity bypass:** any identity whose permissions include a
  `system:*` scope is treated as system and **skips the secure link entirely**
  (`IsSystemIdentity`, `core/runtime/auth.go`). Granting `system:*` to a user
  is granting everything.
- **`Internal: true` is a routing flag, not a security boundary.** It excludes
  a message from HTTP (`core/interface/http/register.go`) but the handler stays
  registered and dispatchable in-process.
- **Denied requests are NOT audited.** The audit link is the innermost link
  (`... → recovery → audit → LocalDispatcher`), so an authz denial that
  short-circuits at `secure` never reaches the audit log. Audit covers handled
  messages only — a real gap for compliance.
- **Audit durability:** buffered ring (4096) with async flush, exponential
  backoff retry, and a **fail-open circuit breaker** that drops entries to
  stderr under pressure (`core/runtime/audit_buffer.go`) — audit is best-effort.
- **Secret sanitization:** global response-level redaction for
  password/hash/secret/token/api-key/credential patterns + per-collection
  overrides (`core/internal/boot/persistence.go`). Keep secrets out of output
  projections structurally too.
- **Rate limiting:** per-policy `Identity` dimension (user/ip/apikey/tenant/
  global) with token bucket; a policy that switches dimension on the request
  key (`buildRateLimitKey`, `core/runtime/rate-limit.go`). Throttling triggers
  template-driven action messages (e.g. disable a user after repeated login
  failures — see the seeded defaults in `core/system/gen_features.go`).
- **Bootstrap:** only `session:create` and `bootstrap:password:set` are
  `BootstrapSafe`; everything else is gated by `BootstrapDispatcher`
  (`core/runtime/bootstrap-dispatcher.go`). Ephemeral key is in-memory only.

## 3. Persistence & data

- **Backend:** SQLite (WAL) via anansi; the whole `query.DatabaseInteractor`
  interface is the seam for custom backends (`core/internal/boot/persistence.go`,
  `core/internal/boot/database.go`). `PersistenceFactory` swaps the backend;
  only SQLite adapters ship.
- **Transactions:** supported at the interactor level
  (`DatabaseInteractor.StartTransaction/Commit/Rollback`); `base.Persistence`
  exposes them. Document-store semantics — no SQL joins.
- **Query DSL:** builder with `Where/Eq/Neq/Lt/Gt/OrderBy/Limit/Offset`,
  offset pagination with `IncludeTotal` (`core/system/collections/query.go`).
  No aggregates, text search, or SQL passthrough exposed at the feature layer.
- **Unique constraints:** field-level `unique: true` only; no composite
  constraints.
- **Migrations:** anansi `migrate generate` (forward-only), applied at boot by
  `migrations.Apply`. No downgrade/rollback path. Codegen regenerates all
  feature schemas (`anansi.json` glob).
- **Blobs:** on-disk object store with bbolt index (`asaidimu/blobs`), resumable
  uploads via a staging manager with a reaper that discards sessions idle >6h
  (`core/system/blobs/store/service.go`). Filesystem-backed, not S3.
  Durability = the whole data dir; back up DB + blobs + idx together.
- **Document ID:** `_id_` auto-injected by codegen; `data.DocumentIDField`.
- **Scale:** single-process embedded. No sharding; blob index is a local bbolt
  file.

## 4. Extensibility

- **Custom decorators:** dispatcher chain links (`DispatcherChainFunc`,
  `DispatcherChain.InsertBefore/InsertAfter/Remove`,
  `core/runtime/chain.go`). Links may rewrite messages
  (`NamespacedDispatcher`). Position matters for security (see #13).
- **Custom interfaces:** implement `runtime.Interface`
  (`Start/Restart/Shutdown`), or use `BuildInterfaces` to replace the default
  HTTP+CLI pair. You keep full control but lose the built-in route/cookie/
  session plumbing unless you reuse it.
- **Custom HTTP surface:** middleware (`httpapi.Middleware`) + `StaticFS`
  (+ `NoRefreshCommands`) per HTTP interface. Raw custom routes are not a
  documented extension point.
- **Auth swap:** `abstract.CredentialsProvider` is the seam; the system module
  builds the default from `SessionService`. No shipped OIDC/SSO adapter.
- **Events:** go-events pub-sub powers the persistence event bus; the
  `notifications` feature is mailer-backed (SMTP). No generic domain-event bus
  for your own events out of the box.
- **CEL rules:** rules can be added/edited at runtime from the DB (compiled
  CEL); Go-native defaults exist (`GoDefaultRules`). Custom Go functions in
  CEL are not exposed as an extension point.
- **Plugins:** compile-time Go modules only; no dynamic loading.
- **TS SDK:** mirrors the Go API over HTTP with reactive stores + auto-refresh
  JWT handling; no plugin mechanism for your module's custom methods.

## 5. Configuration & operations

- **Validation:** none at boot for env values — an unparseable `PORT` is
  silently ignored and the default used (`core/runtime/config.go`). No config
  schema validation.
- **Hot reload:** none; restart required.
- **Deployment:** single binary; graceful shutdown via
  `app.Shutdown(ctx)` (modules in reverse dep order, then interfaces). No
  built-in drain/traffic-draining; `Shutdown` is as graceful as the transport
  implements.
- **Health:** module `Health(ctx) any` exists but no default HTTP health
  endpoint is wired for it.
- **Observability:** zap structured logs + lumberjack rotation; request/trace
  ID correlation. **No metrics / OpenTelemetry / tracing export.**
- **Migrations in CI:** not supported — migrations apply at boot only.
- **Config surface:** `SessionSecret` is the only required field; everything
  else has sane defaults (see the config table in SKILL.md §3).

## 6. Performance & reliability

- **No published benchmarks**; per-request cost is the full chain
  (auth middleware → 8 dispatcher links). Audit is the only async/buffered
  link; everything else is synchronous in-request.
- **Schedules:** in-memory cron (`netresearch/go-cron`) with
  `SkipIfStillRunning` + `Recover` chains; schedule *definitions* persist in
  the DB and are re-registered at boot (`LiveSchedule.Init`). Missed runs
  while down are lost.
- **Notifications:** DB-persisted with a 30-day read TTL and expiry sweep;
  expiry-filtered on read (`core/system/notifications/model.go`).
- **Mail:** `DialAndSend` per email; SMTP failure surfaces as an error. No
  queue/retry/dead-letter for mail.
- **Concurrency:** single-process; SQLite single-writer. No optimistic locking
  — last-write-wins on `UpdateFrom`.
- **Shutdown:** `Close()` syncs logs, closes persistence and blob service;
  `Stop()` stops scheduler. Audit buffer drains on close.
- **Testability:** generated models expose `DangerouslyResetXxxModel()`;
  test-harness patterns are in `feature-template.md` §7.

## 7. Multi-tenancy

- **Not automatic.** `TenantDispatcher` injects `tenant_id` into the context;
  features/models must filter by it themselves (e.g. notifications filter by
  `user_id`+`tenant_id`). Nothing stops a feature from ignoring the tenant.
- No tenant-scoped blobs/schedules isolation by default.

## 8. SDK & licensing

- **TS SDK** mirrors the Go API over HTTP with reactive stores + auto-refresh
  JWT handling. Server docs are discoverable via `GET /api/system/core/docs/list`;
  no OpenAPI/Swagger export found.
- **License:** MIT repo; deps MIT/Apache-2.0. No copyleft blockers for
  commercial closed-source products.
- **Risk:** solo-maintainer project (asaidimu), alpha, pre-1.0, frequent
  breaking changes, small ecosystem. In-tree code is forkable; go-anansi/go-iam/
  go-events are the real bet-the-company dependencies.
- **Lock-in:** message contracts + anansi document model + policies +
  sanitization + SDK are hestia-specific. Plan the exit before you need it.

---

*Every claim above was verified against this checkout of the hestia repo.
Re-verify against the version you pin — things change daily.*