# hestia

**hestia** is an embedded application framework for Go that turns a message-routing kernel into a full-featured API server. It provides authentication, authorization, schema-driven persistence, blob storage, notifications, scheduling, audit logging, and a modular feature system — all in a single binary with no external dependencies. It ships with a typed TypeScript client SDK (`@asaidimu/hestia`).

> **⚠️ Alpha — active development.** hestia is under heavy, daily development. APIs, schemas, configuration, and tooling are changing rapidly and may break between releases. Expect frequent breaking changes; pin versions and treat current docs as snapshots. Not yet recommended for production use.

## Quick Start

```go
package main

import (
	"os"

	hestia "github.com/asaidimu/hestia/core"
)

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		ProjectName:   "myapp",
		SessionSecret: "change-me",
	})
	if err != nil {
		panic(err)
	}
	if err := app.Start(); err != nil {
		panic(err)
	}
	defer app.Close()

	os.Stdout.Sync()
	select {}
}
```

```bash
go run main.go
```

`Setup` loads configuration from the environment (or `.env`), applies migrations, hydrates the built-in services (auth, users, API keys, policies, blobs, audit, collections, operations, notifications, schedules, settings, tenants), and serves on `:8090` with an auto-documented HTTP API plus a CLI interface.

See `examples/basic/` for a minimal standalone server and `cmd/test-server/` for a wired example.

## Install

```bash
go get github.com/asaidimu/hestia
```

CLI tool (project scaffolding & codegen):

```bash
go install github.com/asaidimu/hestia/cmd/hestia@latest
```

TypeScript client SDK:

```bash
npm install @asaidimu/hestia
# or
bun add @asaidimu/hestia
```

---

## Configuration

`Setup` reads environment variables (also loading `.env` and `.env.dev` if present via godotenv). Settings applied directly in `SetupConfig` override env vars.

### Environment Variables

| Var | Default | Description |
|---|---|---|
| `SESSION_SECRET` | — | HMAC-SHA256 signing key (required, or set via `SetupConfig.SessionSecret`) |
| `PORT` | `8090` | HTTP port |
| `API_PREFIX` | `/api` | URL prefix for the HTTP API |
| `APP_DATA_DIR` | `~/.local/share/<project>` | All persistent state (DB, logs, blobs) |
| `DB_PATH` | `<data_dir>/<project>.db` | SQLite file path; `:memory:` for in-memory |
| `BLOBS_DIR` | `<data_dir>/blobs` | Blob file storage root |
| `BCRYPT_COST` | `12` | Password hashing cost (4–31) |
| `SESSION_TTL` | `8h` | Absolute session lifetime |
| `SESSION_IDLE_TTL` | `30m` | Session idle timeout |
| `SESSION_REFRESH_TTL` | `15m` | Sliding-window cookie refresh threshold |
| `COOKIE_DOMAIN` | `""` | Cookie domain restriction |
| `COOKIE_SECURE` | `true` | Require HTTPS for cookies |
| `COOKIE_SAMESITE` | `strict` | `strict`, `lax`, or `none` |
| `SESSION_COOKIE_NAME` | `session` | Session cookie name |
| `SESSION_COOKIE_PATH` | `/` | Session cookie path |
| `ALLOWED_ORIGINS` | `""` | Comma-separated CORS origins |
| `FORCE_BOOTSTRAPPED` | `false` | Skip the bootstrap flow (treat system as already bootstrapped) |
| `LOG_PATH` | `<data_dir>/server.log` | Log file path |
| `LOG_MAX_SIZE` | `100` | Log rotation size (MB) |
| `LOG_MAX_AGE` | `30` | Log retention (days) |
| `LOG_MAX_BACKUPS` | `5` | Max old log files kept |
| `APP_URL` | `""` | Public app URL (used in emails/links) |
| `SMTP_HOST` | — | Outbound mail (notifications) |
| `SMTP_PORT` | — | SMTP port |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | — | SMTP credentials |
| `SMTP_FROM` / `SMTP_FROM_NAME` | — | Sender identity |
| `SMTP_AUTH_TYPE` | — | SMTP auth type |

### Programmatic Config

