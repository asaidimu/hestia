# Self-Update Feature (`updates`) — Complete Spec

**Scope:** `github.com/asaidimu/hestia` (`core/hestia.go`, `core/runtime`, `core/feature/*`, `examples/basic`), consuming `github.com/asaidimu/updater`.
**Status:** Design, approved. Decisions locked via Q&A: store = settings-collection-backed single row; feature registered only when `SelfUpdate` is configured; `HandleUpdateMode` runs inside `Setup` (pre-boot). No code written yet.

---

## 0. Premise

hestia apps should be able to self-update out of the box, on top of the shared
`updater` library (`github.com/asaidimu/updater`). The library already provides
the whole lifecycle — provider-based check/download, eager prepare-then-apply,
pre-boot swap, imperative `UpdateNow`, `Store`, `HandleUpdateMode`/`Cleanup` —
so the hestia work is *wiring*, not new mechanism: plumb config, run the swap
at the right point in `Setup`, and expose the check/approve/apply flow as a
message-driven built-in feature so every app gets it without writing
lifecycle code.

The ERP-shaped flow this enables, end to end:

1. nightly cron check (`updates:check`) — read-only `CheckForUpdate`
2. stage once (`PrepareUpdate`) — skipped when the pending version is already staged
3. `update_available` notification to admins
4. admin reviews changelog (`system:updates:changelog:get`)
5. admin applies in a maintenance window (`system:updates:update:apply`)
6. new binary starts → pre-boot swap in `Setup` → stale pending row self-clears

---

## 1. Grounded findings (verified against source)

### 1.1 Start order decides where the swap runs

- `Setup` (`core/hestia.go:244`) runs: Phase 1 `boot.New(conf)` → Phase 2
  `application.Boot` (opens the persistence manager and **applies
  migrations**, `core/internal/boot/app.go:56-77`) → builds the HTTP and CLI
  interfaces (`core/hestia.go:312-322`).
- `boot.New` is I/O-free — it only builds loggers and the dispatcher
  (`core/internal/boot/app.go:45-54`). Migrations run later, inside `Boot`.
- `Application.Start` (`core/internal/boot/app.go:214-229`) starts **modules
  first**, then **interfaces**. The CLI interface (`core/interface/cli/orchestrator.go:49-58`)
  parses `os.Args` with `flag` and `os.Exit(1)` on any unknown flag — so it
  would kill a `--perform-update` launch.

**Conclusion:** `HandleUpdateMode()` must run inside `Setup`, after
`boot.New(conf)` (so `conf.DataDir` exists) and before `application.Boot` (so
migrations run only after a successful swap — a failed swap must never leave
new DB schema behind the old binary). It runs before the CLI arg parse, and
the updater's args-restore means the CLI then sees the user's real arguments.

### 1.2 Version plumbing today

- `SetupConfig.Version` is passed only into the CLI interface
  (`core/hestia.go:318`, `cli.Config{Version: cfg.Version}`).
- `runtime.Config` (`core/runtime/config.go:38-65`) has **no `Version` field**.
  Features read config via `ps.Config` (`ProviderSet.Config`), so the feature
  needs `Version` on `runtime.Config` to know the running version.

### 1.3 Feature structure

- Features are packages exposing `Registrations(deps) []abstract.MessageRegistration`
  plus `PolicyBindings()`. `core/feature/provider.go` (`InitModels`,
  `CollectRegistrations`) and `core/feature/gen_features.go`
  (`allPolicyBindings` / `allDefaultPolicyBindings`) wire them all into the
  single `SystemModule`.
- The scheduler is `scheduler.Scheduler.Register(name, expr, fn)`
  (`core/runtime/scheduler/scheduler.go:40`); the notifications-cleanup job
  (`provider.go:104-112`) is the pattern for a recurring internal job.
- Notifications go through `abstract.Notifier.Send(ctx, abstract.Notification{...})`
  (`core/abstract/notification.go:22-30`; usage in `core/feature/auth/handler.go:76-86`).
