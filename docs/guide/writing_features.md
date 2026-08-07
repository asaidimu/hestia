# Writing Features

This guide distills what we've learned about building persistence-backed
features on **go-anansi**. It covers the six things that repeatedly bite us:

1. [Generating models from schemas](#1-generating-models-from-schemas)
2. [Generating projections with proper bindings](#2-generating-projections-with-proper-bindings)
3. [Binding input to DTO structs](#3-binding-input-to-dto-structs)
4. [Why DTO structs and not `map[string]any`](#4-why-dto-structs-and-not-mapstringany)
5. [Extending model functionality](#5-extending-model-functionality)
6. [Deriving runtime schemas for DTOs](#6-deriving-runtime-schemas-for-dtos)

The canonical example is the users feature: `core/internal/feature/users/`.
Read it alongside this guide.

---
## 0. Writing Schema
A collection schema is a plain JSON file. 
When working with schemas: 
1. Prefer writing schemas in json. The guide meta schema is at `~/projects/go-anansi/core/schema/meta/schema.json`
2. Schemas that contribute towards collection go into `./core/internal/feature/**/schema/*.schema.json` as plain json files

When updating the database schema, strictly adhere to the following workflow:

1. **Edit the Schema:** Modify the schema files directly.
* *Do not* alter existing IDs.
* Only modify field properties.
* If introducing a new field, set its ID to match the field's name. DO NOT GENERATE MANUAL UUIDS. The migration tool will detect this and assign a proper UUID. 
* If deleting a field, remove it's entry completely preserving the other fields.

2. **Preview Changes:** Validate and preview the migration by running:
```bash
anansi migrate generate --dry-run
```

3. **Generate Migration:** Finalize and generate the required migration files by running:
```bash
anansi migrate generate
```

## 1. Generating models from schemas

**Layout:** schemas that contribute to a persisted collection live at
`./core/internal/feature/<feature>/schema/<name>.schema.json`.

```json
{
  "version": "1.0.0",
  "name": "_user_",
  "fields": {
    "<uuid>": { "name": "email", "required": true, "unique": true, "type": "string" },
    "<uuid>": { "name": "name", "required": true, "type": "string" },
    "<uuid>": { "name": "disabled", "type": "integer", "default": -1 }
  },
  "metadata": {
    "projections": { ... }
  }
}
```

**Regenerate the Go model** after any schema edit:

```bash
anansi codegen golang
```

This emits `<name>.schema.model.go` next to the schema, containing:

- **The model struct** (`SystemUser`) — embeds `data.DocumentModel`, every
  field carries its `anansi` tag.
- **The collection wrapper** (`SystemUsers`) — embeds
  `*collection.ModelCollection[*SystemUser]`, plus
  `SystemUsersCollectionName` and the singleton `InitSystemUsersModel` /
  `UsersModel()` accessors.
- **Projection structs** (`UserRegister`, `UserUpdate`, `UserPublic`).

### Constructor vs. singleton

Prefer a **fresh constructor** over the generated singleton when you need
isolation (per-module setup, tests):

```go
// Generated (pins the first persistence forever):
users, err := schema.InitSystemUsersModel(p, logger)

// Hand-rolled fresh instance, good for tests and Reset lifecycle:
func NewSystemUsers(p base.Persistence, logger *zap.Logger) (*SystemUsers, error) {
    raw, err := p.Collection(context.Background(), SystemUsersCollectionName)
    if err != nil { return nil, err }
    mc, err := collection.NewModelCollection[*SystemUser](raw, logger)
    if err != nil { return nil, err }
    return &SystemUsers{ModelCollection: mc}, nil
}
```

> The generated singleton caches the first persistence it was built with.
> Reusing it across `Reset()` cycles or isolated test persistences silently
> hits the wrong database. When in doubt, construct fresh.

### Field shape is driven by required/nullable

- Required + non-nullable → value type (`Email string`).
- Optional or nullable → pointer type (`Disabled *int64`).

Optional fields must be pointers because **partial updates skip zero fields**
(see §5). That is why `UserUpdate` is all pointers.

---

## 2. Generating projections with proper bindings

Projections are declared under `metadata.projections`, keyed by name. They are
**field subsets** of the root schema, not new schemas.

```json
"UserRegister": {
  "fields": {
    "include":  ["email", "password", "name", "tenant_id", "data"],
    "required": ["email", "password", "name"],
    "optional": ["tenant_id", "data"],
    "tags": {
      "email":     { "input": "payload.{name}" },
      "password":  { "input": "payload.{name}" },
      "name":      { "input": "payload.{name}" }
    }
  }
}
```

Projection DSL:

| Key | Meaning |
| --- | --- |
| `fields.include` | Whitelist of root fields |
| `fields.exclude` | Remove fields from the final set |
| `fields.required` | Force `required=true` (upgrades to value type) |
| `fields.optional` | Force `required=false` (stays/downgrades to pointer) |
| `fields.tags` | Custom struct tags per field, `{FieldProperty}` placeholders |

### The two-tag contract 

Every generated struct carries two contracts (or 1+n where n is the number of tags assigned in the projection.):

- **`anansi` tag** — the persistence contract. Flat doc field names, used by
  `Patch()` / `NewPartialDocumentFromStruct` when writing to the DB.
- **`input` tag** — the dispatch binding contract. Dotted paths into the
  incoming message, used by `BindToTag(&dto, "input")`.

Use `fields.tags` to declare how a projection field arrives on the wire:

- `"input": "payload.{name}"` — field lands in the message `payload`.
- `"input": "arguments.{name}"` — field lands in the message `arguments`.

A projection with only an `anansi` tag (e.g. `UserPublic`) is a **pure output
shape** — never an input — and is how you keep `password` out of responses:

```json
"UserPublic": { "fields": { "exclude": ["password"] } }
```

> Keep projections **flat**. Do not nest them inside DTO structs as references
> to generated types; instead embed them flat (see §3) so the promoted fields
> carry both tag sets down.

---

## 3. Binding input to DTO structs

DTOs describe the dispatch I/O contract — the `{arguments, payload}` incoming
message and the `{document, page}` response. They are **not** persisted.

Put them in the schema package as a standalone file
(`core/internal/feature/users/schema/data_transfer_objects.go`) and export
both the types and their schema accessors (see §6).

### Input DTOs

Bind an incoming message into a typed struct with
`msg.Input().BindToTag(&dto, "input")`:

```go
type UserGetInput struct {
    UserID string `input:"arguments.user_id"`
}

type UserUpdateInput struct {
    UserUpdate // embeds the projection: promotes all anansi + input tags
    UserID     string `input:"arguments.user_id"`
}

func NewUpdateUserHandler(users *schema.SystemUsers) abstract.MessageHandler {
    return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
        var input schema.UserUpdateInput
        if err := msg.Input().BindToTag(&input, "input"); err != nil {
            return nil, err
        }
        // input.UserID, input.Name, input.Email, ...
    }
}
```

Embedding the projection (`UserUpdate`) is how an input DTO inherits all of the
projection's `input:"payload.*"` tags without repeating them. The `arguments`
bound fields are added as siblings.

### The `_id_` special case

The binder treats `_id_` specially — it only binds from the document's
internal id, **not** from `arguments.user_id`. Until that fix lands, use an
explicit DTO field for the resource id:

```go
UserID string `input:"arguments.user_id"`
```

and reference it in the registration's `Arguments` / `ResourceIDField`:

```go
{ Name: "system:users:user:get",
  Input: runtime.Input{ Schema: schema.UserGetInputSchema(),
    Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}},
    ResourceIDField: "user_id" } }
```

### Output DTOs

Response DTOs use `anansi` tags to position fields in the response document.
Output them as the password-free projection:

```go
type UserOutput struct {
    Document UserPublic `anansi:"data"`
}
```

---

## 4. Why DTO structs and not `map[string]any`

The short answer: **"Have you met `UpdateFrom`?"** Working with raw documents
and maps bypasses everything go-anansi gives you.

| `map[string]any` | DTO structs |
| --- | --- |
| No compile-time checks; typo'd keys silently no-op | Field names checked at compile time |
| Update semantics are your problem | `UpdateFrom[*UserUpdate]` skips zero fields for you |
| No validation without extra code | `Validate` / schema-driven validation for free |
| Spreads field names as string literals across the codebase | Field names live in one schema |
| Outputs leak fields unless you remember to scrub | `UserPublic` projection excludes `password` by construction |

Typed partial update:

```go
updated, err := users.UpdateFrom[*schema.UserUpdate, *schema.UserPublic](ctx, id, &input.UserUpdate)
```

Typed full/small updates via the model type:

```go
update := data.New(&schema.SystemUser{Email: email})
m.Update(ctx, id, update)
```

Both routes skip zero fields and never clobber the `DocumentModel` system
embed (`_id_` / `_metadata_`) — something a hand-rolled `map[string]any`
merge almost certainly gets wrong.

> If you need to inspect *which* fields a struct would update, use
> `data.NewPartialDocumentFromStruct(&input.UserUpdate, ctx)` and check
> `len(patch.ToMap())` — still no map[string]any plumbing.

---

## 5. Extending model functionality

Generated `ModelCollection` gives you the generic CRUD (see go-anansi's
`docs/codegeneration.md` §4):

| Method | Behaviour |
| --- | --- |
| `Create(ctx, doc P)` / `CreateMany` | Persist, return hydrated model |
| `FindByID(ctx, id)` / `Read(ctx, q)` | Read into `P` |
| `Update(ctx, id, update P)` / `UpdateMany` | Partial update, zero fields skipped |
| `Replace(ctx, id, replacement P)` | Full replacement |
| `DeleteByID` / `DeleteMany` | Delete |
| `ReadAs[R]` / `CreateFrom[R]` / `UpdateFrom[R]` | Shape operations over projections |

Domain logic goes in a **hand-written extension file** in the schema package,
e.g. `system_user_utils.go`, as methods on the model:

```go
func (m *SystemUsers) Register(ctx context.Context, email, password, name, tenantID string, data map[string]any) (*SystemUser, error) {
    // query existing, hash password, then:
    user := data.New(&SystemUser{
        Email: email, Password: hashed, Name: name,
        Disabled: &minusOne, TokenVersion: &zero,
    })
    return m.Create(ctx, user)
}

func (m *SystemUsers) ChangePassword(ctx context.Context, id, newPassword string) error {
    hashed, _ := runtime.HashPassword(newPassword)
    _, err := m.Update(ctx, id, data.New(&SystemUser{Password: hashed}))
    return err
}

func (m *SystemUsers) IncrementTokenVersion(ctx context.Context, id string) error {
    user, _ := m.GetByID(ctx, id)
    next := int64(1)
    if user.TokenVersion != nil { next = *user.TokenVersion + 1 }
    _, err := m.Update(ctx, id, data.New(&SystemUser{TokenVersion: &next}))
    return err
}
```

Rules of thumb:

- **`data.New(&T{...})` sets `parent`**, which makes `Patch()`,
  `MustDocument()`, and `ID()` work on the result. Always route creates and
  updates through it.
- **Never use map-based updates.** Prefer `Update` with a struct shape or
  `UpdateFrom` with a projection.
- **Bootstrap default state explicitly.** `disabled=-1` (enabled),
  `token_version=0`, `verified=false` are the domain defaults — set them in
  `Register`, don't rely on the DB.
- **Keep accessors returning interfaces when they must cross boundaries.**
  Go has no covariant returns: a method declared as `GetActiveByID(...)
  (abstract.UserIdentity, error)` must return the interface, not `*SystemUser`.
  Implement the interface on the model struct with plain getters.

---

## 6. Deriving runtime schemas for DTOs

The dispatch layer needs a `*definition.Schema` for every registration input
and output. Derive it from the DTO struct with `dispatch.SchemaFromType` so it
never drifts from the Go type.

In `core/runtime/dispatch/input.go`:

- `dispatch.SchemaFromType[T]()` — derives a meta-schema from the `anansi`
  tags of `T`.
- `dispatch.SchemaFromTypeWithTag[T](tag)` — same, but resolves field
  names/paths from `tag` (use `"input"` for input DTOs). An optional
  `omitSystemFields=true` skips the embedded `DocumentModel`.

Export one accessor per DTO in the schema package:

```go
func UserOutputSchema() *definition.Schema      { return dispatch.SchemaFromType[schema.UserOutput]() }
func UserGetInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[UserGetInput]("input") }
func UserUpdateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[UserUpdateInput]("input", true) }
```

Wire them into the feature registration (see `users/feature.go`):

```go
Input:  runtime.Input{ Schema: schema.UserUpdateInputSchema(),
    Arguments: []abstract.ArgDef{{Name: "user_id", ...}}, Payload: definition.FieldTypeObject },
Output: schema.UserOutputSchema(),
```

> Use `"input"` tags for anything bound from the incoming message, `anansi`
> tags for outputs. This mirrors the two-tag contract from §2 and guarantees
> the runtime schema, the struct, and the wire format stay in sync.

---

## Checklist

When adding a new persistence-backed feature:

1. Write `<feature>/schema/<name>.schema.json` (stable UUID keys, required/nullable correct).
2. `anansi codegen golang` to emit the model + projections.
3. Add `input` tags to projections that arrive via messages; add a password-free output projection.
4. `anansi migrate generate --dry-run`, then `anansi migrate generate`.
5. Write DTOs + `*Schema()` accessors in the schema package.
6. Write `*_utils.go` domain methods on the model (typed updates only).
7. Write handlers that `BindToTag(&dto, "input")` and `UpdateFrom`/`CreateFrom`.
8. Register messages in `feature.go` with the derived schemas.
9. Build, vet, and run the feature tests.
