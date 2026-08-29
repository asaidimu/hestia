# hestia

> An embedded application kernel for Go — provides persistence, auth, policies, blobs, notifications, scheduling, and audit as library primitives, not as a standalone server.

[![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)
[![npm](https://img.shields.io/badge/npm-%40asaidimu%2Fhestia-CB3837?logo=npm&logoColor=white)](https://www.npmjs.com/package/@asaidimu/hestia)

**hestia** is not a web server, API server, or standalone daemon. It is a Go library you embed into your application. It turns a message-routing kernel into a full-featured application runtime — authentication, authorization, schema-driven persistence, blob storage, notifications, scheduling, audit logging, and a modular feature system — all in a single binary with zero external service dependencies. You decide what interfaces to expose (HTTP, CLI, Wails desktop, or your own).

> **Alpha — active development.** APIs, schemas, configuration, and tooling are changing rapidly and may break between releases. Pin versions and treat current docs as snapshots. Not yet recommended for production use.

---

## Quick Links

- [Overview & Features](#overview--features)
- [Installation & Setup](#installation--setup)
- [Usage Documentation](#usage-documentation)
- [Project Architecture](#project-architecture)
- [Development & Contributing](#development--contributing)
- [Troubleshooting & FAQ](#troubleshooting--faq)
- [License](#license)

---

## Overview & Features

### What hestia Is

hestia is an **application kernel** — a library that provides the foundational services your application needs, without prescribing how you expose them. Embed it in a web server, a CLI tool, a desktop app via Wails, an IoT firmware, or a mobile backend. The kernel handles:

- **Persistence** — schema-driven document store (SQLite embedded, pluggable to Postgres) with migrations, codegen, and live-change event bus
- **Identity & Access** — session-based auth (JWT + HTTP-only cookies), API key management, CEL-based policy rules, rate limiting, and throttling
- **Data Services** — dynamic schema-less collections (like Firestore), namespaced blob storage with resumable uploads
- **System Services** — notifications (SMTP), cron-based scheduling, audit logging, runtime settings, multi-tenant isolation, self-update

You wire hestia into your application, register your own modules alongside the built-in ones, and choose which interfaces to attach. The kernel does the rest.

### Key Features

- **Embedded Kernel** — a Go library, not a standalone server. You embed it and control the lifecycle.
- **Schema-Driven Persistence** — declare data models as `.schema.json` files; migrations, model structs, and DTOs are generated automatically.
- **Message-Routing Core** — every operation is a named message (`module:feature:scope:action`). HTTP routes, CLI commands, permission scopes, and SDK method names are all mechanically derived from the message name. No manual route registration.
- **Modular Architecture** — the built-in system module provides auth, users, API keys, policies, blobs, collections, operations, audit, notifications, schedules, settings, and tenants. You add your own modules via `SetupConfig.Modules`.
- **Pluggable Interfaces** — attach HTTP (fasthttp), CLI (cobra), Wails desktop, or build your own transport against the `runtime.Interface` contract.
- **Policy-Based Access Control** — CEL expressions for fine-grained authorization, with live rule compilation and hot-reload.
- **TypeScript Client SDK** — `@asaidimu/hestia` mirrors the Go API over HTTP with reactive stores, auto-refresh JWT handling, and observable pagination.
- **Zero External Services** — SQLite embedded in-process, no Redis, no separate database process. Everything runs in a single binary.

---

## Installation & Setup

### Prerequisites

- Go >= 1.27
- C compiler (for `mattn/go-sqlite3` CGo bindings)
- Node.js / Bun (for the TypeScript client SDK)

### Go Library

```bash
go get github.com/asaidimu/hestia
```

### CLI Tool (scaffolding & codegen)

```bash
go install github.com/asaidimu/hestia/cmd/hestia@latest
```

### TypeScript Client SDK

```bash
npm install @asaidimu/hestia
# or
bun add @asaidimu/hestia
```

### Minimal Entry Point

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
    select {} // block forever; app runs in background
}
```

`Setup` loads configuration from the environment (or `.env`), applies migrations, hydrates the built-in services, and returns an `Application` you control. You decide which interfaces to attach and when to start.

### Configuration

`Setup` reads environment variables (also loading `.env` and `.env.dev` if present via godotenv). Settings applied directly in `SetupConfig` override env vars.

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
| `ALLOWED_ORIGINS` | `""` | Comma-separated CORS origins |
| `FORCE_BOOTSTRAPPED` | `false` | Skip the bootstrap flow |
| `LOG_PATH` | `<data_dir>/server.log` | Log file path |
| `LOG_MAX_SIZE` | `100` | Log rotation size (MB) |
| `LOG_MAX_AGE` | `30` | Log retention (days) |
| `SMTP_HOST` | — | Outbound mail (notifications) |
| `SMTP_PORT` | — | SMTP port |

See `core/hestia.go:146` for the full `SetupConfig` struct.

---

## Usage Documentation

### Basic Usage — Web Server

```go
app, err := hestia.Setup(hestia.SetupConfig{
    ProjectName:   "myapp",
    SessionSecret: "change-me",
})
if err != nil {
    log.Fatal(err)
}

// Attach an HTTP interface
http := app.NewHTTPInterface(http.Config{
    Addr: ":8090",
})
app.AddInterface(http)

if err := app.Start(); err != nil {
    log.Fatal(err)
}
defer app.Close()

select {}
```

### Basic Usage — Desktop App (Wails)

```go
app, err := hestia.Setup(hestia.SetupConfig{
    ProjectName:   "myapp",
    SessionSecret: "change-me",
    BuildInterfaces: func(app *hestia.Application, cfg ...*runtime.Config) []runtime.Interface {
        return []runtime.Interface{
            app.NewWailsInterface(wails.Options{...}),
        }
    },
})
```

### Basic Usage — With Custom Modules

```go
app, err := hestia.Setup(hestia.SetupConfig{
    ProjectName:   "myapp",
    SessionSecret: "change-me",
    Modules: []hestia.Module{
        mymodule.New(),
        billingmodule.New(),
    },
})
```

### Client SDK

```ts
import { HestiaClient } from "@asaidimu/hestia"

const api = new HestiaClient({ baseUrl: "http://localhost:8090" })

// Login
await api.auth.login("admin@test.local", "password123")

// List users
const { data: users } = await api.users.find()

// Dynamic collections
const docs = api.collection<Article>("articles")
await docs.create({ title: "Hello World" })

// Reactive pagination
const pager = api.users.page()
pager.subscribe((page) => render(page.data))
await pager.resize(50, 1)
```

### Setup API

```go
// Build the application: load config, migrate, hydrate features, build interfaces.
func Setup(cfg SetupConfig) (*Application, error)

// Application methods:
app.Persistence()   base.Persistence
app.Dispatcher()    abstract.Dispatcher
app.SystemModule()  SystemModule
app.Registrations() []abstract.MessageRegistration
app.RegisterModules(m ...Module) error
app.NewHTTPInterface(cfg http.Config) runtime.Interface
app.NewCLIInterface(cfg cli.Config) runtime.Interface
app.Start() error
app.Shutdown(ctx context.Context) error
app.Close()
```

### CLI Tool

```bash
hestia init                          # scaffold a new project
hestia add module <name> [feature]   # scaffold a module
hestia add cmd <name>                # new server binary
hestia generate modules              # generate module registry
hestia generate features             # regenerate feature wiring
hestia remove module <name>          # remove a module
hestia service new <module> <name>   # scaffold a service
hestia service generate <module> <name>  # regenerate service code
hestia service generate --all        # regenerate all services
```

---

## Project Architecture

### Core Concepts

#### Message

Every operation in hestia is a named message — a colon-delimited quadruple:

```
module:feature:scope:action
```

HTTP routes, CLI commands, permission scopes, and SDK method names are mechanically derived from this name. No manual route registration.

| Message | HTTP |
|---|---|
| `system:auth:session:create` | `POST /api/system/auth/session/create` |
| `system:users:user:get` | `GET /api/system/users/user/get/{user_id}` |
| `system:collections:document:get` | `GET /api/system/collections/document/get/{name}/{doc_id}` |

#### Dispatcher Chain

Every message passes through a chain of middleware before reaching the handler:

```
HTTP ─▶ fasthttp transport ─▶ auth middleware ─▶ route closure ─▶ Dispatcher Chain ─▶ handler
                                                                │
                          ┌─────────────────────────────────────┴──────────────────────────┐
                          │ bootstrap → secure → ratelimit → throttle                       │
                          │ → tenant → blob → recovery → audit                              │
                          │ → LocalDispatcher                                               │
                          └─────────────────────────────────────────────────────────────────┘
```

#### Module

A module is a bundle of services. During the two-phase boot, every module's `Setup` registers its service providers into the shared runtime container; boot seals the container, and `Capabilities` collects message registrations.

```
module/<name>/
├── module.go          // Module: Name(), Setup(rt), Capabilities(rt)
└── <feature>/
    ├── feature.go     // Register(rt) + Registrations(rt) → []MessageRegistration
    ├── handler.go     // service methods
    └── model.go       // store + input/output types
```

### Built-in Features

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

### Persistence

hestia uses **[go-anansi/v8](https://github.com/asaidimu/go-anansi)** as its document store. Schemas are declared as plain `.schema.json` files, and models/DTOs are generated from them.

The default backend is **SQLite** (via `mattn/go-sqlite3`, WAL mode), embedded in-process. A **Pebble-backed event bus** powers change notifications.

#### Schema Workflow

1. Edit `model/*.schema.json` (never rename existing IDs)
2. `anansi migrate generate --dry-run` to validate
3. `anansi migrate generate` to create the migration
4. `anansi codegen golang` to regenerate model structs & DTOs

#### Custom Backend

The persistence layer is database-agnostic. Everything goes through `query.DatabaseInteractor`:

```go
type DatabaseInteractor interface {
    SchemaManager
    SelectDocuments(ctx, schema, query) ([]map[string]any, int64, error)
    InsertDocuments(ctx, schema, records) ([]map[string]any, error)
    UpdateDocuments(ctx, schema, updates, filters, returning) ([]map[string]any, int64, error)
    DeleteDocuments(ctx, schema, filters, unsafeDelete) (int64, error)
    StartTransaction(ctx) (DatabaseInteractor, error)
    Commit(ctx) error
    Rollback(ctx) error
    // ...
}
```

Plug in Postgres or any other backend via `SetupConfig.PersistenceFactory`.

### Request Lifecycle

1. **Auth middleware** tries: session cookie → API key → anonymous
2. **Route closure** builds `{arguments, modifiers, payload}` from the HTTP request
3. **Dispatcher chain** layers bootstrap gating, authorization, rate limiting, throttling, tenant scoping, blob routing, panic recovery, and audit logging
4. **Handler** executes business logic, returns a `*Result` with a `Kind` discriminant
5. **Serializer** converts the result to an HTTP response

### Bootstrap

1. **First run** — no admin seed exists. Creates an admin user with random credentials, generates an ephemeral API key.
2. **Only bootstrap-safe routes** are exposed: `system:auth:session:create` and `system:auth:bootstrap:password:set`.
3. User sets the admin password via the CLI (`hestia --bootstrap`), the client SDK, or directly.
4. **On restart** — the stored password hash seed is compared to the current hash. If they differ, `bootstrapped = true`.
5. All routes become available.

---

## Writing a Module

### Project Structure

```
module/<name>/
├── module.go          // Module: Name(), Setup(rt), Capabilities(rt)
└── <feature>/
    ├── feature.go     // Register(rt) + Registrations(rt) → []MessageRegistration
    ├── handler.go     // service methods (annotate with @hestia.register)
    └── model.go       // store + input/output types
```

### Module Interface

```go
type Module interface {
    Name() string
    Setup(ctx context.Context, rt abstract.Container) error
    Capabilities(rt abstract.Container) ([]Capability, error)
    Dependencies() []string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) any
}
```

### Example Module

```go
type Module struct {
    module.BaseModule
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "mymodule" }

func (m *Module) Setup(ctx context.Context, rt abstract.Container) error {
    return greeter.Register(rt)
}

func (m *Module) Capabilities(rt abstract.Container) ([]abstract.Capability, error) {
    regs, err := greeter.Registrations(rt)
    if err != nil {
        return nil, err
    }
    return []abstract.Capability{{Name: "mymodule", Messages: regs}}, nil
}
```

### Registration

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

### Wiring Into the Application

```go
app, err := hestia.Setup(hestia.SetupConfig{
    Modules: []hestia.Module{mymodule.New()},
})
```

---

## Development & Contributing

### Available Scripts

- `make build` — Build the CLI tool
- `make test` — Run all Go tests
- `make test-server` — Build the auto-reloading test server
- `make test-client` — Run TypeScript client tests

### Testing & Quality Standard

```bash
make test
```

### Test Server

A live, auto-reloading test server runs at `./cmd/test-server` on port **8070**.

```sh
lsof -ti :8070 | xargs kill   # stop the old process
go build -o test-server ./cmd/test-server
./test-server &
```

Default credentials: `admin@test.local` / `password123`

### Contributing Guidelines

1. Fork the project repository.
2. Create a feature branch (`git checkout -b feature/amazing-feature`).
3. Commit changes (`git commit -m 'feat: add amazing feature'`).
4. Push to branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

---

## Troubleshooting & FAQ

### Troubleshooting

- **CGo build fails**: Ensure a C compiler is installed (`gcc` on Linux, `xcode-select --install` on macOS).
- **Port already in use**: Change the `PORT` environment variable or use `SetupConfig` to set a different port.
- **Auth fails on first request**: The system bootstraps on first run. Use the ephemeral API key from startup logs, or set `FORCE_BOOTSTRAPPED=true` for development.

### FAQ

- **Q**: Is hestia a web framework?
  **A**: No. hestia is an application kernel — a Go library you embed. It provides persistence, auth, policies, and other services as primitives. You choose how to expose them (HTTP, CLI, desktop, etc.).

- **Q**: Can I use Postgres instead of SQLite?
  **A**: Yes. The persistence layer is database-agnostic. Implement `query.DatabaseInteractor` and pass it via `SetupConfig.PersistenceFactory`.

- **Q**: How do I add custom authentication?
  **A**: hestia's auth is built-in but extensible. You can add middleware to the dispatcher chain via `SetupConfig.DispatcherChainFunc` or implement a custom `CredentialsProvider`.

- **Q**: Does it work on mobile or IoT?
  **A**: Yes. hestia embeds SQLite and runs in-process. It has no external service dependencies, making it suitable for edge, mobile, and IoT deployments.

---

## License

Distributed under the MIT License. See `LICENSE.md` for more information.

## Acknowledgments

- **[go-anansi/v8](https://github.com/asaidimu/go-anansi)** — schema-driven document store, migrations, codegen
- **[go-iam/v2](https://github.com/asaidimu/go-iam)** — access control (CEL-based rules)
- **[go-events/v2](https://github.com/asaidimu/go-events)** — event bus (Pebble-backed)
- **[go-sqlite3](https://github.com/mattn/go-sqlite3)** — embedded SQLite
- **[fasthttp](https://github.com/valyala/fasthttp)** — HTTP transport
- **[cel-go](https://github.com/google/cel-go)** — CEL policy evaluation
- **[go-cron](https://github.com/netresearch/go-cron)** — scheduler
- **[go-mail](https://github.com/wneessen/go-mail)** — SMTP notifications
- **[wails/v2](https://github.com/wailsapp/wails)** — desktop bindings
- **[zap](https://github.com/uber-go/zap)** — structured logging
- **[cobra](https://github.com/spf13/cobra)** — CLI framework
