# Self-Update Service (`updates`) — Implementation

**Scope:** `core/runtime`, `core/hestia.go`, `core/system/updates/` (new), wiring in `core/system/{provider,module,seeds}.go`, `core/system/settings`, `core/system/notifications`, `examples/basic`.
**Spec:** `todo/update_feature.md` (approved design). Deliberate deviations: updater repo left untouched (the settings store marshals a local tagged struct instead of adding JSON tags upstream); "feature" is a *service*; env-driven provider resolution added per `todo/general_todo.md`.

- [*] runtime config: `Config.Version`, `SelfUpdate *SelfUpdateConfig`, `SelfUpdateConfig` type, env-file precedence (`godotenv.Overload`)
  - **Files:** `core/runtime/config.go`.
- [*] env resolution: `ApplyEnvOverrides(conf)`, `APP_VERSION`, `UPDATE_*` knobs, GitHub/server provider construction
  - **Files:** `core/runtime/env.go` (new).
- [*] deps: add `github.com/asaidimu/updater` (v1.1.0) and `github.com/Masterminds/semver/v3` (direct)
  - **Files:** `go.mod`, `go.sum`.
- [*] `Setup` plumbing: `SetupConfig.SelfUpdate`, `applyTo` copies Version + SelfUpdate, `ApplyEnvOverrides` call, pre-boot `HandleStartup` hook, CLI uses `conf.Version`
  - **Files:** `core/hestia.go`.
- [*] `settings.ErrNotFound` sentinel (both `settings` copies) so the store can `errors.Is`
  - **Files:** `core/system/settings/model.go`, `core/system/settings/model/model.go`.
- [*] updates store: settings-backed `Store` (key `updates:pending`, tenant "") + `Reconcile(currentVersion)` (clear only when both versions parse and pending <= running; leave row on parse failure)
  - **Files:** `core/system/updates/store.go`.
- [*] pre-boot `HandleStartup(cfg, conf)`: `updater.New` + `HandleUpdateMode` + `Cleanup` (Store nil — DB not open yet)
  - **Files:** `core/system/updates/startup.go`.
- [*] updates service: Status/Changelog/Check/Apply handlers, `RunScheduledCheck` (system identity), `update_available` notify to users with `administrator` permission (in-app + email when mailer configured), last-check persisted
  - **Files:** `core/system/updates/service.go`.
- [*] manual registrations + policy bindings (NOT the service generator — conditional gating): `system:updates:{status:get,changelog:get,check:create,update:apply}`, all `administrator`
  - **Files:** `core/system/updates/register.go`.
- [*] `update_available` template seed
  - **Files:** `core/system/notifications/seed.go`.
- [*] wiring: `ProviderSet.{Updater,Updates}`, conditional `InitModels` (build, reconcile, schedule `updates:check`), gated `CollectRegistrations`
  - **Files:** `core/system/provider.go`.
- [*] wiring: conditional policy bindings in `Capabilities` + `allDefaultPolicies()` helper used by `SeedPolicies` and `SeedAll`
  - **Files:** `core/system/module.go`, `core/system/seeds.go`.
- [*] demo: commented `SelfUpdate` block (programmatic + env variants)
  - **Files:** `examples/basic/main.go`.
- [*] tests: store round-trip + reconcile, service handlers with stub provider, config precedence
  - **Files:** `core/system/updates/store_test.go`, `core/system/updates/service_test.go`, `core/runtime/config_test.go`.
- [*] verify: `go build ./... && go test ./...`
- [*] E2E: `tests/e2e/update/` — real binary swap (v1.0.0 → v1.1.0) via in-process RSA-signed update server + `DB_PATH` file DB; programmatic SelfUpdate persistence probe; both pass (`go test ./tests/e2e/update/ -count=1`)
  - **Files:** `tests/e2e/update/main_test.go`, `tests/e2e/update/programmatic_test.go`, `cmd/test-server/main.go` (`version` var + `PORT` env).
  - **Gotchas found & fixed along the way:**
    - CLI interface flag-parses `os.Args[1:]` and `os.Exit(1)` on unknown args — post-swap the new process died on `--perform-update`; `startup.go` now resets `os.Args = os.Args[:1]` after `HandleUpdateMode` returns.
    - `dispatch.Handle` panics on nil input doc → all 4 messages register an explicit empty Input schema (`dispatch.SchemaFromType[NoInput]()`).
    - updater was constructed WITHOUT `Store` in `initUpdates` → pending row never written; now a single store instance is shared by updater + service.
    - test-server `DBPath: ":memory:"` is flaky (connection pool hands out fresh schemaless connections) → E2E uses `DB_PATH` file DB (honored by `ApplyEnvOverrides`).
    - swap waits on the old PID via `signal(pid,0)`, which sees an unreaped zombie as alive → E2E must `go cmd.Wait()` the old process promptly.
    - in-process hestia embeddings under `go test` must pass `BuildInterfaces` returning nil (CLI interface chokes on `-test.*` flags).