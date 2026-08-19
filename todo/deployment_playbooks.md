# Deployment Playbooks — Self-Updating Daemon

**Status:** done.
**Intent:** Ship a hestia app as a systemd daemon that auto-updates from a private GitHub Releases repo, and document the release-publishing side that makes it work.
**Decision:** `UPDATE_SYSTEMD=true` systemd-native apply mode (swap-in-place + clean exit; systemd `Restart=always` restarts the swapped binary as a tracked process). The unmodified spawn-and-swap pipeline is documented as a fallback (Section B of the systemd playbook).

- [*] code: `SelfUpdateConfig.SystemdMode` + `UPDATE_SYSTEMD` env
  - **Files:** `core/runtime/config.go`, `core/runtime/env.go`.
- [*] code: thread `SystemdMode`/`ExePath`/`DataDir` through `Dependencies` + `ProviderSet` (`UpdExe`/`UpdData`) + `initUpdates`
  - **Files:** `core/system/updates/register.go`, `core/system/provider.go`.
- [*] code: systemd-native `apply()` in the service (PrepareUpdate → swap executable temp+rename → ack → delayed `os.Exit(0)`); `exitProcess` indirection for tests; Check/RunScheduledCheck auto-apply reuse it
  - **Files:** `core/system/updates/service.go`.
- [*] tests: `TestSystemdApplySwapsExecutableAndExits`, `TestSystemdApplyUpToDateDoesNothing`, `UPDATE_SYSTEMD` env assertion; E2E still green
  - **Files:** `core/system/updates/service_test.go`, `core/runtime/config_test.go`.
- [*] docs: `docs/playbooks/release-publishing-github.md`
- [*] docs: `docs/playbooks/self-update-github-systemd.md`
- [*] verify: `go build ./... && go vet ./... && go test ./...` (incl. `tests/e2e/update`)