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
- [ ] Print the ephemeral API key to stdout on boot while the system is NOT
      bootstrapped, in `SystemModule.Setup` right after `seedData`
      (`core/system/module.go:96`), guarded by `!m.bootstrapped`.
  - **Details:** output goes to the same stdout used by the banner
    (`application.Loggers.Stdout`); include a hint that the key can be used via
    `X-API-Key`/`Authorization: Bearer` to bootstrap through the server.
  - **Files:** `core/system/module.go` (or surface via boot/app.go after
    RegisterModules so stdout logger is reachable).
- [ ] Surface the auto-generated admin email (when not provided via config)
      alongside the key so users know the seeded account.
  - **Details:** `SeedAdmin` returns `adminEmail`; only print the "auto-generated"
    block when no admin email was configured (`m.opts.AdminEmail`/`m.cfg.AdminEmail`
    both empty).
- [ ] Stop printing the key after bootstrap (security: it grants full admin).
      Decide: gate print on `!m.bootstrapped` only, or also add a
      `bootstrapped()` guard inside `APIKeyAuthenticator.Authenticate` so the
      ephemeral key is disabled post-bootstrap (recommend the latter too — see
      `core/system/auth/adapters.go:46-61`).
- [ ] Verify end-to-end on test server (port 8070):
      first boot prints key; authenticate with `curl -H "X-API-Key: <key>"`,
      call `system:auth:bootstrap:password:set` + `system:core:bootstrap:mark`,
      confirm server stops printing the key on subsequent boots.
- [ ] Tests: unit test that an unbootstrapped boot surfaces the key and a
      bootstrapped boot does not (may need a print-capture helper or a
      `PrintEphemeralKey` hook on SystemModule).

## Notes
- Related to the command-router work (`todo/command_router.md`): the CLI
  `--bootstrap` path uses the dispatcher directly; the printed key targets the
  HTTP/server bootstrap path.
- The ephemeral key grants admin claims unconditionally today even post-bootstrap
  — likely intended for one-shot bootstrap, so disable it after bootstrap while
  adding the print (see security note above).