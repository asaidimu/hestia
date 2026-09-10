# Anansi upgrade: sqlite helpers, views, FTS5 + sqlite decoupling

**Intent:** Adopt go-anansi ≥ v8.9.0 (sqlite `Config`/`Handle` construction
helpers, `CreateView`/`RefreshView` virtual + materialized views, FTS5
text-search) while removing the implicit sqlite dependency from hestia core so
downstream apps only compile sqlite (mattn/go-sqlite3, cgo, `-tags
sqlite_fts5`) when they opt into the default sqlite backend.

**Reference repos:** `~/projects/go-anansi` (upstream: `sqlite/helpers.go`,
`sqlite/query/fts.go`, `sqlite/query/view.go`, `core/persistence/collection/view.go`,
`example/fts/main.go`, `example/views/main.go`). Canonical feature layout:
`core/system/users` (see `todo/migrate_features.md` if present).

- [*] Phase A — bump go-anansi to v8.9.1 and verify new APIs
  - **Context:** hestia pins v8.6.4; helpers/views/FTS landed in v8.8.0–v8.9.0.
  - **Details:** `go get github.com/asaidimu/go-anansi/v8@v8.9.1 && go mod tidy`;
    confirm `sqlite.Config/Handle/NewInteractor/NewMemoryInteractor`,
    `Persistence.CreateView/RefreshView`, `Collection.Refresh`,
    `query.TextSearch().Contains/Exact/Phrase`, `definition.IndexTypeFullText`
    resolve. Record any breaking changes from the bump.
  - **Files:** `go.mod`, `go.sum`.
- [*] Phase B — decouple sqlite from core (breaking change)
  - **Context:** `core/internal/boot/database.go` hand-rolls sqlite DSN/wiring;
    `persistence.go` falls back to it when `PersistenceFactory == nil`;
    `core/hestia.go` + `core/runtime/config.go` type the factory via the
    anansi root package, which transitively imports `sqlite` + mattn. Every
    downstream binary therefore compiles sqlite even when unused.
  - **Details:**
    1. New factory contract in `core/runtime` (no anansi-root import), e.g.
       `PersistenceDeps{Logger, DataDir, DocumentFactoryConfig}` +
       `PersistenceFactory func(deps) (p, cleanup, err)`. Decide
       `InteractorFactory` fate (recommend: remove, single shape).
    2. Core builds persistence via `data.ConfigureDocumentFactory` +
       `persistence.NewPersistence` directly; missing factory fails fast with
       an actionable error.
    3. New opt-in package `core/persistence/sqlite` exposing
       `Default(dbPath, opts...)` / `Memory()` factories; move
       `database.go` logic there on top of the new helpers; only this package
       imports `go-anansi/.../sqlite/*` + mattn.
    4. Migrate callers: `examples/basic`, `examples/wails-test`,
       `cmd/hestia-desktop`, `cmd/hestia/core/command.go`, `cmd/gen-routes`,
       `cmd/docs-server`, `cmd/test-server`, `tests/e2e/update/programmatic_test.go`.
    5. Rebase `core/internal/testutil/persistest.go` onto the plugin.
    6. `go mod tidy`; mattn becomes indirect.
  - **Files:** `core/runtime/config.go`, `core/hestia.go`,
    `core/internal/boot/persistence.go`, `core/internal/boot/database.go`
    (move), `core/internal/boot/database_test.go` (move),
    `core/persistence/sqlite/*` (new), callers above.
  - **Acceptance:** `go list -deps ./core/... | grep -E 'mattn|anansi.*/sqlite'`
    is empty; a binary importing only `hestia/core` builds without sqlite.
- [*] Phase C — sqlite helpers adoption (inside the plugin)
  - **Context:** Plugin replaces the hand-rolled DSN (`database.go`) with
    `sqlite.Config` + `NewInteractor`/`NewMemoryInteractor`, inheriting
    WAL / `_fk=1` / 5s busy-timeout / 4-4 pool defaults; `Handle.Cleanup` is
    the closer. Overrides via plugin options.
  - **Files:** `core/persistence/sqlite/*`.
- [*] Phase D — FTS5 text-search adoption
  - **Context:** `IndexTypeFullText` indexes → FTS5 `_fts` tables + triggers;
    `TextSearch(field).Contains/Exact/Phrase`; bm25 ranking; requires
    `-tags sqlite_fts5` else `no such module: fts5`.
  - **Details:** Scope `GOTAGS=sqlite_fts5` to sqlite-consuming targets
    (Makefile, `.air.toml`, CI, test-server build); declare a fulltext index
    via normalize → `migrate generate` → `codegen golang`; verify
    `collections` query passthrough (`query.FromBytes` handles
    `text_search_query`) with integration tests mirroring upstream
    `example/fts`; document tag requirement + index + query shape.
  - **Files:** build files, one `*.schema.json` + migration, collections
    tests/docs.
- [*] Phase E — views as public collection messages
  - **Context:** `Persistence.CreateView/RefreshView`; virtual (fresh,
    query-composed) vs materialized (CTAS snapshot + `Refresh`, no
    `_metadata_`/`_id_`); writes rejected (`ErrReadOnly`).
  - **Details:** Shipped `system:collections:view:create` (payload
    `{query, materialized}`, target schema auto-resolved from persistence)
    and `system:collections:view:refresh` (admin-gated, regenerated
    registrations/policies/routes). Reads flow through the existing
    `document:query` message. Tests in `core/system/collections/views_test.go`
    (composition, read-only, refresh semantics, materialized drop).
  - **Files:** `core/system/collections/{views,views_test,service,inputs,outputs}.go`,
    `model/{inputs,outputs}.go`, regenerated `registrations.go`/`policies.go`,
    `client/core/routes.gen.ts`.
  - **Follow-up:** no `view:delete` message — virtual views have no physical
    table and `Persistence.Delete` forces `DeletePhysicalData=true`, so
    dropping them needs upstream `DeleteView` (or drop-without-physical)
    support in go-anansi. Materialized views ARE droppable via
    `Persistence.Delete` (covered in test).
  - **Upstream bug found 2026-09-10 (go-anansi `generatePhysicalName`):**
    physical table names are truncated to 24 chars
    (`core/persistence/registry/utils.go`, `maxLength = 24`), so distinct
    collections/views sharing the first ~18 sanitized chars + version map
    onto ONE physical table, and CTAS `CREATE TABLE IF NOT EXISTS`
    silently reuses the stale table (wrong-data reads, no error). E2E
    proof: `e2e_view_snap-<ts>-<rand>` views collided across runs
    (timestamp prefix stable ~11 days, random suffix truncated off);
    snapshots kept serving a 3-row table from an earlier run while the
    base had 2. Client E2E works around it with short unique names
    (`client/packages/core/core/data-view.test.ts:shortId`); system
    collections are unaffected (short stable names), but any user
    collection with a long common prefix will collide. Needs an upstream
    fix (hash/uniqueness suffix in physical names).
- [*] Final verification
  - **Details:** `go build ./...` (± `sqlite_fts5`), `go vet`, full
    `go test ./...` (unskip `TestFirstRunSuppressesKeyWhenBootstrapped` per
    AGENTS.md Tests section), coupling grep check, test-server round-trip.