- The settings feature (`core/feature/settings/model.go`) shows the minimal
  persistence pattern: a raw collection opened from `base.Persistence` with
  `Get`/`Set`/`Unset` — **no `schema.json`, no codegen**. This is the store
  backend for the pending-update row.

### 1.4 updater library surface (as built)

- `updater.New(provider, updater.Config{Version, DataDir, ExecutablePath, ForwardArguments, PermitUpdateFunc, Store})`.
- Methods: `CheckForUpdate(ctx)` (read-only), `PrepareUpdate(ctx)` (download +
  stage + `SaveUpdate`; skips when pending == latest and binary staged),
  `ApplyUpdate()`, `UpdateNow(ctx)`, `HandleUpdateMode()`, `Cleanup()`,
  `HasPreparedUpdate()`, `PendingUpdate(ctx)`.
- `updater.Provider` (`CheckForUpdate(ctx, currentVersion)`,
  `DownloadUpdate(ctx, info, destPath)`); built-ins `NewGitHubProvider`,
  `NewServerProvider`; shared `DownloadBinary`.
- `updater.Store` (`SaveUpdate`/`PendingUpdate`/`ClearUpdate`) — the only
  state the library ever persists is a single pending `UpdateInfo`.
- `UpdateInfo{Version, URL, Changelog, AssetName, Checksum}` — **no JSON tags
  today**, so marshaling a row needs either tags or explicit field mapping.

---

## 2. Design overview

Two coordinated additions:

- **A pre-boot self-update hook in `Setup`**, driven by a new
  `SetupConfig.SelfUpdate`. It runs `HandleUpdateMode()` + `Cleanup()` between
  `boot.New` and `Boot`, consuming `--perform-update` before the CLI and before
  migrations.
- **A built-in `updates` feature** (`core/feature/updates/`), active only when
  `SelfUpdate` is configured. It exposes status/changelog/check/apply messages,
  implements the updater `Store` over the settings collection (one row), runs
  the scheduled check, and notifies admins.

---

## 3. Part A — updater library (minimal)

### 3.1 JSON tags on `UpdateInfo`

Add `json` tags to `updater.UpdateInfo` fields so the settings-backed store can
marshal/unmarshal the single pending row (and so the row stays readable in the
DB):

