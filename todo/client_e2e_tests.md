# Client E2E Test Conversion

- [*] Reorganize `client/` to mirror `core/system/<feature>` layout (git mv, `HestiaCapabilities` → `HestiaCore`, rules merged into `system/policies/rules.ts`).
- [*] Convert all mock-based (`vi.mock("@asaidimu/network-client")`) tests to real E2E tests against the test server (port 8070), per directive: no mocks, tests are E2E; auth disabled so no sessions needed.
  - Converted: `core/collection.test.ts`, `system/{users,apikeys,audit,collections,core,operations,policies(rules+policies),schedules,auth,notifications,settings,blobs}/*.test.ts`.
  - Kept pure unit tests (no network): blobs engine tests (FakeTransport/AimdController/ResumableUpload/UploadQueue/UploadJobStore), HttpTransport SSE parser tests, wails transport test.
  - Deleted mock-based `client/core/routes.test.ts` (superseded).
- [*] Added shared helpers `client/tests/helpers.ts`: `makeClient()` (base `http://localhost:8070`), `uniqueId()`, `collectionSchema()` (valid anansi meta-schema with canonical system field UUIDs).
- [*] Fixed `client/system/blobs/store.ts` `createNamespace` to pass required `{ ns }` arg; made `ns` required in `CreateNamespaceRequest`.
- [*] Fixed `HestiaCollections.create` to unwrap `{ name, schema }` wrapper — server expects the bare `SchemaDefinition` as payload.
- [*] Added `_busy_timeout=5000` to SQLite DSN in `core/internal/boot/database.go` — the shared in-memory DB raced background writes (audit/event bus) against test requests, causing flaky `database query/operation failed` (SQLITE_BUSY).
- [*] Removed lossy client-side normalizers — server already returns proper documents with real `_id_`/`_metadata_` on single-item endpoints (verified manually via curl):
  - `system/collections/store.ts` `find`/`read` now pass through server documents (was inventing `_id_=name`, fake `_metadata_`).
  - `system/operations/store.ts` `toDocument` now passes through server `_id_`/`_metadata_` when present, falls back to synthesized only for list items (which the server returns as plain `{name, description}` records).
  - `system/blobs/store.ts` `asDoc` was already passthrough-correct (spread overrides synthesized values for head/get; list items legitimately lack per-item `_id_`/`_metadata_` so the fallback stands).
  - Tests strengthened to assert real `_id_`/`_metadata_` passthrough for collections create/list/get, operations read, blob upload/head.
- [*] Audited client dispatch params vs server input bindings (`BindToTag` / `input:"..."` tags) for redundant requirements:
  - **Fixed:** `HestiaCollections.create` no longer requires `name` — the server derives it from `schema.name` (`core/system/collections/handler.go:51`). Tests create with `{ data: { schema } }` and assert the derived name round-trips.
  - **Confirmed non-redundant:** rules `create`/`update` name (server `resource_id="name"`, reads from arguments), blob upload `key`/namespace `ns` (URL args), settings `key`+`value`, schedules `user_id` (already optional — defaults to authenticated user), users email/name/password (server-required), notifications `id`, documents `id`.
- [*] Made `make test-client` run vitest with `--fileParallelism=false` — E2E tests share one server (single in-memory SQLite); serial execution is required for determinism.
- [*] Verified: `bunx tsc --noEmit -p tsconfig.json` clean; `make test-client` green twice in a row (20 passed files / 1 skipped, 219 passed / 3 skipped tests).

## Known server-side limitations (excluded from E2E, documented in test files)
- Notifications `markRead`/`markAllRead`: server returns `ERR_PERSISTENCE_VALIDATION_FAILED: Unexpected field 'expires_at'` (`_notifications_` schema lacks `expires_at`). Covered: list + countUnread only.
- Auth password reset/confirm: SMTP sink down (port 1025). Auth bootstrap: would mutate admin password. Both excluded.
- Policy `create`: `POLICY_ALREADY_EXISTS` for every seeded operation. Policies E2E covers query/list/read/setEnabled.