`SetupConfig` (in `github.com/asaidimu/hestia/core`) exposes every setting:

```go
hestia.Setup(hestia.SetupConfig{
	ProjectName:      "myapp",
	SessionSecret:    "change-me",           // required if not in env
	DataDir:          "./data",
	DBPath:           "./data/app.db",
	Port:             8090,                  // BcryptCost, SessionTTL, IdleTTL, RefreshTTL, ...
	AdminEmail:       "admin@example.com",   // override generated seed admin
	AdminPassword:    "secret",
	ForceBootstrapped: true,
	Version:          "1.0.0",

	Modules:   []hestia.Module{myModule.New()}, // register extension modules

	// Hook into the dispatcher chain (after default links) or replace interfaces
	DispatcherChainFunc: func(chain abstract.ChainEditor) { chain.Add(...) },
	BuildInterfaces:     func(app *hestia.Application) []runtime.Interface { return ... },

	Migrate:            func(ctx, p base.Persistence) error { ... }, // post-bootstrap migrations
	PersistenceFactory: func(cfg *anansi.SetupConfig) (base.Persistence, error) { ... },
})
```

---

## Setup API

```go
// Build the application: load config, migrate, hydrate features, build interfaces.
func Setup(cfg SetupConfig) (*Application, error)

// Application handles (methods on the returned value):
app.Persistence()   base.Persistence                           // underlying persistence
app.Dispatcher()    abstract.Dispatcher                        // core local dispatcher
app.SystemModule()  SystemModule                               // built-in system module
app.Registrations() []abstract.MessageRegistration
app.RegisterModules(m ...Module) error
app.NewHTTPInterface(cfg httpapi.Config) runtime.Interface      // fasthttp server
app.NewCLIInterface(cfg cli.Config) runtime.Interface           // version/help/bootstrap CLI
app.Start() error
app.Shutdown(ctx context.Context) error
app.Close()
```

`Version` is printed by the CLI `--version` flag and the startup banner.

---

## Features

Built-in features live under `core/system/<feature>/` (one directory per feature) and are wired together by the system module under `core/system/` (generated wiring in `gen_features.go`). Each feature exposes a message contract (`module:feature:scope:action`), input/output schemas, and policy bindings.

| Feature | Message scope | Purpose |
|---|---|---|
| auth | `system:auth:*` | Sessions, JWT tokens, password auth, bootstrap |
| users | `system:users:*` | User CRUD + registration |
| apikeys | `system:apikeys:*` | API key management with rotation |
| policies | `system:policies:*` | CEL-based operation policies, rate limits, throttling |
| blobs | `system:blobs:*` | Namespaced file storage (incl. resumable uploads) |
| collections | `system:collections:*` | Schema-less document CRUD |
| operations | `system:operations:*` | Operation metadata & policy bindings |
| audit | `system:audit:*` | Access + audit logging |
| notifications | `system:notifications:*` | Mailer-backed notifications |
| schedules | `system:schedules:*` | Cron-based scheduling |
| settings | `system:settings:*` | Runtime settings |
| tenants | `system:tenants:*` | Multi-tenant isolation |

---

## Persistence

