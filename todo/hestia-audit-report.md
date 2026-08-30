# Hestia — Deep-Dive Code Audit: Architecture & Latent Bugs

| | |
|---|---|
| **Repo** | `github.com/asaidimu/hestia` @ `3368956` (v1.10.1, 2026-08-30) |
| **Scope** | 312 Go files (31,189 non-test LOC + 13,272 test LOC), TypeScript client SDK, CLI, codegen tooling |
| **Method** | Static review: file-by-file read of runtime/auth/HTTP/storage paths, two parallel deep sweeps, **every line number below hand-verified against source** |
| **Reader** | You, the maintainer. Blunt mode. |

---

## 0. Executive Summary

The bones are good — dispatcher-chain decorator pattern, policy-as-data, codegen with golden tests, honest self-review docs. The execution has holes, and some of them are the kind that end up on a disclosure blog.

The single worst finding: **any authenticated user can execute any registered operation as the system identity, with no authorization check and no audit trail, via `system:schedules:schedule:create`** (S-1). It is not a policy misconfiguration — it is a structural bypass of your entire security chain, because `LiveSchedule` holds the raw terminal dispatcher.

The second-worst cluster: **the auth lifecycle doesn't close its loops**. The default session secret is a string constant in a public repo (S-2), the login rate limiter can never fire because two layers compare different string formats (S-3), and logout is a literal `return nil` (S-4). Together these make credential attacks cheap and session theft permanent.

**Count: 4 critical, 6 high, 9 medium, ~8 low findings; 15 architecture findings.**

| Severity | Bugs | Architecture |
|---|---|---|
| Critical | S-1, S-2, S-3, S-4 | — |
| High | S-5 … S-10 | A-1, A-2 |
| Medium | S-11 … S-19 | A-4 … A-6, A-8, A-12, A-14 |
| Low | S-20, S-21 | A-3, A-7, A-9 … A-13, A-15 |

**Fix before anything else (all verified, most are S-effort):** S-2, S-3, S-5, S-6, S-10, S-11, S-14, S-12 — then S-1 (M) which requires a small design decision.

---

## How to read this report

- **Severity**: *critical* = remotely exploable compromise of the security model; *high* = exploable preconditions or serious resource failure; *medium* = needs a specific precondition or grants partial advantage; *low* = hygiene that will bite later.
- **Effort**: S = hours, M = 1–3 days, L = a design decision plus a week-ish.
- Everything was verified by reading the source at HEAD. Line numbers refer to `3368956`. Two items are explicitly flagged as needing a runtime check (no Go toolchain in the audit sandbox): fasthttp `RequestCtx.Done()` semantics (S-19) and `asaidimu/updater` provider internals (S-9).

---

# Part 1 — Latent Bugs: Security, Leaks, Correctness

## S-1 · CRITICAL — Any authenticated user can execute any operation as SYSTEM, bypassing the entire dispatcher chain

**Where:** `core/system/schedules/liveschedule.go:124-127`, `core/system/schedules/validate.go:48-62`, `core/system/schedules/policies.go:10`, `core/system/module.go:85`, `core/internal/boot/app.go:147-149`

The schedule fire path dispatches with a **system identity through the raw terminal dispatcher** — not the chain:

```go
// liveschedule.go:124-127
sysCtx := runtimecontext.SystemContext(ctx)
ls.log.Info("schedule: dispatching", ...)
msg := dispatch.NewMessage(message, sysCtx, docInput)
result, err := dispatch.Await(sysCtx, ls.disp, msg)
```

And `ls.disp` is the **raw** `*runtime.LocalDispatcher` (`module.go:85` ← `boot.Dispatcher()` at `app.go:147-149` returns `a.Disp` directly). The chain — `bootstrap → secure → ratelimit → throttle → tenant → blob → audit` (`module.go:465-478`) — is never entered. So: no authorization, no rate limiting, no tenant stamping, **no audit entries**.

Meanwhile schedule creation is open to any authenticated user:

```go
// schedules/policies.go:10 — generated bindings
{Name: "system:schedules:schedule:create", RuleKey: "authenticated", ...},
```

And the create/update-time target validation checks only registration existence and input schema — **never whether the caller may invoke the target**:

```go
// validate.go:53-62 — finds the registration, checks schema, that's all
for i := range *regs {
    if (*regs)[i].Name == message { reg = &(*regs)[i]; break }
}
if reg == nil {
    return common.NewSystemError("SCHEDULE_UNKNOWN_MESSAGE", ...)
}
```

**Exploit:** user registers `message = "system:users:user:create"` with input `{email, password, name, permissions: ["administrator"]}` and `cron = "@every 1h"`. Next tick, hestia creates an admin user — as the system identity, with zero audit rows. Same recipe works for `system:policies:*`, `system:settings:set`, any admin operation. Note that even *if* it went through the chain, `SystemContext` satisfies `IsSystemIdentity`, which makes `SecureDispatcher` skip the policy check (`secure-dispatcher.go:73-79`) — so the identity choice is part of the bug, not just the dispatcher choice.

**Fix (M):**
1. At create/update time, resolve the target op's policy rule and evaluate it **against the caller's claims**; reject if the caller couldn't invoke the op directly.
2. Store the creator's user ID; at fire time dispatch with a **restricted identity derived from the creator** (not `SystemContext`) — so revocation/disable works and policy still applies.
3. Alternatively/additionally: hard allow-list of schedulable messages (`bootstrap_safe` style), default-deny.
4. Route schedule fires through the full chain so audit + rate limit apply.

## S-2 · CRITICAL — Hardcoded default session secret; sessions AND reset tokens are forgeable

**Where:** `core/runtime/config.go:66`, `:151`, `core/system/module.go:90-91`

```go
// config.go:66
DefaultSessionSecret = "3ecb5a2ef5014f88-8a00-8227db8b7298"
// config.go:151 — DefaultConfig() silently ships it
SessionSecret:  DefaultSessionSecret,
```

