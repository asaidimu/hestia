---
name: hestia
description: Build applications on hestia, the embedded Go application framework (auth, authorization, schema-driven persistence, blobs, notifications, scheduling, audit — one binary, no external services). Use this whenever the user is working on a Go project that imports `github.com/asaidimu/hestia/core`, calls `hestia.Setup`, wires `SetupConfig.Modules`, writes a module/feature with `abstract.MessageRegistration`, names messages `module:feature:scope:action`, binds DTOs via `BindToTag`/projections, or asks to add persistence, auth, an API endpoint, or a collection to such an app. Also use it when the user is inside the hestia repo itself and wants to add/extend a built-in feature, understand message routing, bootstrap, or the dispatcher chain. If the task is about the underlying go-anansi persistence layer (schema.json, codegen, migrations, ModelCollection) reach for the `anansi` skill instead — but reach for THIS skill first to understand where the feature fits. If the task is *inside* go-anansi's own source, that's a contributing task — neither skill applies.
---

# Building applications on hestia

hestia is an **embedded application framework for Go**. It turns a message-routing
kernel into a full API server: auth, authorization, schema-driven persistence,
blobs, notifications, schedules, audit — all in one binary. You declare your app
as a set of **modules** and **features**; the framework derives HTTP routes,
permission scopes, and SDK methods mechanically from message names. There are no
hand-registered routes.

This skill is about **building applications on hestia** — going from an idea to
a working `Setup` with modules and features. It is not about modifying hestia's
internals (dispatchers, the http interface, runtime), though it helps you
navigate them.

The mental model that drives every step: **your app is a set of named messages.
Everything else — routes, permissions, SDK methods, HTTP plumbing — is derived.**
Resist thinking in terms of "endpoints." Think in terms of messages, and the
framework does the rest.

> **hestia is under heavy, daily development.** APIs break between releases.
> Prefer patterns copied from working code in the repo over the README, which is
> a snapshot. When in doubt, check `core/abstract/`, `core/hestia.go`, and the
> most recently written built-in features (`users`, `apikeys`).

**Read the bundled references before starting** — they are distilled from the
source and are the primary input. Open repo files only to *verify* a specific
API that changed, not to discover patterns:

| When you are... | Read first |
|---|---|
| Adding/rewriting a feature | `references/feature-template.md` (copyable skeleton) |
| Writing a module / wiring an app | `references/module-template.md` |
| Deciding whether hestia fits | `references/platform-guide.md` (verified Q&A) |
| Doing schema/migration/codegen work | the `anansi` skill + the `core/system/users` reference service |

---

## The development loop

A typical app flows through these stages. Do them in this order; jump in at
whatever stage matches what the user is doing.

1. **Scaffold** the app: `hestia init` (CLI) or hand-write a `main.go` calling
   `hestia.Setup`. See `examples/basic/` for a minimal server.
2. **Wire configuration** — env vars or `SetupConfig`. Get a server up with the
   built-in features before adding your own.
3. **Understand the message contract** — how names map to routes and verbs.
4. **Add a feature** — schema + projections, codegen, DTOs, handlers,
   registrations. Follow the canonical layout.
5. **Wire modules** into `SetupConfig.Modules`.
6. **Test** against the running server (or via `dispatch.NewMessage` unit tests).
7. **Evolve** — when schema changes, follow the anansi migration workflow.

---

## 1. Scaffold and start

Minimal server (`examples/basic/main.go`):

```go
package main

import (
	"fmt"
	"os"

	hestia "github.com/asaidimu/hestia/core"
)

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		Version:       version,
		SessionSecret: "my-test-secret",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}
	if err := app.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Start failed: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()
	select {}
}
```

`Setup` does: load config → apply migrations → hydrate the built-in features
(auth, users, apikeys, policies, blobs, collections, operations, notifications,
schedules, settings, tenants) → register your modules → build interfaces. It
returns an `*Application` you can start.

The **bootstrap flow** matters on first run: hestia seeds an admin user with
random credentials and only exposes two routes until the admin sets a password.
For development, pass `AdminEmail`/`AdminPassword` in `SetupConfig` (or the
equivalent env vars) so the system is usable immediately.

---

## 2. What you get by default

`Setup` with no extra wiring gives you a fully-working API server with every
built-in feature hydrated and authenticated:

