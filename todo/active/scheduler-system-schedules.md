# System schedules never dispatch (notification cleanup, token blocklist prune)

**Status:** fix implemented and unit-tested; production deployment pending.
**Opened:** 2026-11-13
**Affected:** `core/system/schedules`, `core/system/module.go`
**Reported by:** user — "scheduler runs unreliably; root notifications are never cleaned"

## Symptom

The hourly `system:notifications:notification:cleanup` schedule (and
`system:auth:token:blocklist:prune`) appeared in
`GET /api/system/scheduler/job/list` with `prev` advancing every hour, but no
cleanup ever happened. The scheduler itself was running; the *dispatch* was
skipped.

## Evidence from the local production server (`:8090`, v1.16.2)

- `GET /api/system/schedules/schedule/all` returns both seeded schedules with
  `"user_id": ""` and `"protected": true`.
- `GET /api/system/scheduler/job/list` shows both jobs registered with
  `expr: "@every 1h"` and `prev: 2026-11-13T13:42:46+03:00`.
- `/var/lib/hedwig-server/hedwig-server.log` repeats every hour:

  ```text
  warn schedules/liveschedule.go:176
    schedule: dispatch skipped — creator is missing or no longer active
    schedule_id=01a1c06628c67f379c8272f6f1b3c2c7
    message=system:notifications:notification:cleanup creator=""
  ```

## Root causes

1. **Creator-less system schedules were treated as deleted creators.**
   The S-1 hardening made `LiveSchedule.dispatch` resolve the creator's current
   claims and skip the fire when `user_id == ""`. Seeded system schedules
   intentionally have no creator, so every tick was skipped.
2. **Boot ordering registered seeds only after restart.**
   `SystemModule.Setup` called `LiveSchedule.Init` *before* `seedData` (which
   calls `SeedSchedules`). On a fresh install the seeded rows therefore existed
   but were never registered with cron until the process restarted.

## Fix

- `core/system/schedules/liveschedule.go`
  - New `fireContext` helper. User schedules still fire as the creator's
    current claims (S-1 unchanged). Creator-less **protected** schedules fire
    under `runtimecontext.SystemContext` (same identity as `updates:check`)
    and still traverse the full middleware chain. A creator-less schedule that
    is **not** protected is skipped and warned — an unattributed row must
    never escalate to SYSTEM.
- `core/system/module.go`
  - `LiveSchedule.Init` now runs *after* `seedData`, so first-boot seeds are
    registered immediately.
- `core/system/schedules/liveschedule_system_test.go`
  - Protected system schedule dispatches as SYSTEM (no user claims).
  - Creator-less unprotected schedule is skipped.
  - User schedule keeps creator identity and is not SYSTEM.

## Verification

```sh
go test ./core/system/schedules/ -count=1 -timeout 180s
go build ./core/... ./cmd/...
```

Both pass locally.

## Remaining work

- [ ] Deploy the rebuilt binary to `/opt/hedwig-server/hedwig-server` and
      restart `hedwig-server.service` (requires root; not available in the
      agent session).
- [ ] After deploy, confirm the warn line is gone and the cleanup log line
      appears hourly in `/var/lib/hedwig-server/hedwig-server.log`.
- [ ] Note: cleanup only deletes notifications whose `expires_at` has passed
      (`read_at + 30 days`). The root test notifications currently expire
      around 2026-12-11, so they will not disappear immediately even with the
      fix in place — verify by creating a notification with a past expiry.