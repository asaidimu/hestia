# Writing Features

This guide distills how persistence-backed features are built on **go-anansi**
in this repo. It is the accumulated knowledge from converting `users` (the
reference implementation) and `apikeys`. Follow it when building a new feature
or migrating an existing one (audit, notifications, operations, policies,
schedules, settings, tenants).

**Read `core/feature/users/` and `core/feature/apikeys/` alongside this guide —
they are the canonical, fully-migrated examples.**

---

## 1. Canonical layout

A feature is two packages:

```
core/feature/<feature>/
├── feature.go      # Registrations(): wire handlers to messages
├── handler.go      # abstract.MessageHandler funcs, bind DTOs, call the model
├── policies.go     # PolicyBindings() (if the feature has IAM rules)
└── *_test.go
core/feature/<feature>/model/    # package model — everything persistence
├── <name>.schema.json           # schema + projections (hand-written)
├── <name>.schema.model.go       # GENERATED — model, collection, projections
├── data_transfer_objects.go     # DTOs + *Schema() accessors (hand-written)
├── <feature>_utils.go           # domain methods on the generated collection
└── *_test.go
```

- The **root package** (`apikeys`) owns handlers, message registrations, and
  policy bindings. It imports `model`.
- The **`model` package** owns the schema, generated code, DTOs, and domain
  methods. It never imports the root package.
- Generated model files are **never hand-edited** — extend via methods in a
  separate `*_utils.go` file.

---

## 2. Schema + projections (the single source of truth)

Schema lives at `model/<name>.schema.json`. Declare **everything** here,
including projections. Do not define Go structs separately — codegen derives
them from the schema.

```json
{
  "name": "_api_key_",
  "version": "1.0.0",
  "fields": {
    "019f4065-0a3d-7ea1-bc46-bbaeed4bfd6d": { "name": "_id_", "required": true, "unique": true, "type": "string" },
    "<uuid>": { "name": "name", "required": true, "type": "string" },
    "<uuid>": { "name": "hash", "required": true, "type": "string" },
    "<uuid>": { "name": "limits", "type": "object", "schema": { "id": "<uuid>" } }
  },
  "schemas": {
    "<uuid>": { "name": "Limits", "fields": { "<uuid>": { "name": "rph", "type": "integer" } } }
  },
  "metadata": { "projections": { ... } }
}
```

**Field IDs are UUIDv7 and must never be hand-generated.** When adding a field,
use the field **name as the key**; `anansi migrate generate` normalizes it to a
UUIDv7 and preserves it in the lockfile. Never rename an existing UUID key.

**Field shape is driven by required/nullable:**
- Required + non-nullable → value type (`Name string`).
- Optional or nullable → pointer type (`Expiry *string`, `Limits *Limits`).

### Projections

Declared under `metadata.projections`, keyed by name. They are field subsets
of the root schema, not new schemas. Three kinds:

| Kind | Purpose | Shape |
| --- | --- | --- |
| **Input** | Bound from the incoming message | `include` + `required`/`optional` + `tags` with `input:` |
| **Output** | Response shape, secrets excluded structurally | `include` or `exclude` |
| **Create output** | Response with a once-only field (e.g. raw key) | custom DTO struct, see §6 |

Projection DSL:

```json
"APIKeyCreate": {
  "fields": {
    "include":  ["name", "environment", "operations", "expiry", "limits", "ip"],
    "required": ["name"],
    "optional": ["environment", "operations", "expiry", "limits", "ip"],
    "tags": {
      "name":        { "input": "payload.{name}" },
      "environment": { "input": "payload.{name}" }
    }
  }
}
```

- `fields.include` — whitelist of root fields.
- `fields.exclude` — remove fields from the final set (use for output shapes).
- `fields.required` / `fields.optional` — force the generated Go type
  (value vs pointer).
- `fields.tags` — per-field struct tags with `{name}` placeholders.
  `"input": "payload.{name}"` or `"input": "arguments.{name}"`.