| Feature | Message scope | What it gives you |
|---|---|---|
| auth | `system:auth:*` | Sessions (cookie), JWT tokens, password auth, bootstrap |
| users | `system:users:*` | User CRUD + registration |
| apikeys | `system:apikeys:*` | API key management with rotation |
| policies | `system:policies:*` | CEL-based operation policies, rate limits, throttling |
| blobs | `system:blobs:*` | Namespaced file storage (incl. resumable uploads) |
| collections | `system:collections:*` | Schema-less document CRUD (built-in dynamic routes) |
| operations | `system:operations:*` | Operation metadata & policy bindings |
| audit | `system:audit:*` | Access + audit logging |
| notifications | `system:notifications:*` | Mailer-backed notifications |
| schedules | `system:schedules:*` | Cron-based scheduling |
| settings | `system:settings:*` | Runtime settings |
| tenants | `system:tenants:*` | Multi-tenant isolation |

So a base app already has identity, authorization, persistence, storage,
notifications, and scheduling before you write any of your own code.

**Interfaces created by default:** `Setup` builds an HTTP interface
(fasthttp, on `:8090`, prefix `/api`) **and** a CLI interface (`--version`,
`--help`, `--bootstrap`) automatically. Override with `BuildInterfaces`.

**How requests authenticate** (HTTP auth middleware, in order):
1. **Session cookie** (`session` by default) — validated, idle/absolute TTL
   checked, sliding-window refreshed. An `X-Api-Key` header alongside elevates
   the session (recorded for audit as on-behalf-of).
2. **API key** alone (`X-Api-Key` / `X-API-Key`) — validated via the internal
   `system:auth:apikey:validate` message.
3. **Anonymous** — if nothing is supplied, an empty identity flows through and
   the policy engine decides (most routes deny anonymous).

---

## 3. Configuration

`Setup` reads env vars (including `.env`/`.env.dev` via godotenv), then
`SetupConfig` fields override them. Key knobs:

| Concern | SetupConfig field | Env var |
|---|---|---|
| Signing key (required) | `SessionSecret` | `SESSION_SECRET` |
| Port / API prefix | `Port`, `APIPrefix` | `PORT`, `API_PREFIX` |
| State | `DataDir`, `DBPath`, `BlobsDir` | `APP_DATA_DIR`, `DB_PATH`, `BLOBS_DIR` |
| Seed admin | `AdminEmail`, `AdminPassword` | `ADMIN_EMAIL`, `ADMIN_PASSWORD` |
| Sessions | `SessionTTL`, `IdleTTL`, `RefreshTTL` | `SESSION_TTL`, ... |
| Skip bootstrap | `ForceBootstrapped` | `FORCE_BOOTSTRAPPED` |
| Mailer | `Mailer` (SMTP config) | `SMTP_*` |
| Extensions | `Modules`, `DispatcherChainFunc`, `BuildInterfaces` | — |

The full surface is `SetupConfig` in `core/hestia.go`. For a dev server with
auth bypassed, see `cmd/test-server/`.

### The extension-point map

These are the levers for adding or replacing behavior. Each is a field on
`SetupConfig` unless noted:

| Extension point | What it does |
|---|---|
| `Modules` | Add your own features (see §6) — the primary way to extend |
| `Migrate` | Run your own post-bootstrap migrations against `base.Persistence` |
| `PersistenceFactory` | Swap the storage backend entirely (e.g. Postgres) — return a `base.Persistence` |
| `DispatcherChainFunc` | Insert custom links into the dispatcher chain |
| `BuildInterfaces` | Replace the default HTTP+CLI interfaces; `Interfaces` appends extra ones |
| `OnBootstrapped` / `OnReset` | Callbacks fired when the system bootstraps or resets |
| `Mailer`, `AppURL` | Wire SMTP and public URLs for notifications/emails |
| `RegisterModules` (on `*Application`) | Add modules after `Setup`, before `Start` |

Example — custom migrations + a post-setup module:

```go
app, err := hestia.Setup(hestia.SetupConfig{
	SessionSecret: "s3cret",
	Migrate: func(ctx context.Context, p base.Persistence) error {
		return mydata.Migrate(ctx, p) // your schema/data migrations
	},
	PersistenceFactory: func(cfg *anansi.SetupConfig) (base.Persistence, error) {
		interactor, _, _ := myPostgresInteractor(cfg.Logger)
		cfg.Interactor = interactor
		cfg.EventBus = events.NewSimple[...](...)
		return anansi.Setup(*cfg)
	},
})
// register after Setup if you didn't pass Modules up front
if err := app.RegisterModules(mymodule.New()); err != nil { ... }
```

