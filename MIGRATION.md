# Migration Runbook — Target Architecture

This is an ordered, incremental path from the current tree to the one described in
`ARCHITECTURE.md`. Each phase is independently buildable and testable — commit and run the
full test suite after every phase, not just at the end. Phases 1–5 are pure structural
moves (no behavior change, low risk). Phase 6 contains the actual pattern-level fixes from
§11 of `ARCHITECTURE.md` — these change behavior and should be separate, self-contained
commits from the moves, not bundled with them.

**This runs locally, no PRs.** The existing checkout stays exactly as it is, under its
current folder name `server` — untouched, functional, a fallback you can always go back to.
The migration happens in a **separate git worktree**, checked out into a sibling folder
named `framework`:

```bash
cd server                                  # your existing, unmodified checkout
git worktree add ../framework -b migration/framework-restructure
cd ../framework
```

`framework` and `server` share the same `.git` history but are independent working
directories — you can build/test/run either one without the other interfering, and if the
migration goes sideways at any phase, `server` was never touched. Every phase below happens
inside `framework`; treat each phase as a commit on the `migration/framework-restructure`
branch, reviewed by re-reading your own diff before moving on, not by a PR process.

One naming note worth flagging given how much of this design process has been about
avoiding exactly this: `server` (the old folder) and `cmd/server/` (the new binary entry
point introduced in Phase 5) are unrelated to each other — the folder name is your local
checkout, the binary path is inside the repo. No action needed, just don't let the shared
word cause you to look in the wrong place while you have both worktrees open side by side.

`internal/` only becomes read-only-by-convention once this migration is judged complete and
`framework` replaces `server` as the working checkout (Phase 8) — see that phase for how to
enforce it locally without a CI/PR pipeline.

---

## Phase 0 — Baseline

1. Inside `server` (before creating the worktree), tag the current `HEAD` as
   `pre-migration`. This tag is shared history, so it's visible from `framework` too.
2. Confirm the full test suite passes on that tag. If it doesn't, fix that first, in
   `server`, and re-tag — you want a clean baseline to diff behavior against, not a
   migration that also happens to fix an unrelated failing test.
