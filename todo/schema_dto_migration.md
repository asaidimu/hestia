# Schema DTO Migration — Remaining Features

Migrate the remaining features from hand-written JSON-byte output schemas
(`var xJSON = []byte(...)` + `dispatch.MustFromJSON`) to the typed-DTO pattern
(`dispatch.SchemaFromType[T]()`), following `docs/guide/writing_features.md`
and the reference implementations in `users`, `apikeys`, and `blobs`.

## The pattern

**Before** (each feature's `schema.go`):
```go
var healthOutputJSON = []byte(`{ ... }`)
var _healthOutput = dispatch.MustFromJSON(healthOutputJSON)
func healthOutputSchema() *definition.Schema { return _healthOutput }
```

**After** (typed DTO structs + derived schemas, see `core/feature/blobs/outputs.go`):
```go
type HealthOutput struct {
    Document HealthView `anansi:"document"`
}
func healthOutputSchema() *definition.Schema { return dispatch.SchemaFromType[HealthOutput]() }
```

Wire shape stays identical: output DTO fields carry `anansi` tags and are built
into documents via `data.MustNewDocumentFromStruct`; the `Document` envelope
field keeps the top-level `document` marker that `serializeOutputField`
(`core/interface/http/register.go:253`) requires.

## Features to convert (each has a `schema.go` with JSON-byte outputs)

1. `core/feature/operations/` — health, docs, capabilities, message, scheduler-list (5)
2. `core/feature/policies/` — validate, reload, binding, rule, output, list-bindings, list-rules, list-policies (8)
3. `core/feature/collections/` — list, output, query, document (4)
4. `core/feature/auth/` — login, message, elevate, claims (4)
5. `core/feature/notifications/` — list, message, unread-count (3)
6. `core/feature/schedules/` — list, output, message (3)
7. `core/feature/settings/` — output, list, message (3)
8. `core/feature/audit/` — log-query, log-stream (2)

## In scope per feature

- Replace `var xJSON = []byte(...)` blocks and `dispatch.MustFromJSON` with
  typed output DTO structs + `SchemaFromType[T]()` accessors in a
  `data_transfer_objects.go`-style file (or `outputs.go`).
- Update the handler(s) to build results with
  `data.MustNewDocumentFromStruct(&view, ctx)` instead of
  `dispatch.MustNewDocument(map[string]any{...})` / `util.StructToMap`.
- Keep the wire format byte-for-byte identical (verify against the client SDK
  in `client/`).
- Update affected tests; delete any scratch/probe files.

## Out of scope (legit JSON-bytes, leave alone)

- `core/runtime/dispatch/input.go` — `InputMetaSchemaJSON` (the input
  meta-schema validator; hand-written by design).
- `core/internal/migrations/*.go` — generated `target__*_json` schema blobs
  (codegen output).
- `core/feature/collections/handler.go:44` — `definition.FromJSON(schemaBytes)`
  on runtime user-supplied schema JSON.

## Verification

- `go build ./... && go vet ./... && go test ./core/...`
- `cd client && bunx tsc --noEmit` (no new errors) and `bunx vitest --run`
- Manual sanity against the test server on :8070 for each feature touched.