Middleware and static files are per-HTTP-interface, not on `SetupConfig` —
pass them when you build the interface via `BuildInterfaces`:

```go
app.NewHTTPInterface(httpapi.Config{
	Port: 8090,
	Middleware: []httpapi.Middleware{myMW},
	StaticFS:   embeded.FS, // serve a frontend from the binary
})
```

---

## 4. The message contract (the core idea)

A message name is a colon-delimited quadruple:

```
module:feature:scope:action
```

| Message | HTTP |
|---|---|
| `system:auth:session:create` | `POST /api/auth/session` |
| `system:users:user:get` | `GET /api/users/user/{user_id}` |
| `system:apikeys:key:update` | `PATCH /api/apikeys/key/{key_id}` |
| `collection:articles:document:read` | `GET /api/collection/articles/document/{doc_id}` |

Derivation rules (from `core/interface/http/derive.go`):
- **Path** = `/<module>/<feature>/<scope>/<action>` with each declared `Argument`
  appended as `/{arg}`.
- **HTTP method** comes from the `Intent` verb: `Create`→POST, `Read`→GET,
  `Update`→PATCH, `Delete`→DELETE, `Query`→POST, `Stream`→GET, `Check`→POST.

You register messages with `abstract.MessageRegistration`:

```go
{ Name: "mymodule:greeter:hello:create",
  Handler: newHandler(deps),
  Description: "Say hello",
  Intent: abstract.Create,
  Input: runtime.Input{
    Schema:    helloInputSchema(),
    Payload:   definition.FieldTypeObject,
  },
  Output: helloOutputSchema() }
```

Flags on a registration:
- `BootstrapSafe: true` — route stays available before bootstrap completes.
- `Internal: true` — never exposed over HTTP; dispatch-only.
- `Enabled: true` — must be set for the message to be served.

---

## 5. The Module interface

Modules are how you bundle features into an app. Register them via
`SetupConfig.Modules`. The interface (`core/abstract/module.go`):

```go
type Module interface {
	Name() string
	Setup(ctx context.Context, rt abstract.Container) error
	Capabilities(rt abstract.Container) ([]Capability, error)
	Dependencies() []string        // modules that must init first
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) any
}
```

Boot runs a **two-phase** setup: every module's `Setup(ctx, rt)` registers its
service providers into the shared runtime container (seeded with persistence,
logger, dispatcher), then the container is sealed with `Rebuild`, then each
module's `Capabilities(rt)` resolves its services and returns message
registrations. Embed `module.BaseModule` for the no-op `Dependencies`/`Start`/
`Stop`/`Health`.

A module wraps one or more features. Each feature is a package that registers a
service into the container (`Register(rt)`) and returns
`[]abstract.MessageRegistration` from a `Registrations(rt)` function plus a
`Policies()` (if it has IAM rules). The canonical layout (same shape inside the
built-in system module and inside user modules):

```
core/system/<feature>/
├── model/
│   ├── <name>.schema.json        # collection schema + projections
│   ├── <name>.schema.model.go    # GENERATED — never hand-edit
│   ├── data_transfer_objects.go  # input/output DTOs + schema builders
│   └── <feature>_utils.go        # domain methods on the generated model
├── service.go                    # the service struct + New<Feature>Service(rt)
├── handler.go                    # message handler methods (annotate @hestia.register)
├── feature.go                    # Register(rt) + Registrations(rt) → []MessageRegistration
├── provider.go                   # policy bindings (optional)
├── handler_test.go               # handler tests
└── model/<name>_test.go          # domain method tests
```

The full shape of every file is in `references/feature-template.md`.

For a copyable module skeleton (all methods, wiring, and the rules that go with
them), use `references/module-template.md`. The canonical in-repo examples are
`core/system/users/` and `core/system/apikeys/` — read those only to verify a
field name or API that the templates don't pin down.

---

## 6. Building a feature

The built-in features follow the exact same pattern as user modules — there is
no privileged path. **Start from `references/feature-template.md`** and copy
each file's shape; the checklist below is the workflow. The full upstream
narrative lives in the `core/system/users` reference service (read it for context, not
as a substitute for the template).

1. **Write the schema** `model/<name>.schema.json` with projections. Never
   hand-generate field UUIDs — put the field name as the key; codegen normalizes
   it. Field shape: required+non-nullable → value type; optional/nullable →
   pointer type. Declare `metadata.projections` (Input with `input:` tags,
   secret-free Output).