### The two-tag contract

Every generated struct carries tags:

- **`anansi` tag** — the persistence contract (flat doc field names).
- **`input` tag** — the dispatch binding contract (dotted paths into the
  incoming message), only on input projections.
- **`json` tag** — serialization.

### The `_id_` → arguments binding (solved)

The `_id_` field binds from `arguments.*` **directly through the projection
tag** — no DTO sibling hack needed:

```json
"_id_": { "input": "arguments.key_id" }
```

generates `ID string anansi:"_id_,required=true" json:"_id_" input:"arguments.key_id"`,
and `BindToTag(&input, "input")` lands the resource id in `input.ID`. Wire the
same name into `ResourceIDField` and `Arguments` in the registration.

---

## 3. The schema → codegen workflow (exact sequence)

```bash
anansi migrate generate --dry-run   # 1. validate + preview
anansi migrate generate             # 2. normalize IDs, write migration
anansi codegen golang               # 3. emit structs + collection + projections
```

- `migrate generate` rewrites field-name keys to UUIDv7, injects
  `_id_`/`_metadata_`, bumps the schema version, and writes a migration under
  `core/internal/migrations/`. It also updates `schemas.lock.json`.
- A metadata-only change (adding projections) still produces a patch migration
  — that's expected and fine.
- **`codegen golang` regenerates ALL feature schemas** (the `anansi.json`
  glob). After running it, `git diff --stat` may surface *unrelated* drift in
  other features (schema edited but not regenerated). Check and either revert
  unrelated generated files or fold them in deliberately.

---

## 4. What codegen emits (`<name>.schema.model.go`, package `model`)

- **Model struct** — `SystemAPIKey`, embeds `document.DocumentModel`, every
  field carries `anansi` + `json` tags.
- **Projection structs** — `APIKeyCreate`, `APIKeyUpdate`, `APIKeyPublic`, etc.
- **Collection wrapper** — `SystemAPIKeys` embeds
  `*collection.ModelCollection[*SystemAPIKey]`, plus `SystemAPIKeysCollectionName`.
- **Singletons** — `InitSystemAPIKeysModel(p, logger)`, `SystemAPIKeysModel()`,
  and `DangerouslyResetSystemAPIKeysModel()` for tests.

Optional fields are emitted with `anansi:"...,required=false,omitempty"` —
**`omitempty` in the anansi tag is what lets nil pointers (e.g. `Limits`,
`IPConfig` object fields) be skipped on create**. If you ever hit
`field "limits" expects an object (map), got <nil>`, it means the generated
file predates the omitempty fix — re-run `anansi codegen golang`.

---

## 5. DTOs: bind projections directly, no wrapper structs

Input DTOs **are** the projections. Don't wrap them in
`type XxxInput struct { Xxx }` — the projection already carries every
`input:` tag. Bind and derive schemas straight from it:

```go
// handler.go
var input model.APIKeyCreate
if err := msg.Input().BindToTag(&input, "input"); err != nil {
    return nil, err
}
```

```go
// data_transfer_objects.go
func APIKeyCreateInputSchema() *definition.Schema {
    return dispatch.SchemaFromTypeWithTag[APIKeyCreate]("input", true)
}
```

The `true` omits the embedded `document.DocumentModel` from the derived schema.

Simple argument-only inputs still get a small struct:

```go
type APIKeyGetInput struct {
    KeyID string `input:"arguments.key_id"`
}
func APIKeyGetInputSchema() *definition.Schema {
    return dispatch.SchemaFromTypeWithTag[APIKeyGetInput]("input")
}
```

Output DTOs are plain envelopes:

```go
type APIKeyOutput struct {
    Document APIKeyPublic `anansi:"document"`
}
type APIKeyListOutput struct {
    Documents []APIKeyPublic `anansi:"documents"`
}
```