Session tokens are `base64(json).base64(HMAC-SHA256(payload)[:16])` (`session_service.go:107-116`). Anyone who read this repo can forge `{"uid":"<any user id>","exp":9999999999,"tv":0}` — and `token_version` is 0 for every user who never had a profile update bump it (only call site: `system_user_utils.go:153`). Worse, the reset-token key is *derived from the same secret*:

```go
// module.go:90-91
sessionSvc := auth.NewSessionService(m.cfg.SessionSecret)
resetSecret := m.cfg.SessionSecret + ":reset"
```

So password-reset tokens are forgeable too → take over any account by email.

**Fix (S):**
1. Remove `DefaultSessionSecret`. On first boot generate 32 bytes from `crypto/rand`, persist under the data dir (0600), refuse to start if unreadable.
2. If you want dev convenience: log a loud warning + derive from datadir, never a constant.
3. Derive per-purpose keys with HKDF (`session`, `reset`) instead of string concatenation.

## S-3 · CRITICAL — The login rate limiter can never fire; password checking is unthrottled

**Where:** `core/interface/http/middleware.go:137-144` vs `core/interface/http/transport_fasthttp.go:192`, `core/interface/http/register.go:31`

The middleware compares the request's *operation* against the *message name*:

```go
// middleware.go:137-144
if req.Operation == msgSessionCreate {          // "system:auth:session:create"
    key := authRateLimitKey + req.ClientIP
    _, allowed, err := authRateLimiter.CheckAndConsume(...)
    if err == nil && !allowed { return Response{Status: 429}, runtime.ErrRateLimited }
}
```

But the transport sets `Operation` to the HTTP route, not the message name:

```go
// transport_fasthttp.go:192
Operation: method + " " + path,   // "POST /api/system/auth/session/create"
```

These strings are never equal → **the brute-force gate is dead code on the real HTTP path**. Unit tests that invoke the middleware with a raw message name mask it (`register.go:31`). And nothing else throttles password paths: no `RateLimitPolicy` is seeded for `system:auth:session:create`, and `system:auth:token:elevate` + `system:auth:password:confirm` are `rule="public"` password-checking endpoints (`auth/service.go:170-176`, `:282-288`) with no limiter at all. Unlimited credential stuffing, and `elevate` mints a live 5-minute API key on success.

**Fix (S):**
1. Match on the route pattern (the `noRefreshOps` map already does exactly this — `register.go:78`), or map route→message name before the middleware.
2. Add the same gate to `token:elevate` and `password:confirm`.
3. Add a regression test that drives the **transport** (not the bare middleware) and asserts 429 after burst.
4. Bonus: `authRateLimiter` is a package global (`middleware.go:20`) — move it into `Interface` while you're there.

## S-4 · CRITICAL — Logout is a no-op; the token blocklist is dead schema; session IDs never rotate

**Where:** `core/system/auth/service.go:133-135`, `core/system/auth/session_service.go:50-65`, `_token_blocklist_` collection (migration + schema only)

```go
// auth/service.go:133-135
func (s *AuthService) DeleteSession(ctx context.Context, msg abstract.Message,
        input *model.DeleteSessionInput) error {
    return nil
}
```

The `_token_blocklist_` collection is created by a migration and has a generated model — **nothing reads or writes it** (repo-wide grep: schema/migration files only). Server-side "logout" therefore does not exist; `register.go` only clears the browser cookie. Compounding it, `SessionService.Refresh` re-issues the **same** `SessionID` (`session_service.go:53`) and nothing ever consumes a session on password change — so a stolen cookie is valid until the 8h absolute TTL regardless of what the user does, and there's no revocation primitive for admins either.

**Fix (M):**
1. `DeleteSession`: write `sid` (+ `uid`, `exp`) to the blocklist; enforce in `SessionService.Validate` or the middleware's validate path. Honor `ExpiresAt` on blocklist rows to prune.
2. Rotate `SessionID` on `Refresh` (the sliding window already re-issues tokens — `middleware.go:92-99`).
3. Bump `token_version` on logout-all / admin-disable (already exists as a mechanism — `system_user_utils.go:168-170`).

## S-5 · HIGH — Ephemeral bootstrap key: full admin, never expires, non-constant-time compare

**Where:** `core/system/auth/adapters.go:46-61`, `core/system/module.go:289-292`, `todo/first_run_api_key.md` (acknowledged open)

```go
// adapters.go:47-49
if a.ephemeralKey != "" && key == a.ephemeralKey {
    a.logger.Warn("ephemeral API key used for authentication", ...)
```

The key is regenerated per boot and printed to stdout pre-bootstrap (`boot/app.go:51-68`), but **nothing disables it after `bootstrapped` flips** — `ephemeralKey` is never cleared (grep: no mutation site). Anyone who ever captured it (container logs, CI logs, tmux scrollback) keeps permanent admin. `key == a.ephemeralKey` is also a timing-comparable secret.

**Fix (S):**
1. Thread a `bootstrapped func() bool` into `NewAPIKeyAuthenticator`; reject when true (the TODO already specifies this).
2. `subtle.ConstantTimeCompare` (matches the pattern you already use in `hmac.Equal`).

## S-6 · HIGH — Password reset does not invalidate existing sessions

**Where:** `core/system/auth/service.go:177-191` vs `core/system/users/model/system_user_utils.go:148-170`

`PasswordConfirm` — the unauthenticated path used *when credentials are compromised* — writes only the password hash. It never calls `IncrementTokenVersion`, so the middleware's `token_version` check (`middleware.go:82-88`) keeps accepting every pre-reset session for up to 8h, including the attacker's. The authenticated `UpdateProfile` path *does* bump the version — the inconsistency is the bug.

**Fix (S):** call `IncrementTokenVersion` after the password write in `PasswordConfirm` and in `SetBootstrapPassword`.

## S-7 · HIGH — `clientIP` trusts `X-Forwarded-For` unconditionally

**Where:** `core/interface/http/transport_fasthttp.go:489-497`