2. **Run the anansi codegen workflow** (see the `anansi` skill):
   `anansi migrate generate --dry-run` → `anansi migrate generate` →
   `anansi codegen golang`. Check `git diff --stat` for unrelated regeneration
   — `codegen golang` regenerates ALL features.
3. **Write DTOs** in `model/data_transfer_objects.go`. Input DTOs ARE the
   projections — bind them directly, no wrapper structs:
   ```go
   var input model.APIKeyCreate
   if err := msg.Input().BindToTag(&input, "input"); err != nil {
       return nil, err
   }
   ```
   Schema accessor: `dispatch.SchemaFromTypeWithTag[T]("input", true)`.
   Argument-only inputs get a small struct with `input:"arguments.{name}"` tags.
   Output DTOs are plain envelopes with a top-level `document`/`documents`/`page`
   field.
4. **Domain methods** in `model/<feature>_utils.go`, as methods on the generated
   collection. Route creates/updates through `document.New` / `UpdateFrom` —
   never map-based updates. Name them `CreateKey`/`UpdateKey` (not `Create`) to
   avoid shadowing the ModelCollection. Ownership-scope reads with the query
   DSL.
5. **Handler** (`handler.go`) — message handlers as methods on the service
   struct. Each handler is annotated with `@hestia.register(name, intent, rule,
   description, resource_id)`; `hestia service generate` binds it to a
   `dispatch.Handle*` adapter:
   ```go
   // @hestia.register(
   //   name="system:widgets:widget:create",
   //   intent="create",
   //   rule="administrator",
   //   description="Create a widget",
   // )
   func (s *WidgetsService) CreateWidget(ctx context.Context, msg abstract.Message, input *model.WidgetCreate) (*model.Widget, error) {
       return s.model.Create(ctx, input) // model collection method
   }
   ```
   `New<Feature>Service(rt abstract.Container)` (in `service.go`) resolves
   persistence and the feature's model/provider from the container.
6. **Registrations** (`feature.go`) — the generated `registrations.go` is
   produced by `hestia service generate` from the annotations; by hand it is
   `Register(rt)` + `Registrations(rt)` over `abstract.Container`.
7. **Wire it**: init the model in `core/system/module.go` (the `SystemModule`
   seeds the system providers into the shared container, then calls
   `RegisterServices`) and add `@hestia.register` annotations on the service
   methods so `hestia service generate` emits `registrations.go`/`policies.go`.
8. **Migrate/rewrite tests** — delete scratch tests; port the feature's old
   tests to the generated model (`DangerouslyReset<Model>()` +
   `Init<Model>(testutil.NewPersistence(t), ...)`).
9. **Verify** — `go build ./... && go test ./core/system/<feature>/...`, and
   re-check `git diff --stat` for unrelated regeneration.

### Policies (`policies.go`)

