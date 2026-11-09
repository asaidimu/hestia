# Pre-bootstrap session probe

## Problem

A fresh (non-bootstrapped) Hestia server could not reach its own auth page.
`GET /api/system/auth/session/get` answered **403 `ERR_ACCESS_DENIED`**
("handler not available until bootstrapped") because
`system:auth:session:get` was not marked `BootstrapSafe`. The client SDK's
`HestiaAuth.session()` only maps 401 to "no session" and rethrows everything
else, so `HestiaClient.authenticated()` threw during `initializeClient()`.
Consumers (e.g. Hedwig's `ClientProvider`) then kept `serverReady = false` and
the auth page never rendered — the `SetupForm` was unreachable.

## Fix

- `core/system/auth/service.go` — added `bootstrap_safe="true"` to the
  `@hestia.register` annotation for `system:auth:session:get`.
- `core/system/auth/registrations.go` — regenerated with
  `go run ./cmd/hestia service generate system auth` (do not hand-edit).
- `core/system/auth/registrations_test.go` — new regression test pinning that
  the registration is `BootstrapSafe`.

Pre-bootstrap, the handler now runs and returns **401 `UNAUTHORIZED`**
("no active session"), which the SDK maps to `null`, so the auth page can
render `SetupForm`.

## Verification

- `go build ./...`
- `go test ./core/system/auth/... ./core/runtime/...`
- Manual: a fresh in-memory app on `:8091` returned
  `health.check -> bootstrapped:false` and
  `session/get -> 401 UNAUTHORIZED` (previously 403).

## Follow-up

- Publish the hestia release (or use a local `replace`) and bump
  `github.com/asaidimu/hestia` in the Hedwig server, which currently pins
  `v1.16.1` and therefore still answers 403 until rebuilt.
- Optional hardening: the Hedwig UI's `initializeClient()` could isolate the
  `authenticated()` probe so a transport/auth error cannot prevent
  `serverReady`; the server fix removes the trigger, but a failing probe
  currently still blocks the auth page.