```go
type UpdateInfo struct {
	Version   string `json:"version"`
	URL       string `json:"url,omitempty"`
	Changelog string `json:"changelog,omitempty"`
	AssetName string `json:"asset_name,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
}
```

No behavior change. Existing tests stay green.

---

## 4. Part B — config plumbing

### 4.1 `core/runtime/config.go`

Add two fields to `runtime.Config`:

```go
Version     string
SelfUpdate  *SelfUpdateConfig
```

Define `SelfUpdateConfig` in `runtime` (a leaf package — `feature/updates` and
`core/hestia.go` both need it, and neither can import the feature without a
cycle):

```go
// SelfUpdateConfig enables hestia's built-in self-update feature.
type SelfUpdateConfig struct {
	Provider updater.Provider // required: where updates come from

	// CheckSchedule is a cron expression for the recurring check; empty
	// disables the scheduled job (manual checks still work). Default "@every 24h".
	CheckSchedule string

	// DataDir is where staged update binaries live. Defaults to conf.DataDir.
	DataDir string
	// ExecutablePath is where the new binary is copied on swap. Defaults to os.Executable().
	ExecutablePath string

	// ForwardArguments restores the original CLI arguments after the swap.
	ForwardArguments bool
	// AutoApply applies an available update automatically during the scheduled
	// check. Default false — ApplyUpdate is always the explicit admin action.
	AutoApply bool
}
```

`runtime` gains `github.com/asaidimu/updater` as a dependency. (The updater
module is `go 1.26.5`; hestia is `go 1.27rc1` — compatible.)

### 4.2 `core/hestia.go`

- `SetupConfig`: add `SelfUpdate *runtime.SelfUpdateConfig`.
- `SetupConfig.applyTo` (`core/hestia.go:179-242`): copy `Version` and
  `SelfUpdate` into `conf`.
- The CLI interface continues to receive `cfg.Version` as today
  (`core/hestia.go:318`) — no change there.

---

## 5. Part C — pre-boot hook

### 5.1 `core/hestia.go`, inside `Setup`

Insert between `appWrapper := &Application{inner: application}` and the
`if !bootstrap { return appWrapper, nil }` early-return, so it also runs in
partial-bootstrap mode:

```go
if cfg.SelfUpdate != nil {
	if err := updates.HandleStartup(cfg.SelfUpdate, conf); err != nil {
		application.Close()
		return nil, err
	}
}
```

### 5.2 `core/feature/updates/startup.go`

```go
// HandleStartup consumes a --perform-update launch produced by ApplyUpdate
// (waiting for the old process, swapping the executable, clearing the pending
// update), then removes leftover staged binaries. It runs inside Setup, before
// migrations and before the CLI arg parse. Returns normally on a regular launch.
func HandleStartup(cfg *runtime.SelfUpdateConfig, conf *runtime.Config) error {
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = conf.DataDir
	}
	exe := cfg.ExecutablePath
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}
	}
	u, err := updater.New(cfg.Provider, updater.Config{
		Version:          conf.Version,
		DataDir:          dataDir,
		ExecutablePath:   exe,
		ForwardArguments: cfg.ForwardArguments,
	})
	if err != nil {
		return err
	}
	if u.HandleUpdateMode() {
		fmt.Println("updated; resuming normal operation")
	}
	if err := u.Cleanup(); err != nil {
		return fmt.Errorf("cleanup staged update: %w", err)
	}
	return nil
}
```

Notes:

- The provider is available here (`cfg.SelfUpdate.Provider`), so `updater.New`
  is fully constructed — `HandleUpdateMode`/`Cleanup` don't use it, but no
  nil-provider path is needed.
- **Store is deliberately nil at this point** — the DB (and the settings
  collection) isn't open yet. The pending row from a previous run is cleared by
  the post-boot reconcile (§6.2).
- `HandleUpdateMode`'s internal `os.Exit(1)` on a hard failure (bad args, wait
  timeout, copy failure) is the desired behavior: a broken swap must stop the
  process loudly.

---

## 6. Part D — the `updates` feature (`core/feature/updates/`)

### 6.1 `store.go` — updater.Store over the settings collection

Per decision: one row, backed by the existing settings collection, key
`updates:pending`, tenant "" (system-wide). The feature depends on
`*settings.SettingsModel` (already in `ProviderSet` as `ps.Settings`).

```go
const pendingKey = "updates:pending"

type Store struct{ Settings *settings.SettingsModel }

func (s *Store) SaveUpdate(ctx context.Context, info *updater.UpdateInfo) error {
	raw, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return s.Settings.Set(ctx, "", pendingKey, string(raw), "updater")
}

