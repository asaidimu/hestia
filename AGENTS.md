## Persistence Layer
This codebase relies on **go-anansi** for database operations and its proprietary query language.

* **Local Development Path:** `~/projects/go-anansi` for reference.
* **Before writing or migrating any feature, follow the `core/system/<name>/users` reference service** — it is the canonical how-to (layout, schema+projections, codegen workflow, DTOs, domain methods, wiring, tests), and see `todo/migrate_features.md` for the migration plan.
* Schemas that contribute towards a collection go into `core/system/<feature>/model/*.schema.json` as plain JSON files (package `model`).
* The meta schema reference is at `~/projects/go-anansi/core/schema/meta/schema.json`.

## IAM Layer

Identity and Access Management is handled via **go-iam**.

* **Local Development Path:** `~/projects/go-iam`

## Test Server

A live, auto-reloading test server runs continuously at `./cmd/test-server` on port **8070**.

Because operations often require authentication, you must first establish a session using the following endpoint:

* **Endpoint:** `POST /api/system/session`
* **Payload:**
```json
{
  "email": "admin@test.local",
  "password": "password123"
}

```
## Discovering Commands

To discover and understand all available registered commands within the system, query the documentation endpoint:

* **Endpoint:** `GET /api/system/core/docs/list`

---

## Tests

The generated model collections are **process-wide singletons** bound to the
persistence of the first full boot. Consequently a full application boot can
only happen **once per test process** — a second boot hits the closed
database. Tests that need a second boot are therefore **skipped by default**
and must be **unskipped** (remove the `t.Skip`) when running the full suite for
changes that affect bootstrap, so regressions are caught.

Current skipped boot tests:
* `TestFirstRunSuppressesKeyWhenBootstrapped` in
  `core/internal/boot/firstrun_test.go` (passes in isolation; run with
  `go test -run TestFirstRunSuppressesKeyWhenBootstrapped ./core/internal/boot/`).

---

## Writing TODOs

When starting on a new taks, write the steps to a file under the `todo` folder.

### Rules & Best Practices

* **Provide Enough Context:** Because tasks may be picked up by another agent or across different sessions, **never create single-phrase TODOs**. Each task must include enough background, intent, relevant file paths, links, or specific requirements so any agent can take over seamlessly without asking for context.
* **Track State Clearly:** Update task statuses promptly as work progresses.

### Task Status Identifiers

* `[ ]` Tasks that are planned.
* `[-]` Tasks that are in progress.
* `[*]` Tasks that are done.
* `[=]` Tasks that are skipped (include brief reasoning).
* `[X]` Tasks that are blocked (include the blocker details).

### Format Example

```markdown
- [ ] Implement user auth middleware
  - **Context:** Required for protecting `/api/v1/dashboard` routes.
  - **Details:** Use JWT validation based on existing spec in `docs/auth.md`.
  - **Files:** Modify `src/middleware/auth.js` and add unit tests to `tests/auth.test.js`.

```

