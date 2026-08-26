# Front-end log viewing

- [ ] Expose application error logs to front-end users via an API
  - **Context:** Front-end users currently have no way to view application
    error logs. When something breaks they must locate the app's data
    directory on the host filesystem and read log files manually — not
    possible for hosted/non-technical users, and impossible from the web UI.
  - **Current state:**
    - Runtime logs are zap-based, JSON-encoded (`core/internal/boot/logger.go`).
    - Audit logs ARE exposed (`system:audit:log:list`, `system:audit:log:stream`)
      but only cover dispatched operations, not runtime errors (panics,
      handler failures like `updates: apply failed`, scheduler dispatch
      errors, etc.).
    - The silent-failure debugging session on schedules (Aug 2026) showed how
      valuable in-app error visibility is: the failure was only diagnosable by
      instrumenting code and reading server stdout.
  - **Possible approaches (pick one):**
    1. **In-memory ring buffer sink** for zap (e.g. last N=1000 entries,
       level>=warn), queryable via a new message
       `system:core:logs:query` (rule=administrator) returning recent
       entries with level/time/message/fields. Cheap, no persistence, survives
       nothing — good enough for "what just broke".
    2. **Zap sink → audit collection**: register a custom zapcore.WriteSyncer
       that also persists warn/error entries into `_audit_log_` with an
       entry type marker; reuses existing list/stream endpoints and auth.
       Risk: noisy, couples runtime logging to DB availability.
    3. **Logfile + tail endpoint**: configure zap OutputPaths into DataDir and
       add a range-query/tail endpoint. Persistent but adds file I/O and path
       management.
  - **Recommendation:** option 1 (ring buffer) as the core, optionally
    combined with 2 later for durability.
  - **Files:** new `core/system/logs` feature (schema/model/handler/service/
    registrations/policies following the `core/system/users` reference),
    logger wiring in `core/internal/boot/logger.go`.