func (s *Store) PendingUpdate(ctx context.Context) (*updater.UpdateInfo, error) {
	v, err := s.Settings.Get(ctx, "", pendingKey)
	if err != nil {
		if errors.Is(err, errSettingNotFound) { // settings.Get returns "setting not found"
			return nil, nil
		}
		return nil, err
	}
	var info updater.UpdateInfo
	if err := json.Unmarshal([]byte(fmt.Sprint(v)), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *Store) ClearUpdate(ctx context.Context) error {
	return s.Settings.Unset(ctx, "", pendingKey)
}
```

(Note: settings.Get currently returns a bare `fmt.Errorf("setting not found")`
(`core/feature/settings/model.go:45`). Either match on the string, or — cleaner —
surface a sentinel error from settings. See §10 open question.)

### 6.2 `store.go` — reconcile (self-healing after pre-boot swap)

The pre-boot `HandleUpdateMode` can't clear the DB row. After boot, the feature
clears any pending record whose version is no longer ahead of the running one:

```go
// Reconcile drops a pending update that is obsolete relative to the running
// version (e.g. the row left behind by a pre-boot swap that ran before the DB
// was available). Call once at feature init.
func (s *Store) Reconcile(ctx context.Context, currentVersion string) error {
	pending, err := s.PendingUpdate(ctx)
	if err != nil || pending == nil {
		return err
	}
	if !newerThan(currentVersion, pending.Version) {
		return s.ClearUpdate(ctx)
	}
	return nil
}
```

`newerThan` uses `github.com/Masterminds/semver` (already a transitive dep via
updater; the feature imports it directly and hestia's go.mod lists it
explicitly). Non-semver versions fall back to a safe string comparison or
"leave it" — decide in §10.

### 6.3 `feature.go` — registrations (gated on `SelfUpdate != nil`)

```go
type Dependencies struct {
	Updater  *updater.Updater
	Store    *Store
	Notifier abstract.Notifier
	Logger   *zap.Logger
	AutoApply bool
}
```

| Message | Intent | Handler behavior |
|---|---|---|
| `system:updates:status:get` | Read | current version, staged version (`PendingUpdate`/`HasPreparedUpdate`), last-check time |
| `system:updates:changelog:get` | Read | staged `UpdateInfo.Changelog` + version/asset, for admin review |
| `system:updates:check:create` | Create | `CheckForUpdate` + `PrepareUpdate`; on a newly staged release, send `update_available` notification; if `AutoApply` → `ApplyUpdate()` |
| `system:updates:update:apply` | Create | admin-gated maintenance-window apply → `ApplyUpdate()` (process exits on success) |

Policy bindings for all four → `administrator` (matches the secure-by-default
default; explicit so the rule has a Description and shows in policy docs).

### 6.4 Scheduler — recurring check

In `core/feature/provider.go` `InitModels` (mirroring the notifications-cleanup
registration at `provider.go:104`):

```go
if su := ps.Config.SelfUpdate; su != nil {
	if su.CheckSchedule != "" {
		ps.Scheduler.Register("updates:check", su.CheckSchedule, func(ctx context.Context) error {
			return updates.RunScheduledCheck(ctx, ps.Updater, ps.Updates, ps.Notifier, ps.Logger, su.AutoApply)
		})
	}
}
```

`RunScheduledCheck` runs with a system identity (same pattern the CLI
bootstrap uses, `cli/orchestrator.go:96`):

```go
ctx = iam.WithIdentity(ctx, iam.Identity{Permissions: []string{runtime.SystemScopePrefix + ":updates"}, Properties: map[string]any{"system": "updates"}})
```

Flow: `CheckForUpdate` → nil = up to date → done. Else compare
`PendingUpdate().Version` against latest (the beta/hotfix case: a staged
`v3.0.0` with latest `v3.1.0` must re-stage); only download when they differ;
on a newly staged release notify admins; if `AutoApply` and a maintenance
window is open → `ApplyUpdate()`.

### 6.5 Notifications

Seed an `update_available` template (in the feature, matching the notifications
seed pattern) so `abstract.Notification{Recipient, Template: "update_available",
Data: {version, changelog, app_url}, Channels: [in_app, email]}` renders. Sent
only when a *new* version becomes staged (compare against the previous pending
version), never on a no-op re-check.

### 6.6 `policies.go`

`PolicyBindings()` returning the four `administrator` bindings. Because the
feature is conditional, do **not** add these to the static
`allPolicyBindings` var in `gen_features.go` — instead append them in
`SystemModule.Setup` (or the feature init path) only when
`ps.Config.SelfUpdate != nil`, so the policy seed and `SetKnownBindings`
never reference messages that aren't registered. See §7.

---

## 7. Part E — wiring

### 7.1 `core/feature/provider.go`

- `ProviderSet` gains `Updater *updater.Updater` and `Updates *updates.Store`.
- In `InitModels`, when `ps.Config.SelfUpdate != nil`:
  - construct `updater.New(provider, updater.Config{...})` (same resolution as
    `HandleStartup`),
  - build `updates.Store{Settings: ps.Settings}`,
  - run `Updates.Reconcile(ctx, ps.Config.Version)`,
  - register the scheduled check (§6.4).
- In `CollectRegistrations`, append `updates.Registrations(...)` **gated** on
  `ps.Config.SelfUpdate != nil`.

### 7.2 `core/feature/module.go`

- When self-update is configured, append `updates.PolicyBindings()` to the
  binding set used by `SeedPolicies` / `SetKnownBindings` (rather than touching
  the static `allPolicyBindings` in `gen_features.go`). Keep it a per-Setup
  slice so unconfigured apps never reference `system:updates:*`.

### 7.3 `core/feature/gen_features.go`

No change (bindings stay conditional via §7.2). Confirm `git diff --stat`
shows no unrelated regeneration — this feature uses a raw collection, not
codegen.

---

## 8. Part F — demo + tests

### 8.1 `examples/basic/main.go`

Add a commented `SelfUpdate` block showing the opt-in with a GitHub provider
for `asaidimu/hestia`, `Version` injected via ldflags:

```go
app, err := hestia.Setup(hestia.SetupConfig{
	SessionSecret: os.Getenv("SESSION_SECRET"),
	Version:       version,
	SelfUpdate: &runtime.SelfUpdateConfig{
		Provider:        ghProvider(), // updater.NewGitHubProvider(Owner: "asaidimu", Repo: "hestia", ...)
		CheckSchedule:   "@every 24h",
		ForwardArguments: true,
	},
})
```

### 8.2 Tests

- `core/feature/updates/store_test.go` — save/pending/clear round-trip over the
  settings collection; reconcile clears an obsolete pending row and keeps a
  newer one.
- `core/feature/updates/handler_test.go` — status/check/apply with a stub
  provider and the settings-backed store (modeled on `auth_test.go` /
  `settings` tests); check-notify fires only on a newly staged version.
- Optional integration test: a subprocess that launches the test binary with
  `--perform-update ...` through `Setup` with `SelfUpdate` set, asserting the
  swap happens and startup continues (mirrors `updater_test.go`'s
  `HandleUpdateMode` subprocess helper).
- Full suite stays green: `go build ./... && go test ./...`.

---

## 9. Suggested rollout order

1. updater: JSON tags on `UpdateInfo` (Part A).
2. `runtime.Config.Version` + `SelfUpdateConfig` + `applyTo` plumbing (Part B).
3. `updates.HandleStartup` + the `Setup` pre-boot hook (Part C).
4. `updates` feature: `Store` + reconcile + registrations + policies (Part D).
5. ProviderSet / module wiring + scheduler (Part E).
6. `examples/basic` demo + tests (Part F).
7. `go build ./... && go test ./...` full pass.

---

## 10. Open questions (decide during implementation)

- **settings error sentinel** (`settings/model.go:45`): `Get` returns a bare
  `fmt.Errorf("setting not found")`. Surface a package-level sentinel
  (`settings.ErrNotFound`) so the Store can `errors.Is` instead of string
  matching. Small settings change, worth doing.
- **Non-semver versions** in `Reconcile`: Masterminds parse failures should
  not clear or block — fall back to a bytewise compare or leave the row.
- **Notification recipient/template seeding**: confirm the existing template
  seed mechanism (`core/feature/notifications/seed.go`) and add
  `update_available` there or in the updates feature.
- **`AutoApply` + maintenance window**: decide whether apply is gated on an
  external signal (e.g. a runtime setting) or purely on the admin message.
  Default keeps `AutoApply` simple (immediate) with the window handled by when
  the cron fires.