```go
func clientIP(ctx *fasthttp.RequestCtx) string {
    if fwd := ctx.Request.Header.Peek("X-Forwarded-For"); len(fwd) > 0 {
        return string(fwd)
    }
    ...
}
```

`req.ClientIP` keys the login rate limiter (`middleware.go:138`), identity-IP rate rules (`rate-limit.go`), audit `SourceIP`, and access logs — with no trusted-proxy configuration anywhere. Direct-to-server attackers rotate the header per request → fresh rate-limit bucket each time (see S-3: the limiter that would still apply never fires anyway), can poison a victim IP's bucket (login DoS), and can forge audit source IPs.

**Fix (M):** default to `RemoteAddr()`; add a config knob `trustedProxyHops` and only then take the right-most untrusted XFF element. Document it in the deploy playbook.

## S-8 · HIGH — fasthttp server: zero timeouts, 10 GiB buffered bodies, unbounded concurrency

**Where:** `core/interface/http/transport_fasthttp.go:129-143`

```go
const maxRequestBodySize = 10 << 30   // 10 GiB

t.server = &fasthttp.Server{
    Handler:           t.serveHTTP,
    MaxRequestBodySize: maxRequestBodySize,
}
```

`ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxConnsPerIP`, `Concurrency`, `StreamRequestBody` — all unset. With streaming disabled, fasthttp buffers the **whole body in RAM before your handler runs**, so the per-op blob limits (16 MiB / 256 MiB) apply only *after* an anonymous client has already shoved up to 10 GiB into memory. Slowloris and single-request OOM are both trivial.

**Fix (M):**
1. Set sane global timeouts (`ReadTimeout: 30s`, `WriteTimeout: 60s`, `IdleTimeout: 120s`) and `MaxConnsPerIP`.
2. Route-scoped body limits: small default (e.g. 4–8 MiB), large only on blob-chunk routes, and enable `StreamRequestBody` for those.

## S-9 · HIGH — Self-update verifies integrity only, against a checksum it computed itself

**Where:** `core/system/updates/service.go:679-691` (backfill), `:545-563` (vacuous pass), `:569+` (`swapExecutable`)

The GitHub provider hands over an **empty checksum**, so `stageLatest` hashes the freshly downloaded file and records that hash as the "expected" value:

```go
// service.go:679-686
if staged.Checksum == "" {
    digest, err := hashFile(s.stagedBinaryPath())
    ...
    staged.Checksum = digest
```

`verifyStagedBinary` then compares the file against the hash of the same file — corruption is caught, but **a MITM-substituted or malicious binary passes by construction**. Legacy rows with no checksum "pass vacuously" (`if pending == nil || pending.Checksum == "" { return nil }` — the comment literally says "trust the staged binary"). This code path replaces the running executable and exits the process. There is no release signature anywhere in the flow.

