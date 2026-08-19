# Command Router (cobra-like setup system for hestia)

## Background / Intent
Running `test-server --version` (and `--help`) currently runs the full setup
(LoadConfig → `application.Boot` → migrations → model hydration → interfaces)
because flags are only parsed inside `cli.Interface.Start()`, which runs during
`app.Start()` AFTER `hestia.Setup` boots. The goal is a setup system that:
1. Loads config and parses flags/vars FIRST.
2. Routes into a command table (switch) that users can extend with custom
   commands — cobra-like but adopted for hestia (no external dependency).
3. Short-circuits commands that need no boot (`--version`, `--help`) and runs
   boot-stage commands (`--bootstrap`/`setup`) without starting the HTTP server.

## Decisions (user-approved)
- Lightweight internal router in `core/setup` (no spf13/cobra dependency).
- Bootstrap = Boot-stage (migrations + models + modules, NO HTTP server).
- Both flag + positional invocation (`--version` and `version`).
- Rename `--perform-update` → `--update` (cross-repo: updater lib + hestia).

## Files (primary)
- `core/setup/` — new package: `stage.go`, `command.go`, `router.go`.
- `core/hestia.go` — `Setup` refactor to route before `application.Boot`.
- `core/interface/cli/orchestrator.go` — strip version/help/bootstrap switch.
- `cmd/test-server/main.go`, `cmd/auth-test/main.go`,
  `cmd/hestia-desktop/main.go` — guard `if app == nil { return }`.
- `~/projects/updater/updater.go`, `process.go`, `updater_test.go` — flag rename.

## Task Status
- [ ] Implement `core/setup` package (Stage, Command, Context, Router)
  - **Details:** `StageNoBoot` (config only), `StageBoot` (persistence/models,
    no server), `StageServe` (default). Router matches positional name first,
    then registered `--flags`; `Help()` renders the command table.
  - **Files:** new `core/setup/{stage,command,router}.go`.
- [ ] Refactor `hestia.Setup` to route before `application.Boot`
  - **Details:** after `LoadConfig`/`applyTo`/`ApplyEnvOverrides`, build router
    (built-ins + new `SetupConfig.Commands []setup.Command`), parse `os.Args`,
    route: NoBoot → run + return `(nil,nil)`; Boot → `application.Boot` +
    `RegisterModules`, skip interface building, run + close + return; Serve →
    current behavior. Move `updates.HandleStartup` into router as NoBoot cmd.
  - **Files:** `core/hestia.go`, `core/interface/cli/orchestrator.go`.
- [ ] Add built-in commands: `version`/`-v`/`--version`, `help`/`-h`/`--help`
  (NoBoot), `bootstrap`/`setup`/`--bootstrap` (Boot).
- [ ] Rename `--perform-update` → `--update`
  - **Context:** flag hardcoded in updater lib (`~/projects/updater/updater.go`
    lines 172, 274; `process.go:105`; `updater_test.go:104,832`) AND hestia
    (`core/system/updates/startup.go`). Must land together.
  - **Details:** private spawn-and-swap handoff between ApplyUpdate and
    HandleStartup; safe rename as long as both sides change together. Keep
    forward-args handling intact.
- [ ] Update entry points to guard `app == nil`
  - **Details:** handled commands return `(nil, nil)` from `Setup`.
  - **Files:** `cmd/test-server/main.go`, `cmd/auth-test/main.go`,
    `cmd/hestia-desktop/main.go`.
- [ ] Tests
  - **Details:** router unit tests (NoBoot short-circuits pre-boot, Boot runs
    without serving, Serve boots). Integration: `test-server --version` prints
    version without creating a DB/binding 8070; `--help` prints table;
    `--bootstrap` runs admin setup then exits without serving.
  - **Run:** `make test` + client suite if touched.