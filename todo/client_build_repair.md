# Client npm build / turbo repair

**Intent:** `cd client && bun run build` (what CI's version workflow runs)
failed; turbo repo misconfigured.

- [*] Add `packageManager: bun@1.4.0` to `client/package.json`
  - **Context:** turbo 2.10 refuses to resolve the workspace without a
    `devEngines.packageManager` or legacy `packageManager` field
    (`Could not resolve workspace`).
  - **Files:** `client/package.json`.
- [*] Replace deprecated tsdown `external` with `deps.neverBundle`
  - **Context:** mock build warned `` `external` is deprecated ``;
    same semantics (keep `@asaidimu/*` external).
  - **Files:** `client/packages/mock/tsdown.config.ts`.
- [*] Ignore turbo cache dirs
  - **Context:** `.turbo/` caches showed up as untracked after builds.
  - **Files:** `client/.gitignore`.
- [*] Verify: `bun install --frozen-lockfile` (no changes),
  `bun run build` (2/2 tasks successful, no warnings), no stray files.
