# Hestia auth: server session restore + `@asaidimu/iam` integration

**Status:** implemented (Go + client + example verified; IAM dependency pin deferred until its major release)
**Created:** 2026-11-06
**Updated:** 2026-11-06
**Goal:** make an end-to-end "open a new tab and stay logged in" flow work, and
replace the hand-rolled workarounds in `examples/auth-restore/authenticator.ts`
with real client/platform support.

## Background

`@asaidimu/iam` was refactored and committed in `~/projects/iam` at
`e52b6a1 feat(core)!: implement revisioned state architecture and
multi-tab coordination` (on top of `v8.2.0`). That commit is **breaking**
and is **not yet released/tagged**; `package.json` still says `1.0.0` and
semantic-release in IAM's CI is expected to cut the major version.

What the new IAM gives Hestia (see `~/projects/iam/docs/state.md` and
`~/projects/iam/docs/migration-9.md`):

- `createAuthenticator()` is now async (`await createAuthenticator(...)`).
- `onRestore?: () => Promise<Identity | null>` — startup hook that adopts a
  server-known/cookie-backed session. This is exactly what Hestia needs.
- `onLoad(identity)` is `Promise<void>` and runs once after startup.
- `provider.getSnapshot()` — pure, frozen, revisioned read.
- `provider.refreshResult()` — discriminated
  `refreshed | unauthenticated | unavailable | unchanged`; thrown errors keep
  the identity, `null` clears it.
- `refreshLeadTime` replaces `refreshInterval` (alias still works).
- Optional coordination subpaths:
  `@asaidimu/iam/coordination/web-locks`,
  `@asaidimu/iam/coordination/broadcast`,
  `@asaidimu/iam/identity/persistence`.

This unblocks the **IAM-side** requirements of notes
`#auth-20261029-004` and `#auth-20261029-011`. Everything else in the note
series is Hestia-side and still open.

## Blocker chain (in dependency order)

```text
#002 expires_at ─┐
#001 session:get ─┴─> #003 HestiaAuth.session() ──> #006 authenticated()
                                   │
                                   └─> #004 / #011 (IAM now ready) ──> restore flow
#008 heartbeat ownership ─────────────────────────────────────────────┤
#007 subscription + #009 cross-tab ──> #013                           │
                                            └─> #012
#005 logout divergence is independent.
```

## Tasks

### Phase 1 — Server: expose the current session

- [*] Add `system:auth:session:get` (closes `#auth-20261029-001`)
  - **Context:** The session cookie is HttpOnly, so a fresh tab cannot read it.
    `system:auth:session:validate` is `Internal: true` and wants a raw token in
    the request body, so a browser cannot use it. A new tab therefore has no way
    to learn that a valid session exists.
  - **Details:** Add a service method + registration with
    `Intent: abstract.Read`, `Enabled: true`, `Internal: false`,
    `rule="authenticated"` (do **not** add to `NoRefreshCommands`; it must slide
    the session like any other authenticated request). Handler:
    1. read claims from `ctx` (`runtime.GetUserID(ctx)`, `runtime.GetTokenID(ctx)`,
       `runtimecontext.ClaimsFromContext(ctx)` for `ExpiresAt`);
    2. load the user (`s.users.Get(...)` / `GetByID`);
    3. sanitize with `common.ContextWithSanitizationScope(ctx, "users")` and
       `user.MustDocument().Sanitize(sctx)` exactly like `CreateSession`;
    4. return the sanitized user map plus `expires_at` (unix seconds, from the
       claims, falling back to `now + sessionTTL`).
  - **Output type:** add `SessionOutput`/`SessionDocumentView` in
    `core/system/auth/model/outputs.go` with `user` and `expires_at`. Reuse the
    `LoginDocumentView` shape if practical.
  - **Files:** `core/system/auth/service.go`, `core/system/auth/registrations.go`,
    `core/system/auth/model/outputs.go`.
  - **Tests:** request with a valid cookie returns the user + `expires_at`;
    unauthenticated request returns the normal auth error (browser maps to 401);
    sanitization does not leak `password`/secrets.
  - **Notes:** `authMiddleware` already puts `user_id`, `token_id`,
    `expires_at` into identity properties
    (`core/interface/http/middleware.go:99`).