> `serializeOutputField` (in `core/interface/http/register.go`) only checks that
> the output schema declares a top-level `document` / `documents` / `page`
> field, then passes the (sanitized) document body through. The inner schema is
> documentation/introspection, not a filter.

### Registering with derived schemas

```go
{ Name: "system:apikeys:key:update",
  Handler: NewUpdateAPIKeyHandler(deps.APIKeyModel),
  Intent: abstract.Update,
  Input: runtime.Input{
    Schema:          model.APIKeyUpdateInputSchema(),
    Arguments:       []abstract.ArgDef{{Name: "key_id", Type: definition.FieldTypeString}},
    ResourceIDField: "key_id",
    Payload:         definition.FieldTypeObject,
  },
  Output: model.APIKeyOutputSchema() }
```

---

## 6. Custom response shapes (revealing a once-only secret)

You **cannot** `doc.Set()` an arbitrary field onto a container-backed
`document.Document` — it's schema-bound, unknown paths error. To return a
field that isn't in the collection (e.g. the raw API key, shown once on
create/rotate), build a custom output DTO that embeds the public projection
and declares the extra field:

```go
type APIKeyCreatedOutput struct {
    document.DocumentModel `json:"-" anansi:"-"`
    APIKeyPublic
    Key string `anansi:"key,required=false" json:"key,omitempty"`
}
```

Then populate it by binding the created model into the public projection and
re-marshaling:

```go
func keyDocWithSecret(ctx context.Context, k *model.SystemAPIKey, rawKey string) (*document.Document, error) {
    var pub model.APIKeyPublic
    d, err := k.Document()
    if err != nil { return nil, err }
    if err := d.BindTo(&pub); err != nil { return nil, err }
    out := document.New(&model.APIKeyCreatedOutput{APIKeyPublic: pub, Key: rawKey})
    return out.Document()
}
```

---

## 7. Domain methods on the model (`*_utils.go`)

Add domain logic as methods on the generated collection in a hand-written
`model/<feature>_utils.go` file. Two patterns, both typed:

**Full create** — build the model with `document.New` and call `Create`:

```go
func (m *SystemAPIKeys) CreateKey(ctx context.Context, key *GeneratedKey, userID string, req *APIKeyCreate) (*SystemAPIKey, error) {
    active := "active"
    usage := int64(0)
    doc := document.New(&SystemAPIKey{
        Name: req.Name, UserID: userID, Prefix: key.Prefix, Hash: key.Hash,
        Operations: req.Operations, Status: &active, Usage: &usage,
        Limits: req.Limits, Ip: req.Ip, Environment: req.Environment,
    })
    if req.Expiry != nil { doc.Expiry = req.Expiry }
    return m.ModelCollection.Create(ctx, doc)
}
```

**Partial update** — use `UpdateFrom` with the projection:

```go
func (m *SystemAPIKeys) UpdateKey(ctx context.Context, keyID, userID string, req *APIKeyUpdate) (*SystemAPIKey, error) {
    if _, err := m.Get(ctx, keyID, userID); err != nil { return nil, err } // ownership
    return m.UpdateFrom[*APIKeyUpdate, *SystemAPIKey](ctx, keyID, req)
}
```

Rules of thumb:

- **Always route creates/updates through `document.New` / `UpdateFrom`** so the
  `DocumentModel` is wired — never map-based updates.
- **Ownership-scope reads** with the query DSL:
  `query.NewQueryBuilder().Where(data.DocumentIDField).Eq(keyID).Where("userId").Eq(userID)`.
- **Do not shadow the ModelCollection's `Create`/`Update`** — name your domain
  methods `CreateKey`/`UpdateKey` (or call `m.ModelCollection.Create`/`m.Update`
  explicitly inside). `Delete`/`Get`/`List` don't collide, so they're fine.
- **Set domain defaults explicitly** (`status: "active"`, `usage: 0`) — don't
  rely on the DB.