Each feature that has IAM rules declares them via `PolicyBindings()`. This maps
message names to default rule keys so the permission manager knows what a
request to that message must satisfy. The typical shape (see
`core/system/apikeys/policies.go` — it's the canonical example):

```go
package apikeys

import "github.com/asaidimu/hestia/core/system/policies"

func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:apikeys:key:create", RuleKey: "administrator", Description: "Create a new API key"},
		{Name: "system:apikeys:key:update", RuleKey: "administrator", Description: "Update API key metadata"},
	}
}
```

- `policies.Binding` (`core/system/policies/model.go`) has `Name`,
  `Description`, and `RuleKey`. `RuleKey` is the **default** rule name used for
  seeding — it references a CEL rule in the `iam` rule store (built-in defaults
  come from `GoDefaultRules` in `core/system/policies/`).
- Bindings are **seeded** so they persist — the system module calls
  `SeedPolicies`/`EnsureBinding` (see `core/system/module.go` and
  `core/system/policies/seed.go`).
- If your feature has no rules, it's fine to omit the file.

### Reaching built-in features from your module

Your module's `Setup(ctx, rt)` receives the shared runtime container, pre-seeded
with the base providers (persistence, logger, dispatcher). Resolve the typed
built-in models/services from it via `abstract.MustResolve[T](rt)`, or register
your own providers with `abstract.Register`/`abstract.RegisterInstance`:

```go
type Module struct {
	module.BaseModule
}

func (m *Module) Setup(ctx context.Context, rt abstract.Container) error {
	persist := abstract.MustResolve[base.Persistence](rt)
	users := abstract.MustResolve[*usersmodel.SystemUsers](rt)
	// register your own providers, init models against shared persistence, ...
	return myfeature.Register(rt)
}
```

- The canonical wiring for the built-in features is `core/system/module.go`
  (the `SystemModule` — it initializes every model once at startup in
  `Setup`, seeds the system providers into the shared container, then calls
  `RegisterServices`). Mirror that pattern for your own modules: init your
  models in `Setup`, register them as providers, resolve them in
  `Capabilities`/`Registrations`.
- For system-wide state that you need from outside the module (e.g. the admin
  user ID), access it via `app.SystemModule()` — it exposes `UserModel()`,
  `CredentialsProvider()`, `AdminUserID()`, `AdminEmail()`, `Bootstrapped()`.
- Cross-feature consumers within the hestia repo were converted alongside the
  generated models — when `apikeys` moved to generated code, `auth` updated its
  call sites (see §8 of the `core/system/users` reference service).

### Result helpers

Never populate `Result` fields directly — use the dispatch constructors:

```go
dispatch.NewDocumentResult(doc)
dispatch.NewDocumentsResult(docs)
dispatch.NewPageResult(page)
dispatch.NewBlobResult(blob)
dispatch.NewDocumentChannelResult(ch)
dispatch.NewBlobChannelResult(ch)
```

`Result` pools underlying resources. Call `result.Release()` once you've
consumed a result (the HTTP interface does this automatically for responses).

### Custom response shapes

You can't `doc.Set()` an arbitrary field onto a schema-bound document. To return
a field that isn't in the collection (e.g. a once-only secret), build a custom
output DTO that embeds the public projection and declares the extra field, then
populate it by binding the model into the projection and re-marshaling. The
`APIKeyCreatedOutput` pattern is inlined in `references/feature-template.md`
§2.

---

## 7. The dispatcher chain and permissions

Every message flows through a **dispatcher chain** layered by
`SystemModule.DispatcherChain`:

```
bootstrap → secure → ratelimit → throttle → tenant → blob → recovery → audit
→ LocalDispatcher
```

You can insert your own links via `SetupConfig.DispatcherChainFunc` (operates on
`abstract.ChainEditor`), or replace interfaces via `BuildInterfaces`.

`SecureDispatcher` enforces authorization on every message:
1. `PermissionManager.Resolve(msg)` maps message name → rule key.
2. `AccessController.Can(ctx, ruleKey, resource)` evaluates the rule (CEL,
   via go-iam).
3. Unauthorized → `403 ERR_ACCESS_DENIED`.

Features declare their rules in `policies.go` as `PolicyBindings()`. Bootstrap,
password reset, and token validation are system-internal and bypass the security
layer.

---

## 8. Testing

Two levels:

**Against a live server** (`cmd/test-server`, port 8070, auto-reloads):
1. Establish a session: `POST /api/system/session` with
   `{"email": "admin@test.local", "password": "password123"}`.
2. Discover every registered command + docs:
   `GET /api/system/core/docs/list`.

**Unit tests** — use the test harness patterns in `references/feature-template.md`
§7 (they're distilled from `core/system/apikeys/handler_test.go` and
`core/system/users/model/system_user_test.go`):
- The generated model singleton pins the first persistence, so tests must call
  `DangerouslyResetXxxModel()` first, or construct fresh.
- Build message inputs with `data.MustNewDocument(map[string]any{...})` or
  `testutil.InputDoc(t, <InputSchema>(), \`{"arguments": {...}}\`)`.
- For user-driven handlers, attach claims via
  `runtimecontext.ContextWithClaims(ctx, &abstract.Claims{UserID: ...})`.
- Verify build + tests: `go build ./... && go test ./...`.

---

## 9. Client SDK and CLI

**TypeScript SDK** (`@asaidimu/hestia`, source in `client/`) mirrors the Go API
over HTTP with reactive stores and auto-refresh JWT handling:

```ts
import { HestiaClient } from "@asaidimu/hestia"

const api = new HestiaClient({ baseUrl: "http://localhost:8090" })
await api.auth.login("admin@test.local", "password123")
const { data: users } = await api.users.find()
const docs = api.collection<MyType>("articles")
await docs.create({ title: "Hello" })
```

**CLI tool** (`cmd/hestia`):
```bash
hestia init                        # scaffold a new project
hestia generate modules            # regenerate the module registry
hestia module <name> [feature]     # scaffold a module + optional feature
hestia add cmd <name>              # new server binary
hestia remove module <name>
```

---

## 10. Platform decision guide

Before betting a product on hestia, read `references/platform-guide.md` — it
answers the 101 due-diligence questions from `todo/dev-questions.md` against
the actual code. The short version:

**The blocking verdicts:**
- **One app per process.** `boot.ProjectName` and generated model singletons
  are package-global — no two hestia apps in the same binary.
- **Authorization is a single chain link.** `SecureDispatcher` is the only
  gate. Inserting a custom link *before* `secure` (legal via
  `DispatcherChainFunc`) bypasses authz entirely. Don't.
- **`Internal: true` is a routing flag, not security.** It hides a message from
  HTTP but it stays dispatchable in-process.
- **Denied requests are not audited.** The audit link is innermost, so authz
  denials short-circuit before audit. Compliance gap.
- **Sessions are HMAC-SHA256, not JWT** (README is wrong). Revocable via
  `token_version`; stateless, no server-side store.
- **System scope = root.** Any identity holding a `system:*` permission skips
  `SecureDispatcher` (`IsSystemIdentity`). Never grant it casually.
- **No horizontal scaling.** Single-process embedded SQLite + in-process
  dispatcher. Fine for one server; not a distributed platform.
- **No upgrade contract.** Alpha, forward-only migrations, no rollback.

**The good news:**
- Unbound messages default to `administrator` (secure-by-default).
- Policy changes apply instantly (`CacheTTL: 0`, LiveRepository).
- Custom decorators = chain links; custom interfaces = `runtime.Interface`;
  backend swap = `PersistenceFactory`.
- MIT + MIT/Apache-2.0 deps; no copyleft blockers.

---

## 11. The invariants — rules that never change

These are hard rules, not style. Violating any of them produces a build break,
a security hole, or a migration catastrophe.

1. **Never hand-edit generated files.** `*.schema.model.go` and anything under
   a `model/` directory marked generated are regenerated by `anansi codegen
   golang`. Your edits get silently overwritten.
2. **Never re-key schema field UUIDs.** Add new fields with
   `"id": "<field-name>"`; delete fields by removing their entry. Renumbering an
   existing field's ID corrupts migration history.
3. **Never shadow `Create`/`Update`/`Read` on the generated collection.** Name
   domain methods `CreateWidget`/`UpdateWidget`, or you hide the
   ModelCollection methods and break `UpdateFrom`.
4. **Never use map-based updates.** Creates go through `document.New`,
   updates through `UpdateFrom`. Map updates bypass projection validation.
5. **Never insert a dispatcher chain link before `secure`.** That link would
   run before authorization and bypass `SecureDispatcher` for every message
   after it. Insert after `secure`, or accept the exposure.
6. **Never grant `system:*` casually.** `IsSystemIdentity` skips
   `SecureDispatcher` — it is root.
7. **Never expose the collection's secret fields.** Output projections must
   exclude hashes/secrets; reveal raw material only via a custom output DTO
   (one-shot create/rotate response), never in list/get projections.
8. **Never build your own route table.** Routes are derived from message
   names. Hand-registered HTTP paths drift out of sync with the SDK and policy
   engine.
9. **Don't stack two apps in one process.** `boot.ProjectName` and the model
   singletons are global. One hestia app per binary, per process.
10. **Don't trust a caller-supplied `user_id` over claims.** In handlers,
    `runtimecontext.ClaimsFromContext` wins; a bound `user_id` argument is only
    a fallback for admin/system callers.

---

## Reference map

Bundled references (read these first):
- `references/feature-template.md` — copyable feature skeleton: schema, DTOs,
  domain methods, handler, registrations, policies, tests.
- `references/module-template.md` — copyable module skeleton + app wiring.
- `references/platform-guide.md` — verified answers to the 101 platform
  due-diligence questions.

Repo pointers (verification only, not discovery):
- `core/hestia.go` — public API: `Setup`, `SetupConfig`, `Application`.
- `core/abstract/` — interfaces & envelope types (message, module, result,
  registration). Zero implementation — the source of truth for shapes.
- `core/runtime/dispatch/` — message/result constructors, `SchemaFromTypeWithTag`.
- `core/interface/http/derive.go` — route + method derivation rules.
- `core/system/users/`, `core/system/apikeys/` — canonical features.
- the `core/system/users` reference service — upstream feature-writing narrative.
- `examples/basic/` — minimal standalone server.
- Schema/persistence details — the `anansi` skill, or `docs/guide/*.md`.

**Do not go digging through hestia's runtime internals to rediscover patterns**
— the docs, the reference features, and the `anansi` skill already encode them.
When a pattern seems hard, it's usually a signal you're working against the
grain: reconsider the message/feature decomposition rather than hacking the
chain.