- [*] Return absolute session expiry from login (closes `#auth-20261029-002`)
  - **Context:** `LoginOutput`/`LoginDocumentView` only carries `user`; the
    absolute expiry lives only in the cookie, so IAM consumers hard-code 8h and
    drift from the server.
  - **Details:** `CreateSession` already discards the expiry returned by
    `credProv.CreateSession(user.ID, s.sessionTTL)`; keep it and emit
    `expires_at` (unix seconds) in the login document. Mirror it in
    `system:auth:session:get` so both responses agree.
  - **Files:** `core/system/auth/service.go`,
    `core/system/auth/model/outputs.go`, `client/packages/core/system/auth/types.ts`.
  - **Tests:** login response contains `expires_at`; value equals
    `now + sessionTTL` within tolerance; mock client/routes updated.

### Phase 2 — Client: add `HestiaAuth.session()`

- [=] Declare `@asaidimu/iam` in the client package
  - **Context:** `client/packages/core/package.json` does not list
    `@asaidimu/iam`, yet `examples/auth-restore/authenticator.ts` imports it.
    The example has no `package.json` of its own.
  - **Skipped:** the example is not part of any package build, and `@asaidimu/iam`
    at `e52b6a1` is an unreleased breaking refactor (`package.json` still says
    `1.0.0`). Adding a `file:` link to `client/packages/core/package.json` would
    couple the published client to an unreleased local checkout. The example was
    typechecked against the local IAM source instead (see Verification). Add the
    dependency when IAM cuts its major release; do **not** pin `8.2.0`, which
    lacks `onRestore`.

- [*] Implement `HestiaAuth.session()` (closes `#auth-20261029-003`)
  - **Context:** `resume()` exists but throws `Method not implemented`, so a
    fresh tab cannot adopt the cookie-backed session.
  - **Details:** implement `session(): Promise<LoginResult | null>` (either on
    top of `resume()` or replacing it):
    - dispatch `system:auth:session:get` with
      `notifyAuthStateChange: false` so "not logged in" does not run the global
      401 handler;
    - 200 → `setIdentity(user)`, return `{ user, expires_at }`;
    - 401 → clear local identity and return `null`;
    - anything else (network error, 5xx) → **throw**. Callers must distinguish
      "not logged in" from "could not check", otherwise an offline blip logs
      users out.
  - **Types:** add `expires_at?: number` to `LoginResult`
    (`client/packages/core/system/auth/types.ts`).
  - **Files:** `client/packages/core/system/auth/store.ts`,
    `client/packages/core/system/auth/types.ts`,
    `client/packages/mock/src/routes.ts` + `server/auth.ts` (mock route).
  - **Tests:** 200/401/throw cases; `notifyAuthStateChange: false` respected;
    identity is set/cleared correctly.

- [*] Make `authenticated()` server-confirmed (closes `#auth-20261029-006`)
  - **Context:** `client/packages/core/container.ts:164` returns `false`
    immediately when the local identity is `null`, even when the cookie is
    valid — the direct cause of "every tab asks me to log in".
  - **Details:** when local identity is `null` and not in api-key mode, call
    `auth.session()`; `null` → `false`, thrown error → propagate. Keep the
    api-key shortcut. The heartbeat is a poor probe because it carries no
    claims.
  - **Files:** `client/packages/core/container.ts`.
  - **Tests:** new tab with valid cookie → `true`; 401 → `false`; network
    failure → throws.

### Phase 3 — Client lifecycle

- [*] Real auth-state subscription (closes `#auth-20261029-007`)
  - **Details:** `onAuthStateChange(cb): () => void` backed by
    `store.watch("identity")`, so it fires on login, logout, restore, and
    refresh — not only on 401. Keep 401 → clear. Return an unsubscribe.
  - **Files:** `client/packages/core/container.ts`.

- [*] HestiaClient owns the heartbeat (closes `#auth-20261029-008`)
  - **Context:** the 5-minute heartbeat is what slides the server session (8h
    absolute / 30min idle / 15min refresh TTL). Today every consumer wires it
    by hand, so restored sessions, expiry logouts, and cross-tab logouts never
    start/stop it.
  - **Details:** start when `store.identity` becomes non-null, stop when it
    becomes null. Keep `startHeartbeat`/`stopHeartbeat` as an opt-out. A
    suspended machine exceeding the idle limit is expected behaviour.
  - **Files:** `client/packages/core/container.ts`.

