# user:query ignored filter/pagination (returned all users)

**Intent:** `POST system:collections:user:query` with a QDSL body returned
every user unfiltered. Two compounding causes found 2026-09-10.

- [*] Free-form QDSL payload on UserQueryInput
  - **Context:** `UserQueryInput` only declared
    `{username, limit, cursor}` under payload, so the input pool stripped
    `filter`/`pagination` before the handler ran (the audit `LogQueryInput`
    already used a free-form map — users was the odd one out). Nothing
    constructed or read the typed fields; the handler ignores bound input.
  - **Files:** `core/system/users/model/data_transfer_objects.go`.
- [*] Normalize singular `filter` → canonical `filters` at parse boundary
  - **Context:** deeper bug found while verifying: the wire key is `filters`
    (plural — Go struct tag + TS `QueryDSL` type agree), but callers send
    `filter`; `query.FromBytes` silently drops unknown top-level keys, so
    the query ran unfiltered. Fixed with a `filter`-as-alias normalization
    (deployed TS clients send singular; no client change needed).
  - **Details:** new `parseCollectionQuery` helper used by all three QDSL
    parse sites (`runCollectionQuery`, `NewReadCollectionHandler`,
    view `parseViewQuery`).
  - **Files:** `core/system/collections/query.go`, `views.go`.
- [*] Regression tests + live verify
  - **Files:** `core/system/users/handler_test.go`
    (`TestUserQueryInputPreservesQDSL`),
    `core/system/collections/filter_test.go`
    (`TestNamedCollectionQueryAppliesFilter`, both spellings, exact row),
    `client/packages/core/system/users/store.test.ts` (both spellings E2E).
  - **Verified:** :8070 live — both spellings return exactly the match;
    Go users/collections suites + TS users suite green.
- [ ] Follow-ups (not done)
  - Upstream: `query.FromBytes` should reject unknown top-level keys
    instead of silently dropping them (go-anansi `core/query`).
  - The :8090 dev server still runs pre-fix code — restart it to pick this up.
- [*] Block system collections on generic document routes + project user:query
  - **Context:** follow-up report — `user:query` returned raw docs incl.
    bcrypt password hashes. Per direction, system collections are no longer
    addressable through the generic `document:*` messages at all.
  - **Details:** `SYSTEM_COLLECTION` guard in `NewCollectionQueryHandler`
    and `NewDocument{Get,Create,Update,Delete,UpdateMany}` handlers
    (internal `_x_:read` messages and direct persistence access untouched —
    the server itself still needs raw reads, e.g. auth validation).
    `system:collections:user:query` now serves via `NewUsersQueryHandler`,
    delegating row materialization to the generated users model
    (`ReadAs[*UserPublic]`, envelope from one raw read) — no hand-rolled
    binding anywhere; pagination defaults extracted into shared
    `ensureQueryPagination`. Follow-on find: the client `HestiaPolicies`
    store queried `_operation_policy_` through the blocked generic route,
    so added dedicated `system:collections:operation_policy:query`
    (mirrors `audit_log:query`) and pointed the client + mock table at it.
  - **Files:** `core/system/collections/{query,handler,service}.go`,
    `model/inputs.go`, `inputs.go`, regenerated registrations/policies/
    routes, `client/.../policies/store.ts`, mock routes, `filter_test.go`
    (rejection matrix + no-password assertion), TS users/policies tests.
  - **Verified:** live — `user:query` returns exact match with no `password`
    key; generic query/get on `_user_` → `SYSTEM_COLLECTION`; Go core tree
    + full client suite (293 passed) green.
