# General TODOs

- [*] Config precedence: env should override hardcoded `SetupConfig` values
  - **Context:** `Setup` runs `runtime.LoadConfig` (env) then `cfg.applyTo(conf)`, so a hardcoded `SetupConfig` currently wins over env. Desired: env wins.
  - **Details:** After `applyTo` in `core/hestia.go`, re-apply every env-backed knob (non-empty only) via `runtime.ApplyEnvOverrides(conf)`.
  - **Files:** `core/hestia.go`, `core/runtime/config.go`, `core/runtime/env.go` (new).
  - **Status:** Implemented. `ApplyEnvOverrides` called in `Setup` after `applyTo`; tested in `core/runtime/config_test.go`.
- [*] Env-file precedence: `.env.dev` overrides `.env` which overrides set environment variables
  - **Context:** `LoadConfig` uses `godotenv.Load()`, which never overrides existing vars — today the precedence is process env > `.env` > `.env.dev` (the opposite of the desired order).
  - **Details:** Switch to `godotenv.Overload()` then `godotenv.Overload(".env.dev")` so `.env.dev` > `.env` > process env.
  - **Files:** `core/runtime/config.go`.
  - **Status:** Implemented; tested in `core/runtime/config_test.go` (`TestLoadConfigEnvFilePrecedence`).
- [ ] Notification broadcast capability
  - **Context:** The in-app and email channels target a single `Recipient`; the updates `update_available` alert must reach all admins, which today means a manual per-admin fan-out in the updates service.
  - **Details:** Add a broadcast path to the notifier (e.g. `abstract.Notifier.Broadcast` or a recipient resolver on channels) so one send reaches every user holding a permission/role; then consume it from the updates service.
  - **Files:** `core/abstract/notification.go`, `core/runtime/notification/notifier.go`, consumer `core/system/updates/`.