hestia uses **[go-anansi/v8](https://github.com/asaidimu/go-anansi)** as its document store. Schemas are declared as plain `.schema.json` files under `core/system/<feature>/model/`, and models/DTOs are generated from them.

The default backend is **SQLite** (via `mattn/go-sqlite3`, WAL mode), embedded in-process — no separate database process to manage. A **Pebble-backed event bus** powers change notifications.

### Schema workflow

1. Edit `model/*.schema.json` (never rename existing IDs)
2. `anansi migrate generate --dry-run` to validate
3. `anansi migrate generate` to create the migration
4. `anansi codegen golang` to regenerate model structs & DTOs

See `core/system/users/` for the canonical feature layout (schema + projections, codegen, DTOs, domain methods, wiring, tests) and `todo/migrate_features.md` for the migration plan.

### Custom Backend

The persistence layer is database-agnostic. Everything goes through `query.DatabaseInteractor`:

```go
type DatabaseInteractor interface {
	SchemaManager
	SelectDocuments(ctx, schema, query) ([]map[string]any, int64, error)
	SelectStream(ctx, schema, query) (<-chan map[string]any, <-chan error, error)
	InsertDocuments(ctx, schema, records) ([]map[string]any, error)
	UpdateDocuments(ctx, schema, updates, filters, returning) ([]map[string]any, int64, error)
	DeleteDocuments(ctx, schema, filters, unsafeDelete) (int64, error)
	Query(ctx, query) (*RawQueryResult, error)
	StartTransaction(ctx) (DatabaseInteractor, error)
	Commit(ctx) error
	Rollback(ctx) error
	Capabilities() Capabilities
}
```

`SetupConfig.PersistenceFactory` gives full control over the persistence setup:

```go
hestia.Setup(hestia.SetupConfig{
	PersistenceFactory: func(cfg *anansi.SetupConfig) (base.Persistence, error) {
		interactor, _, _ := myPostgresInteractor(cfg.Logger)
		cfg.Interactor = interactor
		cfg.EventBus = events.NewSimple[...](...)
		return anansi.Setup(*cfg)
	},
})
```

go-anansi currently ships only SQLite adapters; a custom interactor (Postgres, etc.) plugs straight in and all hestia features work unchanged.

---

## Request Lifecycle

```
HTTP ─▶ fasthttp transport ─▶ auth middleware ─▶ route closure ─▶ Dispatcher Chain ─▶ handler
                                                                   │
                                        ┌──────────────────────────┴──────────────────────┐
                                        │ bootstrap → secure → ratelimit → throttle       │
                                        │ → tenant → blob → recovery → audit              │
                                        │ → LocalDispatcher                               │
                                        └─────────────────────────────────────────────────┘
```

1. **Auth middleware** tries: session cookie → API key → anonymous
2. **Route closure** builds `{arguments, modifiers, payload}` from the HTTP request (APIs auto-derived from message names)
3. **Dispatcher chain** layers bootstrap gating, authorization, rate limiting, throttling, tenant scoping, blob routing, panic recovery, and audit logging
4. **Handler** executes business logic, returns a `*Result` with a `Kind` discriminant
5. **Serializer** converts the result to an HTTP response (sanitized once)

---

## Core Concepts

### Message

```go
type Message interface {
	ID() string
	Name() string
	Context() context.Context
	Input() *data.Document
}
```

Named envelopes routed through the chain. Names are colon-delimited quadruples — HTTP routes, permission scopes, and SDK method names are mechanically derived — no manual route registration:

```
module:feature:scope:action
```

| Message | HTTP |
|---|---|
| `system:auth:session:create` | `POST /api/system/auth/session/create` |
| `system:users:user:get` | `GET /api/system/users/user/get/{user_id}` |
| `system:auth:session:delete` | `DELETE /api/system/auth/session/delete` |
| `system:collections:document:get` | `GET /api/system/collections/document/get/{name}/{doc_id}` |

### Result

```go
type Result struct {
	Kind            ResultKind          // explicit discriminant
	Document        *document.Document
	Documents       document.DocumentSet
	Page            *Page
	Blob            Blob
	DocumentChannel <-chan *document.Document
	BlobChannel     <-chan Blob
}
```

Construct via helpers — never populate `Result` fields directly:

```go
dispatch.NewDocumentResult(doc)
dispatch.NewDocumentsResult(docs)
dispatch.NewPageResult(page)
dispatch.NewBlobResult(blob)
dispatch.NewDocumentChannelResult(ch)
```

Results pool underlying resources. Call `result.Release()` once you've consumed a result (the HTTP interface does this automatically) to return pooled documents back to the pool.

### Module

```go
type Module interface {
	Name() string
	// Setup registers the module's service providers into the shared runtime
	// container rt. Boot pre-populates rt with base providers (persistence,
	// logger, dispatcher); modules resolve them via abstract.MustResolve[T](rt).
	Setup(ctx context.Context, rt abstract.Container) error
	// Capabilities is called after every module has been set up and the shared
	// container has been sealed (Rebuild). Resolve services from rt and return
	// the message registrations they expose.
	Capabilities(rt abstract.Container) ([]Capability, error)
	Dependencies() []string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) any
}
```

Modules embed `module.BaseModule` (from `github.com/asaidimu/hestia/core/runtime/module`) for the no-op `Dependencies`/`Start`/`Stop`/`Health`.

The built-in system module provides auth, users, API keys, policies, audit, blobs, collections, operations, notifications, schedules, settings, and tenants. Extension modules register via `SetupConfig.Modules`.

---

## Security

`SecureDispatcher` enforces authorization on every message:

1. **PermissionManager.Resolve(msg)** maps message name → rule key
2. **AccessController.Can(ctx, ruleKey, resource)** evaluates the rule against the caller's identity (CEL-based, via go-iam)
3. Unauthorized calls receive `403 ERR_ACCESS_DENIED`

System-internal dispatches (bootstrap, password reset, token validation) bypass the security layer entirely. `RateLimitDispatcher` and `ThrottleDispatcher` enforce policy-defined rate limits and throttling.

---

## Bootstrap

1. **First run** — no admin seed exists. Creates an admin user with random credentials, generates an ephemeral API key, `bootstrapped = false`.
2. **Only bootstrap-safe routes** are exposed: `system:auth:session:create` (`POST /api/system/auth/session/create`) and `system:auth:bootstrap:password:set` (`PATCH /api/system/auth/bootstrap/password/set`).
3. User sets the admin password via the CLI (`hestia --bootstrap`), the client SDK, or directly against the bootstrap password route with the ephemeral key, new password, and email.
4. **On restart** — the stored password hash seed is compared to the current hash. If they differ (password was changed), `bootstrapped = true`.
5. All routes become available.

---

## Writing a Module

```
module/<name>/
├── module.go          // Module: Name(), Setup(rt), Capabilities(rt)
└── <feature>/
    ├── feature.go     // Register(rt) + Registrations(rt) → []MessageRegistration
    ├── handler.go     // service methods (annotate with @hestia.register)
    └── model.go       // store + input/output types
```

A module is a bundle of services. During the two-phase boot every module's
`Setup` registers its service providers into the shared runtime container
(`abstract.Register[T](rt, ctor)`); boot then seals the container once with
`Rebuild`, and calls each module's `Capabilities` to collect message
registrations:

```go
type Module struct {
	module.BaseModule
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "mymodule" }

func (m *Module) Setup(ctx context.Context, rt abstract.Container) error {
	return greeter.Register(rt) // registers the GreeterService provider
}

func (m *Module) Capabilities(rt abstract.Container) ([]abstract.Capability, error) {
	regs, err := greeter.Registrations(rt)
	if err != nil {
		return nil, err
	}
	return []abstract.Capability{{Name: "mymodule", Messages: regs}}, nil
}
```

The feature registers a service constructor into the container and resolves it
in `Registrations`; the handler is a method on that service:

```go
func Register(rt abstract.Container) error {
	return abstract.Register[*GreeterService](rt, func(c abstract.Container) (*GreeterService, error) {
		return NewGreeterService(c)
	})
}

func Registrations(rt abstract.Container) ([]abstract.MessageRegistration, error) {
	s := abstract.MustResolve[*GreeterService](rt)
	return []abstract.MessageRegistration{
		{Name: "mymodule:greeter:hello:create", Handler: dispatch.HandleDocument[HelloInput, *Greeting](s.Hello), Intent: abstract.Create},
	}, nil
}
```

Messages are named `module:feature:scope:action` (four segments). Boot validates
this grammar and registers every handler on the dispatcher.

Registration:

```go
hestia.Setup(hestia.SetupConfig{
	Modules: []hestia.Module{mymodule.New()},
})
```

Built-in features follow the same pattern — there is no privileged path. The
built-in services live under `core/system/<feature>/` and are collected by
`core/system/services.go` (generated by `hestia service generate --all`).

---

## Client SDK

The TypeScript SDK (`@asaidimu/hestia`, source in `client/`) mirrors the Go API over HTTP with reactive stores and auto-refresh JWT handling:

```ts
import { HestiaClient, WailsTransport } from "@asaidimu/hestia"

const api = new HestiaClient({ baseUrl: "http://localhost:8090" })
// Desktop Wails builds use the same API with a different transport:
// new HestiaClient({ transport: new WailsTransport() })

await api.auth.login("admin@test.local", "password123")
const { data: users } = await api.users.find()

const docs = api.collection<MyType>("articles")
await docs.create({ title: "Hello" })

const pager = api.users.page()          // reactive, observable pagination
pager.subscribe((page) => render(page.data))
await pager.resize(50, 1)
```

---

## CLI Tool

```bash
hestia init                          # scaffold a new project (hestia.json, entry point, Makefile)
hestia add module <name> [feature]   # scaffold a module (module.go + one feature service)
hestia add cmd <name>                # new server binary
hestia generate modules              # generate <autogen>/modules.go from the configured modules dirs
hestia generate features             # [library dev only] regenerate core/system/gen_features.go
hestia remove module <name>          # remove an external module
hestia service new <module> <name>   # scaffold a service inside a module
hestia service generate <module> <name>  # regenerate a service's registrations.go and policies.go
hestia service generate --all        # regenerate every module's services + collectors
```

`hestia.json` configures `module` (import path), `modules` (where modules live —
a single dir or a list; new modules/services are scaffolded into the first
entry), and `autogen` (where generated code, e.g. `modules.go`, is written).
The generated `<autogen>/modules.go` registry is passed to `SetupConfig.Modules`.
A module is a directory with a `module.go` plus service subdirectories; each
module's `services.go` collector is generated by `hestia service`.

---

## Project Structure

```
├── core/                          # framework
│   ├── hestia.go                  # public API — Setup, Application, SetupConfig
│   ├── abstract/                  # interfaces & envelope types (zero implementation)
│   ├── system/                    # the built-in system module: module + its services
│   │   ├── module.go              # SystemModule wiring (dispatcher chain, providers)
│   │   ├── services.go            # service collector (generated, wires all services)
│   │   ├── gen_features.go        # generated feature wiring (DO NOT EDIT)
│   │   ├── provider.go, seeds.go, policies.go
│   │   ├── <feature>/             # built-in services (auth, users, apikeys, ...)
│   │   │   ├── registrations.go / service.go / policies.go / model.go
│   │   │   └── model/*.schema.json    # anansi schema declarations
│   ├── interface/
│   │   ├── http/                  # fasthttp transport, auth middleware, routing
│   │   └── cli/                   # CLI transport (version/help/bootstrap)
│   ├── internal/
│   │   ├── boot/                  # application bootstrap, loggers, persistence manager
│   │   ├── migrations/            # migrated schema application
│   │   └── util/ , testutil/
│   └── runtime/                   # dispatchers, config, auth, policies, mailer, scheduler
├── client/                        # TypeScript SDK (@asaidimu/hestia)
├── cmd/
│   ├── hestia/                    # CLI tool (init, add, generate, remove, service)
│   ├── test-server/               # auto-reloading test server (:8070, auth bypass middleware)
│   ├── docs-server/               # documentation site server
│   ├── hestia-desktop/            # Wails desktop shell (WailsTransport)
│   ├── auth-test/                 # auth test utilities
│   └── gen-routes/                # route generation utilities
├── docs/                          # documentation (getting-started, guide, api, client)
├── examples/
│   ├── basic/                     # minimal standalone server
│   └── wails-test/                # desktop app example
├── schemas/                       # shared schema metadata
└── scripts/
```

---

## Dependencies

- **[go-anansi/v8](https://github.com/asaidimu/go-anansi)** — schema-driven document store, migrations, codegen
- **[go-iam/v2](https://github.com/asaidimu/go-iam)** — access control (CEL-based rules)
- **[go-events/v2](https://github.com/asaidimu/go-events)** — event bus (Pebble-backed)
- **go-sqlite3** / **pebble** — embedded SQLite + LSM storage
- **fasthttp** — HTTP server
- **cel-go** — CEL policy evaluation
- **golang-jwt/jwt/v5** — JWT tokens
- **go-cron** — scheduler
- **go-mail** — SMTP notifications
- **wails/v2** — desktop bindings
- **zap** — structured logging 
- **lumberjack** — log rotation
- **cobra** — CLI framework 
- **google/uuid** — UUIDv7 message IDs

---

## License

MIT

Copyright © 2026 asaidimu

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