- [*] Cross-tab auth signalling (closes `#auth-20261029-009`)
  - **Details:** broadcast only "auth changed", never claims; do not re-post from
    a received message (ping-pong). IAM now ships
    `createBroadcastTransport()` for snapshot sync if the IAM layer needs it,
    but `HestiaClient` should still expose a small auth-changed channel.
  - **Files:** `client/packages/core/container.ts`.

### Phase 4 — Adopt the new IAM in the reference example

- [*] Replace restore-as-authenticate with `onRestore`
      (closes `#auth-20261029-004`, `#auth-20261029-011`)
  - **Context:** `examples/auth-restore/authenticator.ts:91` models restore as a
    variant of `authenticate()` because IAM 8.2.0 had no startup hook.
  - **Details:** rewrite against the committed IAM:
    - `await createAuthenticator({ ... })`;
    - `onRestore: () => client.auth.session()` (returning the user/identity or
      `null`) instead of the `AuthRequest` union;
    - delete `AuthRequest` and the restore branch;
    - use `onLoad` for post-startup notifications;
    - map `expires_at` onto IAM's per-login `session.ttl`;
    - use `refreshResult()` semantics where it currently guesses between
      "rejected" and "unavailable".
  - **Files:** `examples/auth-restore/authenticator.ts`.
  - **Note:** update the `@note #auth-20261029-010` context block and the
    `@see` list once done.

- [*] Delete the example heartbeat bridge (closes `#auth-20261029-012`)
  - **Depends on:** heartbeat ownership (Phase 3).
  - **Files:** `examples/auth-restore/authenticator.ts`.

- [*] Move cross-tab + 401 bridge into `HestiaClient`
      (closes `#auth-20261029-013`)
  - **Depends on:** `#007` and `#009`.
  - **Files:** `examples/auth-restore/authenticator.ts`.

### Phase 5 — Independent cleanup

- [*] Fix logout divergence (closes `#auth-20261029-005`)
  - **Context:** `client/packages/core/system/auth/store.ts:81` — `logout()` can
    leave the client and IAM identity diverged, and the error-code check is
    dead. Independent of the IAM release; can be done any time.
  - **Files:** `client/packages/core/system/auth/store.ts`.

## Verification

- **Go:** `go build ./...`, `go test ./core/system/auth/...`.
  Restart the test server on `:8070` after Go changes
  (`lsof -ti :8070 | xargs kill`, rebuild `./cmd/test-server`).
- **Manual:** `POST /api/system/session` with
  `admin@test.local` / `password123`, then open a second browser context and
  confirm `authenticated()` is `true` with no re-login.
- **Client:** `bun run test` (and `test:browser` where relevant) in `client/`.
- **Example:** build/run `examples/auth-restore` against the local IAM build.

### Results (2026-11-06)

- Go build/test were run with a temporary modfile pointing at a clean
  `go-anansi v8.10.1` archive because `~/projects/go-anansi` is on a broken
  `refactor/encoding-collapse` branch:
  `go build -modfile=/tmp/opencode/hestia-verify.mod -tags sqlite_fts5 ./...`
  and `go test -modfile=... -tags sqlite_fts5 ./core/system/auth/...` — pass.
- Client typecheck: `bunx tsc --noEmit -p tsconfig.json` — only the pre-existing
  `system/blobs/compaction.test.ts` `unknown` errors remain.
- Client tests: `bunx vitest run packages/core/system/auth/auth.test.ts
  packages/core/tests/auth.e2e.test.ts` — 10/10 pass (server on `:8070`);
  `packages/core/system/auth/session.test.ts` — 3/3 pass (200 / 401 / network
  error paths for `HestiaAuth.session()`).
- Example: typechecked against `~/projects/iam` source at `e52b6a1` — pass.
  (`cmd/gen-routes` could not run for the same go-anansi reason, so
  `routes.gen.ts` was updated by hand to match its output.)

## Open decisions

- Which consumer version to pin: local `file:` link until IAM's major release
  lands, then the published version. **Deferred** (see Phase 2).
- Whether `system:auth:session:get` should also be surfaced in the mock package
  and in `client/packages/mock/src/seed.ts` rules (yes, for parity).
  **Done.**
- Whether `expires_at` is unix seconds (proposal) or milliseconds — pick one and
  use it in both login and session:get. **Unix seconds**, matching
  `SessionInfo.ExpiresAt` and `abstract.Claims.ExpiresAt`.