**Fix (M):** require a provider-supplied checksum **plus** a cryptographic release signature (minisign/cosign) with the public key pinned out-of-band; refuse to apply when either is missing; delete the vacuous-pass branch. (Depends on `asaidimu/updater` exposing the artifacts — flagged as needing verification of that library's capabilities.)

## S-10 · HIGH — Notification stream: double-close panic or goroutine + subscription leak (both paths broken)

**Where:** `core/system/notifications/service.go:246-258` (variant in `core/system/audit/service.go:50-85`)

```go
go func() {
    select {
    case <-msg.InputChannel():
    case <-ctx.Done():
        close(docCh)                     // close #1
        s.model.Unsubscribe(ctx, subID)
    }
    // Phase 2: stream is live. Only close when the client goes away.
    <-ctx.Done()
    close(docCh)                         // close #2 — if Done fired in phase 1: PANIC
    s.model.Unsubscribe(ctx, subID)
}()
```

There is no `return` in the phase-1 branch. If `ctx.Done()` fires during phase 1 (client disconnects mid-request), execution falls through, the second `<-ctx.Done()` returns immediately, and `close(docCh)` runs twice → `panic: close of closed channel` in an unrecovered goroutine → **process crash**. If instead `ctx` is a context that never fires (the Wails/desktop path, or fasthttp's `RequestCtx` depending on version — see S-19), the goroutine parks on phase 2 forever: **one leaked goroutine + one live persistence subscription per stream request**. The recent "fix stream cleanup logic" commit (907a823) did not fix either direction; `stream_repro_test.go` documents the shape of the problem.

**Fix (S):** `return` after the phase-1 cleanup, and guard close/unsubscribe with a `sync.Once`. Mirror the fix in `audit/service.go`.

## S-11 · MEDIUM — Disabled users keep working API keys (cache query matches disabled users)

**Where:** `core/system/module.go:383-388`, `core/system/auth/adapters.go:71-75`, `core/system/apikeys/model/system_api_key_utils.go:155-201`

The session path checks `GetActiveByID` which correctly filters `disabled Eq(-1)` (`system_user_utils.go:84-88`). But the API-key path resolves owner claims through the `LiveUsers` cache, whose query is:

```go
// module.go:386-389
return query.NewQueryBuilder().
    Where(data.DocumentIDField).Eq(key).
    Where("disabled").Neq(0).      // matches active (-1) AND disabled (1)
    Build()
```

`Neq(0)` keeps disabled users in the cache, so `loadUserClaims` still returns their permissions and their keys keep authenticating. `ValidateKey` checks only the key's own status/expiry — never the owner's state.

**Fix (S):** `Eq(-1)` in the cache query; also assert owner-not-disabled in `ValidateKey` (defense in depth — keys outlive cache staleness windows).

## S-12 · MEDIUM — Schedules Get/Update/Delete have no ownership check (IDOR), and Update can re-arm another user's schedule

**Where:** `core/system/schedules/service.go:155-234` (Update), `:245-257` (Delete)

`List` scopes by `user_id`; Get/Update/Delete look up purely by `_id_`. Any authenticated user who obtains a schedule ID (IDs are returned by `List`) can read another user's schedule — payloads may contain secrets — delete it, or repoint its `message`/`input` (feeding S-1).

**Fix (S):** compare `existing.user_id` (and tenant) against caller claims before mutation in Get/Update/Delete; keep an administrator bypass via the policy layer, not the handler.

## S-13 · MEDIUM — Reset tokens are replayable and never consumed

**Where:** `core/system/auth/credential_provider.go:87-96` (issue), `:98-136` (validate)

Reset tokens are stateless HMACs — the validation checks format, MAC, expiry; nothing marks them consumed. One leak (mail relay log, browser history, referrer) = unlimited replays for 5 minutes to set the victim's password, and the token remains valid even *after* being used. Combined with S-2 (forgeable when the default secret is used) this is the second half of account takeover.

**Fix (S):** persist consumed token IDs (the `_token_blocklist_` collection is sitting right there), or embed a per-user nonce and check it. Rotate the reset secret independently (HKDF, see S-2).

## S-14 · MEDIUM — Input validation fails open when the validator can't be built

**Where:** `core/runtime/dispatch/input.go:139-148`

```go
func ValidateInputDocument(s *definition.Schema, doc data.Documenter) ([]common.Issue, bool) {
    if s == nil { return nil, true }
    v, err := getDocValidator(s)
    if err != nil {
        return nil, true        // swallowed → operation permanently unvalidated
    }
    return v.ValidateLoose(doc.ToMap())
}
```

A malformed registration schema or validator construction error silently disables schema validation for that operation forever — no log, no 500. A silent, permanent validation bypass.

**Fix (S):** return the error (fail closed → 500 per dispatch contract) or at minimum `logger.Error` with the operation name. Prefer fail-closed: a broken schema is a boot-time bug, not a runtime inconvenience.

## S-15 · MEDIUM — Audit trail is fail-open and dropped at shutdown

**Where:** `core/runtime/audit_buffer.go:99-125` (circuit breaker), `:185-197` (`drain` discards with `_ =`), `core/system/module.go:500-523` (Stop never flushes), `core/runtime/access-log-dispatcher.go:121-136` (lazy init race)

Once the 4096-entry buffer fills or inserts fail, the breaker opens and **all subsequent audit events are dropped** with only a stderr line until one insert happens to succeed:

```go
// audit_buffer.go:109-118
select {
case b.entries <- entry:
    return nil
default:
    b.wg.Done()
    b.mu.Lock(); b.failed = true; b.mu.Unlock()
    b.logger.Error("audit buffer full, entering fail-open mode", ...)
```

Two more lifecycle defects ride along: (a) `Buffer()` lazily initializes without a lock and is called from concurrent completion goroutines (`access-log-dispatcher.go:207`) — two first-requests can each spawn a buffer/flush-loop, orphaning one; (b) `SystemModule.Stop` closes Scheduler/RateStore/BlobSvc/WorkflowRuntime but **never calls `AuditDispatcher.Sync()`/`Close()`** — and because the chain is built via `Wrap()` clones, nobody in production holds the instance that would flush. Up to 4096 compliance entries vanish on every shutdown. An attacker who can induce DB latency also suppresses exactly the records of their own activity.

**Fix (M):** construct the buffer eagerly in `ProviderSet` and pass it in; blocking-with-timeout `Write` instead of fail-open (or a disk spill file for drops); call `Audit.Sync()` in `Stop`; alert on breaker state.

## S-16 · MEDIUM — Internal error causes leak to HTTP clients

**Where:** `core/interface/http/transport_fasthttp.go:366-387`

```go
} else {
    sysErr = common.NewSystemError("INTERNAL_ERROR", err.Error())
}
issue := sysErr.ToIssue()
...
json.NewEncoder(ctx).Encode(map[string]any{
    "error": responseErrorBody{ ..., Details: issue.Cause },
```

Non-system errors put `err.Error()` straight into the response message, and `WithCause`-wrapped system errors propagate the raw cause into `details` — SQL errors, filesystem paths (`open staged update /opt/...`), internal strings, all reachable by anonymous clients on public endpoints.

**Fix (S):** log the cause server-side; return an opaque message; gate `details` behind an admin/debug flag.

## S-17 · MEDIUM — Unbounded pagination and full result drains on authenticated endpoints

**Where:** `core/system/collections/query.go:147-158`, `core/interface/http/register.go:266-276`

A client-supplied `pagination.limit` passes through untouched — `limit: 1000000000` materializes the whole collection; `drainChannelResponse` then accumulates every document in a slice before serialization. Attacker-controlled memory growth. (The QDSL→SQL compilation itself lives in `go-anansi` and could not be audited here — add a targeted test asserting user-derived filter values are parameterized.)

**Fix (S):** clamp `Limit` to a server max; page large results instead of draining.

## S-18 · MEDIUM — Update goroutines reuse the fasthttp `RequestCtx` after the handler returns

**Where:** `core/system/updates/service.go:293-300` (`Check` auto-apply), `:392-398` (`Apply`)

```go
go func() {
    time.Sleep(300 * time.Millisecond)
    s.applySwap(ctx)      // ctx is the request context
}()
return document.New(&ApplyView{Message: "update applied; restarting"}), nil
```

fasthttp recycles `RequestCtx` objects for the next request on the connection once the handler returns; `applySwap` → `store.PendingUpdate(ctx)` reads values from that recycled context 300 ms later. Data race / cross-request value bleed, on the path that ends in `os.Exit`. The scheduler-driven call site is fine; the HTTP ones are not.

**Fix (S):** detach before spawning — `context.WithoutCancel(context.Background())` (or `context.Background()` plus explicitly copied values).

## S-19 · MEDIUM — SSE producer teardown depends entirely on `ctx.Done()` semantics

**Where:** `core/interface/http/transport_fasthttp.go:310-321`, `core/interface/http/register.go:218-244`

The SSE writer returns on flush error but never closes/drains `stream`; the producer (`streamChannel`) exits only on `ctx.Done()`. In the HTTP path `ctx` is the fasthttp `RequestCtx` — whether `Done()` fires on client disconnect is version-dependent fasthttp behavior that **must be verified against the pinned v1.73** (flagged: no toolchain here to test). If it fires: the writer's early return races the producer's close (double-send into a closed writer path, plus S-10's sibling in notifications triggers a panic). If it never fires: one stuck goroutine per aborted SSE request plus whatever upstream keeps producing. The `#mem-20260821-003` note claims this was fixed — the fix leans entirely on the uncertain behavior.

**Fix (M):** give the writer side ownership of a `done` channel the producer selects on; add an SSE idle timeout. This is the audit's #1 "write a runtime test" item.

## S-20 · LOW — API-key allowlist is skipped on type confusion; missing property = allow-all

**Where:** `core/runtime/secure-dispatcher.go:31-45`, `:81-90`

If the `operations` property exists but isn't `[]any`/`[]string`, both assertions fail and the gate is **skipped** (treated as allow-all). "Nil = allow all" is also the designed default. One producer change (e.g. JSON round-trip quirks) silently turns scoped keys into unrestricted ones.

**Fix (S):** enforce whenever the property exists in any form; deny on unparseable type. Longer term, make `Operations` a typed field on claims (it already is — `Claims.Operations`) and drop the untyped property hop.

## S-21 · LOW — Grab-bag (each S-effort)

- **`system:core:docs:list` is `public`** (`operations/policies.go:15`) — anonymous attackers get the full API surface incl. schemas. Require auth, or split a minimal public view.
- **`system:core:reset` is a state-changing GET** (`registrations.go:130-140`, `Intent: Read` → GET per `derive.go:23-24`). Re-declare as `delete` intent.
- **SMTP `TLSOpportunistic`** (`mailer.go:63`) — credentials fall back to plaintext on downgrade. Default `TLSRequired`, opt out explicitly.
- **Rate limiting fails open on store errors** (`rate-limit.go:96-100`, `middleware.go:140` `if err == nil && !allowed`). Count consecutive store errors; alert.
- **CEL rules that fail to compile are silently skipped on reload** (`policies/service.go`, `Reload` loop `continue`) — stale authorization persists with no signal. Collect and report failures; keep the last good set.

# Part 2 — Architecture: What Will Make Maintenance a Nightmare

## A-1 · HIGH — `SystemModule` + `ProviderSet` is the god object of the whole system

**Where:** `core/system/module.go:73-133` (Setup), `:138-228` (seedProviders), `core/system/provider.go:46+`

```go
// provider.go:46-61 — ~30 public fields spanning 12 feature domains
type ProviderSet struct {
    Persist      base.Persistence
    Config       *runtime.Config
    Logger       *zap.Logger
    Bootstrapped bool
    Users         *model.SystemUsers
    APIKeys       *apikeysmodel.SystemAPIKeys
    Policies      *policies.PolicyModel
    ...
}
```

`Setup` is a ~60-line imperative boot script touching every feature domain (users, policies, blobs, schedules, workflows, notifier, updater). Every feature change ripples through `module.go`, `provider.go`, `seeds.go`, `gen_features.go`, and generated `services.go` — three parallel hand-maintained lists of the same features. The DI container exists but is used as a singleton registry, not for inversion of control.

**Fix (L):** split into per-feature sub-modules (the `abstract.Module` + topo-sort infrastructure already supports it); `SystemModule` becomes a thin aggregator; `ProviderSet` per feature.

## A-2 · HIGH — A 13-layer request path whose security-critical order is encoded in exactly one function

**Where:** `core/system/module.go:465-483`, `core/runtime/chain.go:65-75`, `core/hestia.go:86-89`, `core/interface/http/server.go:166-193`

One HTTP message crosses: fasthttp handler → access-log middleware → user middleware → auth middleware → registration handler → **7 dispatcher links** (`bootstrap → secure → ratelimit → throttle → tenant → blob → audit`) → `LocalDispatcher` terminal goroutine. That order is security-relevant (audit sees post-tenant context; `secure` before `ratelimit` means rejected requests don't consume tokens) and lives nowhere but `DispatcherChain()`:

```go
// module.go:465-478 — the single source of truth, silently mutable
chain := runtime.NewDispatcherChain(
    runtime.LinkEntry{Name: "bootstrap", Link: ...},
    runtime.LinkEntry{Name: "secure",   Link: ...},
    runtime.LinkEntry{Name: "ratelimit", Link: ...},
    ...
)
if m.opts.DispatcherChainFunc != nil {
    m.opts.DispatcherChainFunc(chain)      // embedders may reorder security links
}
```

`InsertBefore/InsertAfter/Remove` (`chain.go:34-63`) let embedders move `secure` behind `ratelimit` with no validation, and your own `todo/async_dispatcher_refactor.md:43` documents a stale ordering that no longer matches the code — proof of the single-source problem.

**Fix (M):** declare the canonical order as data (`DefaultChainOrder`), validate on `Build` (reject unknown/missing security links), document the stack in a `doc.go`, and restrict `DispatcherChainFunc` to insertion of *new* links at declared positions.

## A-3 · MEDIUM — Layering inversions in three places

- `core/internal/boot` (lowest layer) imports `core/system` and `core/system/logs` (`boot/app.go:19-20`) — the logging ring buffer is boot-seeded infra living in feature-land, and `SystemModule.Setup` resolves it via `abstract.MustResolve[*logs.RingBuffer]` (`module.go:75`).
- A domain service imports the HTTP interface: `operations/service.go:160-167` computes HTTP routes via `httpapi.IntentToHTTPMethod/DeriveRoute/IntentToHTTPPath` — the same triplet `utils/wails/dispatch.go:278-284` and `cmd/gen-routes/main.go:79-81` also call. Route math is transport-neutral; it's welded to `core/interface/http`.
- `utils/wails` imports `httpapi` internals (`SystemErrorToStatus`, `ExtractPathParams`, `BuildInputDocument`) — the "desktop adapter" is coupled to the HTTP package's guts.

**Fix (S each):** move `RingBuffer/Sink` to `core/runtime/logsink`; move intent→method/path derivation to a transport-neutral package (`core/abstract` or `core/runtime/route`); all four consumers follow.

## A-4 · MEDIUM — Three transports re-implement the same envelope/marshal/error contract

**Where:** `core/interface/http/transport_fasthttp.go` (`writeSuccess`/`writeError`), `utils/wails/dispatch.go:430-470` (`buildResponse`) **and** `:490-542` (`writeResult` — the same switch twice in one file), plus the CLI orchestrator. `wails/dispatch.go` is 564 lines: its own route table, its own `net/http` server, its own session/claims resolution, its own error→status mapping (`:544-564`).

Every envelope change (e.g. fixing S-16's leak) must now be made in 3+ places. **Fix (L):** extract `dispatch.MarshalResult(result, intent)` + `dispatch.StatusForError(err)` into `core/runtime/dispatch`; make fasthttp, wails, and CLI thin framings over them.

## A-5 · MEDIUM — Config sprawl: five precedence strata and three manifests

**Where:** `core/runtime/config.go:219-247` (`LoadConfig` + godotenv `Overload`), `:252-341` (`applyCommonEnvOverrides`), `core/runtime/env.go` (`ApplyEnvOverrides` re-applies the same vars a second time), `core/hestia.go:184-256` (70-line hand-written `applyTo`), manifests `hestia.json` + `anansi.json` + `schemas.lock.json`.

Effective precedence is defaults → `LoadConfig` env pass → `SetupConfig.applyTo` → `ApplyEnvOverrides` → `.env.dev` > `.env` > process env. Your own header note (`env.go:1-19`, `#org-20260821-004`, P1: "delete env.go entirely") is still open. `applyTo` must be hand-extended for every new field — merge-conflict magnet.

**Fix (M):** one canonical `LoadConfig(project, setup)` with a documented source order; generate the merge from the struct tags (codegen infra exists); consolidate manifests.

## A-6 · MEDIUM — Three error conventions; unknown codes silently map to 500

**Where:** `core/runtime/errors.go:11-39` (sentinels) vs ~175 inline `common.NewSystemError(` calls (workflows alone: 38, policies/model: 23, collections/handler: 13) vs 50+ raw `fmt.Errorf`. `transport_fasthttp.go`'s `systemErrorToStatus` only knows the sentinel codes, so feature-scoped codes like `WORKFLOW_INVALID_NODES` and `SCHEDULE_INVALID_CRON` fall through to 500.

**Fix (M):** lint/codegen check forbidding inline codes that duplicate sentinels; feature-scoped sentinel files; extend the status map with a fallback convention (e.g. `*_INVALID_*` → 400).

## A-7 · LOW — Global mutable state and `init()`-time panics

`authRateLimiter` package global (`middleware.go:20`), `abstract.SystemScopePrefix` mutable global (`system_options.go:5`), `boot.ProjectName` set imperatively (`hestia.go:259-261`), `celEnv` built in `init()` with `panic` (`policies/docprocessor.go:15-27`), `updates.exitProcess = os.Exit` test seam (`updates/service.go:28`). Each is small; together they make parallel tests and library embedding annoying. **Fix (S):** move all into `Options`/constructors; `sync.OnceValue` for the CEL env.

## A-8 · MEDIUM — Byte-identical duplicate input types (and 24 more per your own review)

**Verified:** `core/system/blobs/inputs.go` ≡ `core/system/blobs/model/inputs.go` (diff: package line + trailing newline only). `todo/blobs_refactor.md` has the execution plan; the second-pass review catalogued the pattern as `#cruft-20260821-001..-024` across 8 packages. One edit to the wrong copy compiles fine and silently diverges the wire schema.

**Fix (M, mechanical):** execute `todo/blobs_refactor.md`, then sweep the rest. Best enforced with a CI check: `go run ./cmd/hestia service generate && git diff --exit-code`.

## A-9 · LOW — Committed codegen with no freshness check in CI

**Where:** `.github/workflows/test.yaml` runs only `make test` + `go build`. Generated artifacts (`registrations.go`, `policies.go`, `client/core/routes.gen.ts`) have no drift check — a new `@hestia.register` without two manual commands ships a stale TS route table and stale policy bindings. Also `cmd/gen-routes` boots a full server (sqlite + migrations + seeds) just to learn the route table.

**Fix (S):** CI step: regenerate + `git diff --exit-code`; derive routes from `Registrations()` without a full boot.

## A-10 · LOW — Boot warns instead of failing on registration errors

**Where:** `core/internal/boot/app.go:182-195` — duplicate handler names and message-name grammar violations are Warn-and-continue, while schema validation 20 lines later fails boot. Since the entire security model keys off operation names (policies/bindings/audit), a silently dropped registration means a feature is absent *or* its policy binding dangles.

**Fix (S):** collect and return as a boot error.

## A-11 · LOW — Migrations silently skip version mismatches; UUID-string ordering

**Where:** `core/internal/migrations/registry.go:88-95`

```go
if sc.Version.String() != m.From {
    continue          // no error, no log — schema never converges
}
```

A hand-edited DB or partial failure boots "successfully" with a schema no migration will ever fix. Ordering relies on UUIDv7 string sort (`:44-50`) and per-collection map iteration (`:71`) is nondeterministic (benign today, fragile later).

**Fix (S):** log/collect skipped migrations; fail (or loud warn + `--force`); assert chain continuity per collection at startup.

## A-12 · MEDIUM — Two god files

- `core/system/workflows/service.go` (882 lines): definition CRUD + compile/register lifecycle + ad-hoc runs + run introspection + SSE streaming + a node-registry API that **ships raw eval-able JS strings to the client** (`NodeHandlesJS`, `:660-689`) + its own JSON helpers duplicated from `policies` and `operations`.
- `core/system/updates/service.go` (763 lines): `Check`/`Stage`/`RunScheduledCheck` each re-implement the check→stage→notify→record→auto-apply→sleep(300ms)→swap dance with slightly different orderings — a fix in one gets missed in the others (this exact class of bug is S-18). Two constructors exist because codegen migration is unfinished. ~40% of the file is inline devnote essays.

**Fix (M):** split workflows into `definitions/runtime/stream/registry` files; extract `runUpdatePipeline(ctx, opts)` used by all three entry points; finish the codegen migration and delete `NewServiceFromDeps`.

## A-13 · LOW — Misleading provenance comments and self-contradictory devnotes

`operations/handler.go:1-17` claims boot uses those handlers — grep shows **zero non-test references**; the file is dead, and `NewDocumentationHandler` duplicates `DocsList` from `service.go`. Multiple devnotes are marked `issue resolved status=open` (contradiction) — e.g. `middleware.go:29` and `register.go:201` claim rate-limit/stream fixes that are demonstrably not fully working (S-3, S-19). `tenants/` is a half-feature: model initialized, collection created and migrated, no service, no handlers. Misleading comments are worse than none.

**Fix (S):** delete `operations/handler.go`; park or finish `tenants` with a one-line decision note; adopt a devnote convention where "resolved" removes the status field.

## A-14 · MEDIUM — The `Dispatcher.Send` contract exists only as a comment

**Where:** `core/abstract/dispatcher.go:74-88` — "non-nil return = synchronous rejection, no goroutine; nil = accepted, `onComplete` fires exactly once, even on panic." Every chain link must honor this by hand; nothing enforces it; `go vet` can't see it. `AuditDispatcher.Send` currently gets it right (`access-log-dispatcher.go:152-158` wraps `onComplete` only after `next.Send` succeeds), but nothing prevents the next link from double-firing or leaking goroutines. Your planned durable-lane epic (`todo/async_dispatcher_refactor.md` Epic 2) adds more contract-carrying links.

**Fix (M):** template-method `runtime/dispatch.Link` helper (`PreCheck` + `OnComplete`) that encodes the invariant once; a contract test all links must pass (`async_contract_test.go` is a start).

## A-15 · LOW — `os.Exit` inside the service layer

`auth/service.go:276` (`SetBootstrapPassword` → `os.Exit(0)` mid-request) and the update swap path. It works, but it makes the library un-embeddable for hosts that manage their own lifecycle, and it means the request never returns. **Fix (S):** return a sentinel (`ErrRestartRequired`) and let `cmd/hestia` interpret it.

# Part 3 — Performance Notes

Most performance risk here is really the security findings wearing a different hat; the structural ones worth tracking separately:

| Area | Finding | Notes | Effort |
|---|---|---|---|
| Request memory | S-8: 10 GiB buffered bodies, no `StreamRequestBody` | single-request OOM by anonymous clients | M |
| Query memory | S-17: uncapped `pagination.limit` + `drainChannelResponse` | full-collection materialization | S |
| Audit throughput | S-15: 4096-entry buffer, fail-open breaker, per-entry `wg.Add/Lock` | contention + silent drop under load | M |
| AuthZ cache | S-11: `LiveUsers` cache retains disabled users | cache correctness, minor memory | S |
| Validator caches | `dispatch/input.go` uses bounded `ManagedCache` (1024 entries, positive-TTL=0) — fine; but `schedules/validate.go:24-27` and `structmap.go` (per `todo/cache-audit.md`) keep unbounded maps keyed by pointers | bounded migration path exists, reuse it | S |
| Scheduler / ratestore | checked: tickers stop cleanly, store has idle eviction loop and consistent lock order (`ratestore/inmemory.go:160-181`, deadlock concern already fixed) | healthy | — |

# Part 4 — Dependency Risks

No CVE database was consulted (no network in the audit sandbox); these are reasoned from the pinned versions and how they're used:

| Dependency | Version | Risk | Action |
|---|---|---|---|
| Go toolchain | `go 1.27rc1` (`go.mod:3`, CI pins `1.27.0-rc.1`) | a **release candidate** as the production/CI toolchain — no normal patch cadence, security fixes land in stable minors | pin a stable release the day it's out; until then accept explicitly |
| `natefinch/lumberjack` | `v2.0.0+incompatible` | the 2019 pre-module line; misses years of rotation/cleanup fixes | bump to `v2.2.x` |
| `wneessen/go-mail` | `v0.8.1` + `TLSOpportunistic` (S-21) | older minor + downgrade-to-plaintext posture | bump; `TLSRequired` default |
| `wailsapp/wails/v2` | `v2.12.0` | desktop builds inherit WebView2/WebKit CVE surface | keep current; scope to `cmd/hestia-desktop` only |
| `mattn/go-sqlite3` | `v1.14.48` | cgo SQLite — generally fine; injection safety depends on go-anansi's compiler (S-17 note) | keep current; add parameterization test |
| `valyala/fasthttp` | `v1.73.0` | current line, but all DoS knobs unset (S-8) and `RequestCtx`-as-context semantics underpin S-18/S-19 | configure; verify `Done()` |
| Client `package.json` | `uuid ^14`, `typescript ^7` | ahead of widely-known stable majors — verify these are real, intentional pins | audit + lockfile review |
| Supply chain | several first-party `asaidimu/*` libs (anansi, go-iam, hermes, blobs, updater) | bugs in these are effectively in-repo bugs (e.g. un-auditable SQL compiler, updater internals behind S-9) | add `govulncheck` to CI; consider pinning + reviewing upgrades |

# Part 5 — Docs & CI

- **CI** (`.github/workflows/test.yaml`): test + build only. No `go vet`, no linter, no `-race` (you *have* a `race_test.go` — CI never runs it), no `govulncheck`, no codegen drift check (A-9), no client tests (`make test-client` exists in the Makefile and isn't wired). Cheap wins, all S-effort.
- **Makefile**: fine; `test-server` target resolves (`cmd/test-server` exists).
- **Docs**: VitePress source in `docs/` plus fully generated HTML committed under `cmd/docs-server/static/` — another generated-artifact-without-drift-check class (A-9 pattern).
- **Embedded devnotes**: the `@note` discipline is unusual and *good* in intent, but `updates/service.go` carries 30+ line essays inline (A-12), several are marked `resolved status=open` (A-13), and at least two document fixes that don't fully work (S-3, S-19). Move resolved notes to `docs/` or a CHANGELOG-style ledger; keep code comments for invariants only.
- **Stale todo docs**: `todo/first_run_api_key.md`'s title ("prints no API key") no longer matches the code (printing was added, `boot/app.go:51-68`) while its *security* section (post-bootstrap key validity) remains accurate and unaddressed (S-5). Prune as you fix.

# Part 6 — What's Genuinely Good (so you don't over-correct)

- Password hashing: bcrypt cost 12, correct mismatch-vs-error handling; tokens/keys use `crypto/rand`; `hmac.Equal` used for MAC comparison.
- Cookie defaults: `Secure`, `HTTPOnly`, `SameSite=Strict` (`config.go:159-165`); TS client stores no tokens (no `localStorage` usage found).
- CEL policy evaluation fails **closed** on eval error (`rulecompiler.go`); policy/rule management endpoints are administrator-gated.
- Static file serving goes through `io/fs` — path traversal is rejected; SPA fallback can't escape the FS root.
- Rate store: bounded with idle eviction; the documented shard→bucket deadlock was actually fixed.
- `SessionService.Validate` enforces `ExpiresAt` at the token level (the Wails gap from `#review2-20260821-001` is genuinely fixed).
- Codegen with golden-file tests; a self-review culture (`docs/reviews/`, `todo/`) that most repos don't have — this audit confirmed several of your own open items (below) rather than contradicting them.

# Part 7 — Cross-Reference: Your Own Known Issues vs. Reality (verified today)

| Acknowledged in | Claim | Status at `3368956` |
|---|---|---|
| `todo/blobs_refactor.md` | `blobs/inputs.go` ≡ `model/inputs.go` | **Confirmed, still present** |
| `todo/cache-audit.md` | unbounded caches (`structmap.go`, schedules validators) | **Still present** (input.go's is now bounded) |
| `todo/first_run_api_key.md` | ephemeral key valid post-bootstrap | **Confirmed** (S-5); the "not printed" part of the doc is stale |
| `env.go:1-19` (P1) | merge env.go into config.go | **Still open** |
| `todo/async_dispatcher_refactor.md` Epic 2 | durable lane; FireAndForget is best-effort | **Still open** (latent `RequestCtx` reuse also present, S-18) |
| `todo/blobs_refactor.md`, `HandlerGenerics.md`, `general_todo.md` | planned refactors | Open per file headers |
| Second-pass review §SRP | inputs/outputs duplication pattern | **Confirmed** (A-8) |
| Second-pass review `#review2-20260821-003` | `InsertBefore/InsertAfter` naming bug | **Actually fixed** (`chain.go:37,48`) |
| `#mem-20260821-003` | stream goroutine leak fixed | **Partially** — depends on unverified `ctx.Done()` (S-19); notifications still broken (S-10) |
| `#sec-20260821-003` | auth rate limiting added | **Added but dead on the HTTP path** (S-3) |

# Part 8 — Prioritized Fix Order

**Batch 1 — "stop the bleeding", one focused day (all S):**

| # | Item | Why first |
|---|---|---|
| 1 | S-2 default secret | every deployment today is forgeable |
| 2 | S-3 login limiter + elevate/confirm gates | brute force currently free |
| 3 | S-6 reset bumps token_version | closes the post-takeover session hole |
| 4 | S-10 stream `return` + `sync.Once` | one disconnect away from a process crash |
| 5 | S-5 bootstrap-key guard | kills the permanent-admin log artifact |
| 6 | S-11 `Eq(-1)` | two-line fix, real authz gap |
| 7 | S-12 schedule ownership | prerequisite hygiene for S-1 |
| 8 | S-14 validation fail-closed | one-line policy change |
| 9 | S-18 detached contexts in updates | removes a race on the `os.Exit` path |
| 10 | A-9 CI drift check + A-10 boot fail-fast + A-11 migration skip warn | makes all future changes verifiable |

**Batch 2 — design-required fixes (week 1–2, M):**

- S-1 schedule authorization model (decide: creator-identity dispatch vs. target-policy evaluation at create time vs. allow-list — recommend doing 1+2 together)
- S-4 real logout via blocklist + session-ID rotation
- S-7 trusted-proxy IP handling
- S-8 fasthttp timeout/body-limit configuration
- S-9 update signing (requires `updater` provider work)
- S-15 audit buffer lifecycle (eager construction, Sync on Stop, spill-on-drop)
- S-19 SSE teardown + the fasthttp `Done()` runtime test
- S-13, S-16, S-17, S-20, S-21 fill-ins

**Batch 3 — structural (month 1+, mostly M, two L):**

- A-2 chain order as validated data; A-14 `Link` template method (do together)
- A-5 config consolidation (you already scoped it in `env.go`'s note)
- A-6 error-convention lint + sentinel backfill
- A-12 split workflows/updates god files; A-4 shared transport marshaling (L); A-1 per-feature modules (L)
- A-3, A-7, A-8, A-13, A-15 mechanical sweep

---

## Appendix — Method & Limitations

- **Verified evidence**: every `file:line` citation and snippet in this report was re-read from the working tree at `3368956` by the auditor (not taken on faith from tooling). Repo-wide claims ("nothing reads/writes", "no references outside tests") were grep-verified.
- **Not run**: `go build/vet/test` — the audit sandbox has no Go toolchain and the toolchain fetch is blocked; static analysis only. The two findings that hinge on runtime/library behavior are flagged inline: fasthttp `RequestCtx.Done()` semantics (S-19, affects S-10's exploitability on the HTTP path) and `asaidimu/updater` provider internals (S-9).
- **Not audited**: `go-anansi` query-compilation internals (SQL injection surface — recommend the parameterization test from S-17), `go-iam` rule evaluation depth, the Wails runtime itself. The TS client was reviewed at the store/transport level only.
- **Snippet fidelity**: excerpts are trimmed for brevity; line ranges are exact but may show elided middle lines (`...`). Re-read the cited range before quoting in an advisory.
