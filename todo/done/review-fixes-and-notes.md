# Review fixes + unmerged devnotes

**Status:** done

## Context

`review.md` lists five weaknesses found in the boot/DI/devnotes layers. Four are
still present on `main` (`10cd000`); item 5 (devnotes index scoped to Anansi)
was already fixed. Two Git bundles in the repo root also carry review notes that
were never merged:

- `hestia-review.bundle` (`review/auth-restore-and-blobs-devnotes` @ `99b1d74`)
  — auth + blob notes are already on `main`; only the 6 `#tool-20261029-*`
  notes on `cmd/hestia/*` are unmerged.
- `hestia-devnotes-20261103.bundle` (`devnotes/err-20261103` @ `e116d62`) — 7
  `#err-20261103-*` error-reporting notes, unmerged.

## Tasks

- [*] Fix `Application.Close()` so it runs the shutdown sequence
  - **Context:** `core/internal/boot/app.go:339` only syncs logs and closes
    persistence; `README.md:103,162` tell users `defer app.Close()` after
    `Start()`, which leaves modules/interfaces running.
  - **Details:** make `Shutdown` idempotent and track whether `Start` ran;
    `Close` calls `Shutdown(context.Background())` before closing resources.
  - **Result:** `Application` now tracks `started`/`stopped`/`closed` under a
    mutex. `Close` runs `Shutdown` first (no-op if `Start` never ran) and is
    safe to call twice; `Shutdown` is idempotent. Covered by
    `core/internal/boot/lifecycle_test.go`.
- [*] Propagate module startup failures from `Start()`
  - **Context:** `app.go:297-312` logs `m.Start()` errors and continues;
    `core/hestia.go:130-136` always returns nil.
  - **Details:** collect errors with `errors.Join`, still start interfaces,
    return the joined error; update `cmd/docs-server/main.go`.
  - **Result:** `boot.Application.Start()` returns `errors.Join(...)`; the
    public `hestia.Application.Start()` forwards it; `cmd/docs-server` panics
    on failure.
- [*] Reject missing module dependencies in `topoSort`
  - **Context:** `app.go:243-249` only counts a dependency when it exists in
    `byName`, so typos are silently ignored despite the doc comment.
  - **Details:** return a boot-time error naming the module and the unknown
    dependency; add a regression test.
  - **Result:** `topoSort` returns `module %q depends on unknown module %q`;
    `TestTopoSortRejectsMissingDependency` and
    `TestTopoSortOrdersDependencies` cover it.
- [*] Make DI transient resolution goroutine-safe
  - **Context:** `core/runtime/di/container.go:226-240` shares `resolving` and
    `stack` across goroutines and releases the lock before calling the
    constructor, so concurrent resolution of the same transient is reported as
    a cycle.
  - **Details:** use the cycle tracking only during single-threaded `Build()`;
    at runtime, resolve without touching shared tracking. Add a concurrency
    regression test.
  - **Result:** `resolve` only touches `resolving`/`stack` while
    `c.building` is true. `TestConcurrentTransientResolutionDoesNotReportCycle`
    (8 workers) and `TestBuildDetectsCycle` cover both directions.
- [*] Merge the 13 unmerged devnotes
  - **Context:** 6 `#tool-20261029-*` notes (`cmd/hestia/*`) and 7
    `#err-20261103-*` notes (`core/interface/http/transport_fasthttp.go`,
    `core/system/module.go`, `core/system/collections/handler.go`,
    `client/packages/core/core/errors.ts`).
  - **Details:** apply the note-only hunks from the bundles, then hand-patch
    `.devnotes/index.json` (do **not** run `devnotes migrate` or
    `devnotes index update` — both mangle legacy headers and lose data).
  - **Result:** both patches applied cleanly (`git apply --check` before
    applying); index hand-patched with 13 entries and refreshed per-file
    `hash`/`noteIds`. `devnotes list` shows all 13; the 10 touched files are
    not reported stale by `devnotes index status`.
- [*] Verify
  - `go build -modfile=/tmp/opencode/hestia-verify.mod -tags sqlite_fts5 ./...`
    — pass.
  - `go test -count=1 -modfile=... ./core/internal/boot/... ./core/runtime/di/... ./core/system/auth/... ./core/interface/http/...`
    — pass.
  - Client typecheck: only the pre-existing `system/blobs/compaction.test.ts`
    errors remain.