- **Best-effort side effects are best-effort**: don't fail a read just because
  a usage counter failed to bump.

---

## 8. Wiring

`core/feature/provider.go` initializes models once at startup:

```go
apiKeys, err := apikeysmodel.InitSystemAPIKeysModel(ps.Persist, ps.Logger)
if err != nil { return fmt.Errorf("init system api keys model: %w", err) }
ps.APIKeys = apiKeys
```

`ProviderSet` holds `APIKeys *apikeysmodel.SystemAPIKeys` and hands it to both
`apikeys.Registrations(...)` and any cross-feature consumers.

**Cross-feature consumers must be converted too.** When `apikeys` moved from a
hand-rolled `APIKeyModel` to the generated `*model.SystemAPIKeys`, we updated:
`core/feature/provider.go` (init + field type) and `core/feature/auth/` —
`feature.go` (Dependencies type), `token.go` (`CreateKey` + `APIKeyCreate`),
`adapters.go` (authenticator's model field). The call sites changed:
`Create(ctx, key, uid, &CreateKeyRequest{...})` →
`CreateKey(ctx, key, uid, &model.APIKeyCreate{...})`, `Expiry` went from
`string` to `*string`.

---

## 9. Sanitization

`core/internal/boot/persistence.go` configures global sanitization: `hash`,
`secret`, `token`, `api[_-]?key`, `credential` (and `password`) patterns are
redacted by the HTTP layer, with scoped overrides per collection. The HTTP
serializer always calls `Sanitize()`. Still, **keep secrets out of output
structurally** via output projections (`APIKeyPublic` excludes `hash`) — don't
rely on redaction alone.

---

## 10. Testing

Mirror `core/feature/users/model/system_user_test.go` and
`core/feature/apikeys/handler_test.go`:

```go
func testModel(t *testing.T) *model.SystemAPIKeys {
    t.Helper()
    model.DangerouslyResetSystemAPIKeysModel()
    m, err := model.InitSystemAPIKeysModel(testutil.NewPersistence(t), zap.NewNop())
    if err != nil { t.Fatalf("InitSystemAPIKeysModel: %v", err) }
    return m
}

func testMsg(name string, input data.Documenter) abstract.Message {
    return dispatch.NewMessage(name, context.Background(), input)
}
```

- The generated singleton pins the first persistence, so tests **must** call
  `DangerouslyResetXxxModel()` first, or construct fresh.
- Build message inputs with `data.MustNewDocument(map[string]any{...})` or
  `testutil.InputDoc(t, schema, json)` for schema-derived input.
- For `userID`-driven handlers, attach claims via
  `runtimecontext.ContextWithClaims(ctx, &abstract.Claims{UserID: ...})`.
- **Delete `zz_probe_test.go`-style scratch files** — they're debug leftovers
  that break the build the moment APIs change.

---

## Checklist — converting or adding a feature

1. Write `model/<name>.schema.json` with projections (input tags on input
   projections, secret-free output projection).
2. `anansi migrate generate --dry-run` → `anansi migrate generate` →
   `anansi codegen golang`. Check `git diff --stat` for unrelated regeneration.
3. Write `model/data_transfer_objects.go` — bind projections directly
   (`SchemaFromTypeWithTag[Projection]("input", true)`), small arg DTOs for
   argument-only inputs, output envelopes.
4. Write `model/<feature>_utils.go` — domain methods, ownership-scoped reads,
   `document.New` / `UpdateFrom`, no `Create`/`Update` shadowing.
5. Write `handler.go` — `BindToTag(&input, "input")`, call the model.
6. Update `feature.go` — registrations with derived schemas +
   `Arguments`/`ResourceIDField`/`Payload`.
7. Update `provider.go` (+ any cross-feature consumers like `auth`).
8. Migrate/rewrite tests; delete scratch tests.
9. `go build ./... && go test ./core/feature/<feature>/...`.
