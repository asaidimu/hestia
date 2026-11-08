# Update blobs library to v1.3.5

**Status:** done

## Context

`go.mod` pinned `github.com/asaidimu/blobs v1.3.4`. v1.3.5 (released
2026-11-06) fixes a real multi-tenant data-isolation bug class:

- namespaces are now keyed by an immutable UID instead of their name;
- blob manifests/chunk records are owned per namespace (identical bytes in
  two namespaces no longer overwrite each other);
- `DeleteNamespace` is two-phase and actually erases index records, segment
  files, WAL and the directory (returns `*errors.CleanupPendingError` when the
  erase is deferred);
- `Open` upgrades a format-1 index in place and exposes
  `Store.MigrationReport()`.

The upgrade is **one-way**: back up `<DataDir>/blobs/` before the first boot
with v1.3.5, and do not open the upgraded index with v1.3.4.

## Tasks

- [*] Bump `github.com/asaidimu/blobs` to `v1.3.5` in `go.mod`/`go.sum`
  - **Details:** only the blobs version changes; no new transitive deps.
- [*] Log the index migration in `core/system/blobs/store/service.go`
  - **Details:** after `store.Open`, if `MigrationReport().Ran`, log the
    namespace/blob/chunk counts and warn for each `Unrecoverable` chunk.
  - **Why:** an operator must know an in-place, one-way upgrade ran and
    whether any blob chunks could not be recovered.
- [*] Handle `CleanupPendingError` in `Service.DeleteNamespace`
  - **Details:** `errors.As` the blobs error; log a warning (namespace name,
    UID, cause) and return nil, because the name is already released and the
    namespace is unreachable. `Open`/`CleanupDeletedNamespaces` finishes the
    erase.
  - **Why:** otherwise a cancelled request or backend failure reports delete
    failure after the namespace is already gone.
- [*] Verify
  - `go build -modfile=/tmp/opencode/hestia-blobs135.mod -tags sqlite_fts5 ./...` — pass
  - `go test -count=1 -modfile=/tmp/opencode/hestia-blobs135.mod -tags sqlite_fts5 ./core/internal/boot/... ./core/runtime/di/... ./core/system/auth/... ./core/interface/http/... ./core/system/blobs/...` — pass
  - `go vet` on `./core/system/blobs/...` — pass
  - `go.mod`/`go.sum` diff is only the blobs version.
- [*] Add `TestServiceNamespaceLifecycle` in
  `core/system/blobs/store/service_test.go`
  - **Details:** create → put/get → delete → re-create the same name, and
    assert the re-created namespace does not see the old blob. This exercises
    the v1.3.5 namespace-isolation/erase behaviour through the Hestia wrapper.

## Operational notes

- Back up `<DataDir>/blobs/` before the first v1.3.5 boot.
- Never downgrade to v1.3.4 after the index has been upgraded (format 2).
- Migration runs synchronously inside `store.Open`, so boot can take longer
  on a large store.
- `DeleteNamespace` may leave data on disk after returning success (logged
  warning); it is erased on the next `Open`.

## Follow-up (not blocking)

- `#blob-20261029-011`: the Hestia `DeleteNamespace` handler still
  unregisters handlers/bindings before calling the store, so a failure
  mid-way can leave a namespace without handlers. The v1.3.5 store side is
  now atomic, but the Hestia-side ordering is still worth revisiting.