3. Create the `framework` worktree as shown above, branched from `pre-migration`.
4. **Resolve unverified items before relying on them**: open the actual source (in either
   worktree — they're identical at this point) for the three items flagged "unverified" in
   `ARCHITECTURE.md` §11 — the `NamespacedDispatcher` step (§11.2), the JWT_SECRET panic
   site (§11.12), and what `internal/module/system/feature/core/` actually does (§11.13).
   Each affects a specific later phase; resolve them now so those phases aren't blocked
   mid-move.

## Phase 1 — Scaffolding, no moves yet

Create the new empty directories so subsequent phases are pure `git mv`:
```
internal/abstract/
internal/core/blobstore/
internal/core/identity/
internal/app/feature/
internal/interface/api/http/
internal/interface/cli/
internal/boot/
internal/shared/testutil/
module/
cmd/server/
```
Nothing imports these yet. Commit as "scaffolding, no logic changes."

## Phase 2 — Extract `internal/abstract/`

Move interface declarations only — no implementations — out of `internal/core/dispatcher.go`,
`internal/module/module.go`, and `internal/transport/transport.go` into
`internal/abstract/{dispatcher,module,transport,message,result}.go`. Update every import
site. This phase touches the most files by import-count but changes zero behavior — it's
mechanical.

Add the `Kind ResultKind` discriminant to `Result` here (§11.5) — this is the one behavior
addition worth doing in the same phase as the type's move, since every call site touching
`Result` gets visited anyway. Do **not** yet remove the old nil-check logic in
`serializeResponse` — add `Kind` as populated-but-unused first, verify existing tests still
pass, then switch `serializeResponse` to use it in Phase 6 as its own reviewable diff.

Verify: `go build ./...` succeeds, full test suite passes.

## Phase 3 — Consolidate `internal/core/`

1. Move dispatcher implementations (`SecureDispatcher`, `LocalDispatcher`,
   `AccessLogDispatcher`) into `internal/core/`, renamed to kebab-case files per the
   established convention (`secure-dispatcher.go`, etc.).
2. Move `utility/blobs/{dispatcher,service}.go` → `internal/core/blobstore/`.
3. Move `utility/auth/{context,system}.go` → `internal/core/identity/`. Leave
   `utility/auth/password.go` where it is for now — it moves to `feature/auth/` in Phase 4,
   since it's feature-specific, not framework-generic.
4. Resolve the `NamespacedDispatcher` question from Phase 0 here: either give it a
   permanent home in `internal/core/` with a doc comment stating its purpose, or delete it
   if it's dead code.

Verify: build + tests.

## Phase 4 — Consolidate `internal/app/` (the default module)

This is the largest phase. Per feature (`auth`, `users`, `apikeys`, `policies`, `audit`,
`blobs`, `collections`, and the feature currently named `core`):

1. Move `internal/module/system/feature/<name>/` → `internal/app/feature/<name>/` as-is
   first (pure path move, verify build), *then* in a follow-up commit for that same
   feature:
2. Move the matching model file from `internal/module/system/models/` into the feature
   directory as `model.go`.
3. Move any feature-specific service files in from `internal/utility/` or
   `internal/module/system/services/`:
   - `auth/`: `services/jwt.go` + `utility/jwt/jwt.go` — **consolidate these into one
     `feature/auth/jwt.go`, do not move both.** Diff them first; if they're a thin-wrapper
     split, merge; if genuinely duplicated, delete one and update its callers.
   - `auth/`: `services/blocklist.go` → `feature/auth/blocklist.go`.
   - `auth/`: `utility/auth/password.go` → `feature/auth/password.go`.
4. Add `doc.go` to each feature per the file-role contract (§2.1 of `ARCHITECTURE.md`) —
   messages owned, one-line purpose, dependencies.
5. For the feature currently named `core` specifically: confirm what it does (Phase 0),
   rename its directory to match, update its imports and its message-name prefixes if those
   also said "core" — **check whether renaming its message names breaks anything currently
   calling `system:core:...`, since per §11.3 message names are effectively public API.**
   If it's already exposed and in use, keep the message name and only rename the Go package
   and directory, not the wire-level name.

`internal/module/system/models/seed.go` is cross-feature bootstrap data — move it to
`internal/app/` root (not into any single feature) as `bootstrap.go`. `models/seed.go` is
conceptually different from feature-owned `seed.go` files (e.g. `auth/seed.go`) — verify
this distinction against the actual file contents before moving; if they turn out to
overlap, consolidate rather than keep two divergent seeding mechanisms.

Delete `internal/utility/response/` if Phase 0 confirms it's empty/unused.

Verify after every feature: build + tests. Don't batch all seven features into one commit —
if something breaks, you want to know which feature broke it.

## Phase 5 — `internal/interface/` and `internal/boot/`

1. Move `internal/orchestrator/api/*` → `internal/interface/api/`.
2. Move `internal/orchestrator/cli/*` → `internal/interface/cli/`.
3. Move `internal/transport/http/transport.go` → `internal/interface/api/http/transport.go`
   (per the earlier decision: HTTP transport is API-only, doesn't need to be a top-level
   sibling of `interface/`).
4. Move `internal/core/registration/derive.go` → `internal/interface/api/derive.go` — this
   is HTTP route derivation specifically, not a module-agnostic core abstraction (§4.1 note
   in `ARCHITECTURE.md`).
5. Move `internal/utility/session/session.go` → `internal/interface/api/session.go`.
6. Move `internal/app/{app,builder,config,database,logger,persistence}.go` →
   `internal/boot/` (note: this is the *old* `internal/app` wiring package, being renamed —
   sequence this phase carefully against Phase 4, which creates a *new*, different
   `internal/app` as the default module. Do this rename first if there's any path
   collision risk, or use an intermediate directory name during the transition).
7. Move root `main.go` → `cmd/server/main.go`. This file now explicitly lists active
   modules: `boot.Run(app.New())`. Strip any module-specific knowledge out of
   `internal/boot/run.go` itself — it should only know `abstract.Module`, never `app` or
   any feature by name.

Verify: build + tests, and manually confirm the server still starts and serves a request.

## Phase 6 — Pattern-level fixes (behavior changes, separate commits)

Each of these is a `ARCHITECTURE.md` §11 item. Give each its own commit, in this order
(later ones depend on earlier ones), so a `git bisect` or a plain revert can isolate any one
fix without touching the others:

1. **§11.6 — single sanitization owner.** Remove the `Sanitize()` call from handlers;
   confirm `serializeResponse` still sanitizes (now via `Result.Kind`, using the field added
   in Phase 2). Add a test that a handler returning unsanitized data still comes out
   sanitized over HTTP.
2. **§11.4 — internal dispatch fast path.** Add the internal-only dispatch entrypoint that
   bypasses `SecureDispatcher`/`AccessLogDispatcher` by construction. Repoint JWT
   `token:validate`/`token:check` calls at it. Confirm the access log no longer contains
   these internal messages, and confirm token validation still enforces the same checks
   (write a test that an expired/blocklisted token still fails, going through the new path).
3. **§11.1 — CLI uses the full chain.** Change the CLI orchestrator to use the same
   `DispatcherChain()` as HTTP instead of the reduced `SecureDispatcher()`-only chain.
   Confirm CLI actions now appear in `_access_log_`.
4. **§11.8 — blocklist check becomes read-only.** Change `IsBlocklisted()` to a pure
   `SELECT ... WHERE exp >= now` with no `DELETE`. Add a periodic background purge (a
   ticker goroutine started in `boot.Run`, or reuse whatever job-scheduling mechanism
   already exists — check before adding a new one).
5. **§11.7 — self-registering features.** Extend `cmd/sdkgen` to scan feature directories
   and generate the registrations-aggregation file. This is the largest of the behavior
   changes — build it against the now-stable `internal/app/feature/*` tree from Phase 4, and
   verify by deleting a feature from the generated list and confirming its routes disappear.
6. **§11.3/§1.3.1 — message naming grammar + SDK derivation.** While `sdkgen` is already
   being extended in the previous step, add the `module:feature:scope:action` grammar
   check (exactly four segments, `action` from the fixed vocabulary, `action` agrees with
   `Intent`) and the class/method derivation rule (`class = PascalCase(module)+PascalCase(feature)`,
   `method = camelCase(action)+PascalCase(scope)`). Audit every currently-registered message
   name against the grammar first — the worked examples in `ARCHITECTURE.md` §1.3.1 suggest
   the existing names already comply, but confirm rather than assume, especially for the
   `collection:*` dynamic-collection namespace, which needs an explicit decision (generic
   parameterized client vs. excluded from static generation) rather than silently falling
   out of whatever `sdkgen` happens to do with a runtime-only feature segment.
7. **§11.10 — dependency structs for handler constructors.** Mechanical, low-risk, do
   per-feature alongside adding the missing handler tests (§11.11) since both touch the
   same constructor call sites.
8. **§11.12 — config fail-fast.** Change `NewConfig()` to return `error` instead of
   panicking; update `cmd/server/main.go` to handle it before any orchestrator starts.
9. **§11.9 — relocate schema JSON + update `sdkgen`.** Move `schemas/<name>/schema.json`
   into each feature's `schema/` directory. Before changing `sdkgen`'s scan path, read
   `migrations/registry.go` to confirm whether migration ordering is global or per-collection
   — if global, `sdkgen` needs to aggregate across feature directories in a stable order
   (e.g. alphabetical by feature, then by existing migration UUID), not just scan and emit
   in filesystem-walk order.

## Phase 7 — Testing gap (§11.11)

Add handler tests for every feature touched in Phase 4, using the mock-model pattern
already established for the features that do have tests. Do this per-feature, immediately
after that feature's Phase 4 commit, rather than as one large deferred effort — you're
already inside each handler's file re-establishing its dependencies via the new
`<Feature>Dependencies` struct (§11.10), which is the natural moment to also write the test
that exercises it.

## Phase 8 — Merge back and lock the boundary

Since there's no PR pipeline, "landing" the migration means folding `framework` back into
`server` deliberately, then enforcing the boundary locally going forward:

1. From inside `server`: `git merge migration/framework-restructure` (fast-forward, since
   `framework` was branched from `server`'s own history and nothing else landed on `server`
   in the meantime). At this point `server`'s working tree *is* the new structure —
   you can keep calling the folder `server` or rename it; the worktree served its purpose as
   an isolated space to build and test the migration without risking the working checkout.
2. Remove the now-merged worktree: `git worktree remove ../framework`.
3. Add a local **pre-commit hook** (`.git/hooks/pre-commit`, or checked into the repo and
   symlinked via a setup script, so it survives clones) that fails the commit if the staged
   diff touches `internal/` and the current branch isn't the designated canonical branch:
   ```bash
   #!/bin/sh
   canonical_branch="main"
   current_branch=$(git rev-parse --abbrev-ref HEAD)
   if [ "$current_branch" != "$canonical_branch" ]; then
       if git diff --cached --name-only | grep -q '^internal/'; then
           echo "internal/ is read-only outside '$canonical_branch'. Put this change in module/ instead, or switch branches if this really is a template update."
           exit 1
       fi
   fi
   ```
   This is a local, bypassable guard (`git commit --no-verify` skips it) rather than a
   server-enforced one — since there's no CI, that's the honest limit of what "read-only"
   means here. Document that limit alongside the hook so it isn't assumed to be stronger
   than it is.
4. Update `README.md` to state the `internal/` vs `module/` rule explicitly, including the
   fact that enforcement is a local hook, not a server-side gate.
5. Tag as `migration-complete`.

## Rollback notes

Every phase above is a separate commit by design specifically so any single phase can be
reverted without unwinding the others — Phases 1–5 are pure moves (revert = re-run the
inverse `git mv`), Phase 6 items are independent behavior changes (revert = revert that one
commit; none of them depend on a later Phase 6 item). Because the whole migration lives on
its own branch inside the `framework` worktree until Phase 8, the ultimate rollback at any
point before merging back is simply: delete the `framework` worktree and the
`migration/framework-restructure` branch. `server` was never touched, so there is nothing to
undo there. Phase 7 and 8 have no rollback risk beyond that, since they're additive (tests)
or local-tooling-only (the hook).
