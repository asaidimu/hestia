# First-run prints no API key — cannot bootstrap via server

## Issue
On first run the server never prints an API key to the screen, so users cannot
authenticate and bootstrap via the running HTTP server.

## Investigation (root cause confirmed)
- Every boot, `SystemModule.seedData` generates a random in-memory **ephemeral
  API key** (16 random bytes, hex) — `core/system/module.go:257-261` — and
  wires it into the `APIKeyAuthenticator` (`module.go:111`,
  `core/system/auth/adapters.go:47`).
- The HTTP middleware accepts that key via `X-API-Key` / `Authorization:
  Bearer` and calls `identityProv.Authenticate("api_key", key)` →
  `APIKeyAuthenticator.Authenticate` (`core/interface/http/middleware.go:87-106`,
  `core/interface/http/identity_provider.go:51-55`). When the presented key
  equals the ephemeral key it returns **full admin claims**.
- Therefore the key IS usable for server-side bootstrap (`system:auth:bootstrap:password:set`
  + `system:core:bootstrap:mark`), but the key is **never printed**: the
  `SystemModule.EphemeralKey()` getter (`module.go:442`) has no printing caller.
- SeedAdmin also auto-generates a random admin email/password when none are
  configured (`core/system/auth/seed.go:53-69`) — those are not surfaced either.
- The ephemeral key is regenerated on every process start (never persisted), so
  it must be printed on EVERY boot until the system is bootstrapped.

## Fix plan
- [*] Print the ephemeral API key to stdout on boot while the system is NOT
      bootstrapped.
  - **Details:** implemented `Application.printFirstRunKey()` in
    `core/internal/boot/app.go`, called after `SetSystemModule` in both `Boot`
    and `Reset`. Prints the key, a "not bootstrapped" hint, and the
    auto-generated admin email (when none configured). Added
    `UserOutput.Println` in `core/internal/boot/logger.go`.
- [*] Surface the auto-generated admin email (when not provided via config)
      alongside the key so users know the seeded account.
- [ ] Stop printing the key after bootstrap (security: it grants full admin).
      Print is already gated on `!m.bootstrapped`; the ephemeral key itself is
      still accepted by `APIKeyAuthenticator.Authenticate`
      (`core/system/auth/adapters.go:46-61`) even post-bootstrap — add a
      `bootstrapped` guard there to disable it.
- [ ] Verify end-to-end on test server (port 8070): the test server forces
      bootstrap, so this needs a non-forced run (e.g. a fresh `cmd` without
      `ForceBootstrapped`) — authenticate with `curl -H "X-API-Key: <key>"`,
      call `system:auth:bootstrap:password:set` + `system:core:bootstrap:mark`,
      confirm server stops printing the key on subsequent boots.
- [*] Tests: `TestFirstRunPrintsEphemeralKey` (asserts key printed on
      unbootstrapped boot) in `core/internal/boot/firstrun_test.go`.
      `TestFirstRunSuppressesKeyWhenBootstrapped` passes in isolation but is
      SKIPPED by default (second full boot per process collides with closed
      model singletons) — unskip when running the full suite for
      bootstrap-affecting changes (see AGENTS.md "Tests").

## Notes
- Related to the command-router work (`todo/command_router.md`): the CLI
  `--bootstrap` path uses the dispatcher directly; the printed key targets the
  HTTP/server bootstrap path.
- The ephemeral key grants admin claims unconditionally today even post-bootstrap
  — likely intended for one-shot bootstrap, so disable it after bootstrap while
  adding the print (see security note above).