# Todo Index

Working documents for Hestia. Files are grouped by state:

- [`active/`](active/) — workstreams that are partially landed or currently in progress.
- [`pending/`](pending/) — planned or captured work that has not been started (or only has open follow-ups).
- [`done/`](done/) — completed work kept for reference and history.

## Current focus

No task is claimed right now. The last session completed the auth/session IAM
integration, the blobs `v1.3.5` upgrade, and the boot/DI review fixes — all in
[`done/`](done/). The streaming implementation guide
([`../docs/guide/streaming.md`](../docs/guide/streaming.md)) is written; the
next feature workstream is the persistence-events stream in
[`pending/persistence-events-stream.md`](pending/persistence-events-stream.md).

The nearest in-flight workstream is **HandlerGenerics §2** — see
[`active/HandlerGenerics.md`](active/HandlerGenerics.md). Section 1 (document
standardization) has shipped; section 2 (struct-first output and registration)
is next.

## Active

| File | Summary |
|---|---|
| [`active/HandlerGenerics.md`](active/HandlerGenerics.md) | Handler framework spec. §1 (document standardization) shipped; §2 (struct-first output + registration) is next. |
| [`active/async_dispatcher_refactor.md`](active/async_dispatcher_refactor.md) | Async-native dispatcher. Epic 1, 1.5, 1.6 landed; Epic 2 (durable lane via `go-events/v2`) remains. |
| [`active/streaming-codegen.md`](active/streaming-codegen.md) | Streaming + codegen coverage. B1–B10, B12, B13 landed; B11 deferred (needs a persistent-connection transport); B14 (client SDK streaming ergonomics) open. |

## Pending

| File | Summary |
|---|---|
| [`pending/blobs_refactor.md`](pending/blobs_refactor.md) | Deduplicate `core/system/blobs/inputs.go` vs `model/inputs.go`; pick `model/` as canonical, update consumers, delete the copy. |
| [`pending/cache-audit.md`](pending/cache-audit.md) | P1: replace unbounded `sync.Map` caches (`docValidators`, `fieldsCache`, `inputPoolCache`, `envelopeSchemas`) with the anansi cache. |
| [`pending/command_hooks.md`](pending/command_hooks.md) | Pre/post command hooks. Captured, open questions unresolved — do not implement until they are settled. |
| [`pending/command_router.md`](pending/command_router.md) | Cobra-like setup/command router so `--version`/`--help` short-circuit boot; no external dependency. |
| [`pending/dev-questions.md`](pending/dev-questions.md) | 101 due-diligence questions for adopting Hestia as a production platform (reference). |
| [`pending/first_run_api_key.md`](pending/first_run_api_key.md) | First-run key printing is implemented; remaining: stop printing after bootstrap and verify end-to-end on `:8070`. |
| [`pending/framework-dir.md`](pending/framework-dir.md) | Framework/consumer directory layout spec (reference for CLI and codegen). |
| [`pending/future-review.md`](pending/future-review.md) | Correctness review checklist: error handling, nil safety, resource cleanup, type assertions, HTTP/dispatch paths. |
| [`pending/general_todo.md`](pending/general_todo.md) | Config precedence done; open: notification broadcast, sync.Map cache replacement, future correctness review. |
| [`pending/hedwig_cleanup.md`](pending/hedwig_cleanup.md) | Remove the deprecated Hedwig flows editor `api.ts` and migrate consumers to the Hestia client. |
| [`pending/hestia-audit-report.md`](pending/hestia-audit-report.md) | Deep-dive audit (4 critical, 6 high, 9 medium, ~8 low findings) — reference and fix backlog. |
| [`pending/login_credential_boundary.md`](pending/login_credential_boundary.md) | Unify the auth/users credential boundary: no raw `SystemUser` (password hash) across it; add a login-response regression test. |
| [`pending/persistence-events-stream.md`](pending/persistence-events-stream.md) | Expose go-anansi persistence events over SSE. Guide written in `docs/guide/streaming.md`; design questions and implementation tasks are open. |
| [`pending/sanitize_type_blindness_followups.md`](pending/sanitize_type_blindness_followups.md) | Surface document sanitization failures instead of dropping rows; make the upstream sanitizer type-aware. |
| [`pending/user_query_filter.md`](pending/user_query_filter.md) | QDSL `filter`/`filters` fix landed; follow-ups: reject unknown top-level query keys upstream, restart the `:8090` dev server. |

## Done

| File | Summary |
|---|---|
| [`done/auth-session-iam-integration.md`](done/auth-session-iam-integration.md) | Server session restore + `@asaidimu/iam` client integration (multi-tab, heartbeat, `session()`). |
| [`done/blobs-1.3.5-update.md`](done/blobs-1.3.5-update.md) | Blobs `v1.3.5` bump, migration-report logging, `CleanupPendingError` handling, namespace lifecycle test. |
| [`done/blob_slash_keys.md`](done/blob_slash_keys.md) | Greedy catch-all route segments so blob keys containing `/` route correctly. |
| [`done/prebootstrap-session-probe.md`](done/prebootstrap-session-probe.md) | Make `system:auth:session:get` bootstrap-safe so a fresh server returns 401 (not 403) and the auth page can render. |
| [`done/review-fixes-and-notes.md`](done/review-fixes-and-notes.md) | Boot lifecycle fixes (`Close`/`Shutdown`, `Start` errors, `topoSort`, DI) and the 13-note devnotes merge. |
| [`done/root_rename.md`](done/root_rename.md) | `administrator` → `root` role rename across rules, bindings, seeds, tests, client, and docs. |
| [`done/sse-stream-lifecycle.md`](done/sse-stream-lifecycle.md) | S-19 follow-up: transport/context-lifetime fix, panic-safe `streamsink`, logs cursor, audit subscribe ordering, workflow replay dedupe, TS `openStream`/async-stream migration, in-repo SSE E2E test, CORS credentials. |
