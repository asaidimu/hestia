# hestia

**hestia** is an embedded application framework for Go that turns a message-routing kernel into a full-featured API server. It provides authentication, authorization, persistence, blob storage, access logging, and a modular feature system — all in a single binary with no external dependencies.

## Quick Start

```go
package main

import (
    "fmt"
    "os"

    "github.com/asaidimu/hestia"
)

func main() {
    if err := hestia.Run(hestia.SetupConfig{}); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

Set `JWT_SECRET` and run:

```bash
JWT_SECRET=my-secret go run main.go
```

That starts a full server on `:8090` with auth, users, API keys, policies, blobs, audit logging, collection management, and a CLI.

## Install

```bash
go get github.com/asaidimu/hestia
```

CLI tool:

```bash
go install github.com/asaidimu/hestia/cmd/hestia@latest
```

---

## Configuration

### Environment Variables

| Var | Default | Required | Description |
|---|---|---|---|
| `JWT_SECRET` | — | **yes** | HMAC-SHA256 signing key (any length) |
| `PORT` | `:8090` | no | Listen address (must include colon; `:0` for random) |
| `APP_DATA_DIR` | `~/.local/share/anansi` | no | All persistent state (DB, logs, blobs) |
| `DB_PATH` | `<data_dir>/anansi.db` | no | SQLite file path; `:memory:` for in-memory |
| `BCRYPT_COST` | `12` | no | Password hashing cost (4–31) |
| `JWT_ACCESS_TTL` | `15m` | no | Access token lifetime (Go duration) |
| `JWT_REFRESH_TTL` | `168h` (7d) | no | Refresh token lifetime |
| `JWT_RESET_TTL` | `5m` | no | Password reset token lifetime |
| `BLOBS_DIR` | `<data_dir>/blobs` | no | Blob file storage root |
| `LOGS_DIR` | `<data_dir>` | no | Structured JSON log output (disabled if empty) |
| `LOG_MAX_SIZE` | `100` | no | Log rotation size (MB) |
| `LOG_MAX_AGE` | `30` | no | Log retention (days) |
| `LOG_MAX_BACKUPS` | `5` | no | Max old log files kept |
| `COOKIE_DOMAIN` | `""` | no | Cookie domain restriction |
| `COOKIE_SECURE` | `true` | no | Require HTTPS for cookies |
| `COOKIE_SAMESITE` | `strict` | no | `strict`, `lax`, or `none` |
| `ACCESS_COOKIE_NAME` | `access_token` | no | Access token cookie name |
| `ACCESS_COOKIE_PATH` | `/` | no | Access token cookie path |
| `REFRESH_COOKIE_NAME` | `refresh_token` | no | Refresh token cookie name |
| `REFRESH_COOKIE_PATH` | `/api/auth/session` | no | Refresh token cookie path |

### Programmatic Config

The `core.Config` struct (in `github.com/asaidimu/hestia/internal/core`) exposes every setting, plus additional fields not available from env vars:

```go
type Config struct {
    Port              string            // ":8090"
    DataDir           string            // persistent state root
    DBPath            string            // "file:...?cache=shared&_fk=1" for SQLite
    JWTSecret         string            // required
    BlobsDir          string            // file storage
    BcryptCost        int               // 12
    AccessTokenTTL    time.Duration     // 15m
    RefreshTokenTTL   time.Duration     // 168h
    ResetTokenTTL     time.Duration     // 5m
    AdminEmail        string            // override random seed email
    AdminPassword     string            // override random seed password
    ForceBootstrapped bool              // skip bootstrap flow
    LogPath           string
    LogMaxSize        int
    LogMaxAge         int
    LogMaxBackups     int
    CookieConfig      CookieConfig
    InteractorFactory InteractorFactory          // see "Persistence" section
    PersistenceFactory func(*anansi.SetupConfig) (base.Persistence, error) // see "Persistence" section
}
```

Pass a `*core.Config` directly via `SetupConfig.Config` to bypass env-var loading:

```go
hestia.Run(hestia.SetupConfig{
    Config: &core.Config{
        Port:      ":8080",
        JWTSecret: "my-secret",
        DBPath:    ":memory:",
    },
})
```

---

## Setup API

```go
// Full lifecycle — builds app, starts orchestrators, blocks on signal.
func Run(cfg SetupConfig) error

// Build app and return handles — for custom orchestration.
func Setup(cfg SetupConfig) (*boot.Application, *app.SystemModule, error)

// Helper to construct the dispatcher chain and orchestrators.
func BuildOrchestrators(a *boot.Application, mod *app.SystemModule, version string) Orchestrators

// Prints bootstrap instructions (ephemeral key, admin info) to stdout.
func PrintBootstrapStatus(a *boot.Application, mod *app.SystemModule)
```

```go
type SetupConfig struct {
    Config          *core.Config                     // nil → auto-detect from env
    Version         string                           // printed in CLI `version` command
    Modules         []Module                         // user-defined modules
    Options         app.Options                      // ForceBootstrapped, admin overrides
    Migrate         func(ctx, base.Persistence) error // user migrations after hestia's
    DispatcherHooks []func(Dispatcher) Dispatcher     // chain middleware layers
}
```

---

## Persistence

hestia uses **[go-anansi/v8](https://github.com/asaidimu/go-anansi)** as its document store. The default backend is **SQLite** (via `mattn/go-sqlite3` with WAL mode), embedded in-process — no separate database process to manage.

### Default Setup (SQLite)

By default the persistence manager creates a SQLite database at `DB_PATH` (`<data_dir>/anansi.db`) with:

- **WAL journal mode** for concurrent read/write
- **Foreign keys** enabled (`_fk=1`)
- **Shared cache** (`cache=shared`) for in-process access
- A **Pebble-backed event bus** for change notifications
- **Sanitization** with automatic field-masking of passwords, tokens, secrets, API keys, and hashes

### Custom Backend via InteractorFactory

The persistence layer is fully database-agnostic. Everything goes through `query.DatabaseInteractor`:

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

To switch backends, set `Config.InteractorFactory`:

```go
cfg := &core.Config{
    InteractorFactory: func(logger *zap.Logger) (query.DatabaseInteractor, func(), error) {
        // Return any DatabaseInteractor implementation
        interactor, closer := myPostgresInteractor(logger)
        return interactor, closer, nil
    },
}
```

This replaces the default `NewDatabase()` call entirely — the interactor you return is plugged directly into go-anansi's document layer. All hestia features (auth, users, blobs, collections, audits, policies, etc.) work unchanged.

### Custom Persistence

`Config.PersistenceFactory` gives you full control. The factory receives a minimal `anansi.SetupConfig` with the logger and sanitization rules pre-populated. No interactor or event bus is created — you set those up yourself.

```go
import (
    "github.com/asaidimu/go-anansi/v8"
    "github.com/asaidimu/go-anansi/v8/core/persistence/base"
    "github.com/asaidimu/go-anansi/v8/core/persistence/orchestrator"
    "github.com/asaidimu/go-anansi/v8/core/query"
)

hestia.Run(hestia.SetupConfig{
    Config: &core.Config{
        PersistenceFactory: func(cfg *anansi.SetupConfig) (base.Persistence, error) {
            interactor, _, _ := myPostgresInteractor(cfg.Logger)
            bus := events.NewSimple[...](...) // your event bus

            cfg.Interactor = interactor
            cfg.EventBus = bus

            return anansi.Setup(*cfg)
        },
    },
})
```

Unlike the default path, no interactor, database connection, or event bus is created before the factory. You own the full lifecycle.

### Custom Dispatcher Layers

`SetupConfig.DispatcherHooks` adds middleware after the default chain (SecureDispatcher → BlobDispatcher → AccessLogDispatcher):

```go
hestia.Run(hestia.SetupConfig{
    DispatcherHooks: []func(hestia.Dispatcher) hestia.Dispatcher{
        func(next hestia.Dispatcher) hestia.Dispatcher {
            return myratelimit.New(next, 100)
        },
    },
})
```

Each hook receives and returns a `Dispatcher`. The outermost hook runs first on entry and last on exit:

### PostgreSQL

go-anansi currently ships only SQLite adapters. To use PostgreSQL:

1. Implement `query.DatabaseInteractor` for PostgreSQL (or use a community adapter)
2. Optionally implement `query.QueryFactory` if you need custom query DSL translation
3. Pass it via `InteractorFactory`

The `InteractorOptions.SchemaName` field exists specifically for PostgreSQL schema support. The same approach works for any SQL or NoSQL backend that can implement the interface.

---

## Request Lifecycle

```
HTTP ─▶ Transport ─▶ authMiddleware ─▶ routeClosure ─▶ Dispatcher Chain ─▶ handler
                                                         │
                                             ┌───────────┴───────────┐
                                             │ AccessLogDispatcher    │
                                             │ SecureDispatcher       │
                                             │ BlobDispatcher         │
                                             │ LocalDispatcher        │
                                             └───────────────────────┘
```

1. **Auth middleware** tries: Bearer JWT → access cookie → API key → anonymous
2. **Route closure** builds `{arguments, modifiers, payload}` from the HTTP request
3. **Dispatcher chain** layers on security, blob routing, audit logging
4. **Handler** executes business logic, returns `*Result` with `Kind` discriminant
5. **Serializer** converts result to HTTP response (sanitizes once)

---

## Core Concepts

### Dispatcher

```go
type Dispatcher interface {
    Send(msg Message) (*Result, error)
}
```

The central routing primitive. Implementations wrap each other as decorators. The base `LocalDispatcher` holds a `map[string]handlerEntry` (thread-safe).

### Message

```go
type Message interface {
    ID() string
    Name() string
    Context() context.Context
    Input() *data.Document
}
```

Named envelopes routed through the chain. Names are colon-delimited quadruples:

```
module:feature:scope:action
```

HTTP routes, permission scopes, and SDK method names are mechanically derived from the message name — no manual route registration.

| Message | HTTP |
|---|---|
| `system:auth:session:create` | `POST /api/auth/session` |
| `system:users:user:get` | `GET /api/users/user/{user_id}` |
| `system:auth:session:delete` | `DELETE /api/auth/session` |
| `collection:articles:document:read` | `GET /api/collection/articles/document/{doc_id}` |

Input is always `{arguments, modifiers, payload}`. Handlers access fields via `doc.GetOr("payload.email", "")`.

### Result

```go
type Result struct {
    Kind            ResultKind          // explicit discriminant
    Document        *data.Document
    Documents       data.DocumentSet
    Page            *Page
    Blob            Blob
    DocumentChannel <-chan *data.Document
    BlobChannel     <-chan Blob
}
```

Construct via helpers — never populate `Result` fields directly:

```go
registration.NewDocumentResult(doc)
registration.NewDocumentsResult(docs)
registration.NewPageResult(page)
registration.NewBlobResult(blob)
registration.NewDocumentChannelResult(ch)
registration.NewBlobChannelResult(ch)
```

### Module

```go
type Module interface {
    Name() string
    Setup(ctx context.Context, persist base.Persistence) error
    Capabilities() []Capability
}
```

The default system module (`internal/app/`) provides auth, users, API keys, policies, audit, blobs, and collections. Extension modules live under `module/<name>/` and register via `SetupConfig.ExtraModules`.

---

## Security

`SecureDispatcher` enforces authorization on every message:

1. **PermissionManager.Resolve(msg)** maps message name → rule key
2. **AccessController.Can(ctx, ruleKey, resource)** evaluates the rule against the caller's identity
3. Unauthorized calls receive `403 ERR_ACCESS_DENIED`

System-internal dispatches (bootstrap, password reset, token validation) bypass the security layer entirely.

Built-in features manage policies, API keys, sessions, and their permissions through the same message system — no special admin API.

---

## Bootstrap

1. **First run** — no admin seed exists. Creates admin user with random credentials, generates an ephemeral API key, `bootstrapped = false`
2. **Only bootstrap-safe routes** are exposed: `POST /api/auth/session` and `PUT /api/bootstrap/password`
3. User calls `PUT /api/bootstrap/password` with the ephemeral API key, new password, and email
4. **On restart** — `SeedAdmin` compares the stored password hash seed vs the current hash. If they differ (password was changed), `bootstrapped = true`
5. All routes become available

---

## Writing a Module

```
module/<name>/
├── module.go
└── feature/<feature>/
    ├── register.go     // Registrations(deps) → []MessageRegistration
    ├── handler.go      // business logic closures
    ├── model.go        // persistence wrapper
    ├── schema.go       // input/output schemas
    ├── defaults.go     // DefaultOperations() → permission scopes
    ├── policies.go     // custom rules (optional)
    └── seed.go         // seed data (optional)
```

Handlers are stateless constructors returning closures:

```go
type Dependencies struct {
    UserModel *users.UserModel
    JWTService core.JWTService
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
    return []abstract.MessageRegistration{
        {Name: "mymodule:greeter:hello:create", Handler: newHandler(deps), Intent: registration.Create},
    }
}

func newHandler(deps Dependencies) abstract.MessageHandler {
    return func(ctx context.Context, msg abstract.Message) (*registration.Result, error) {
        name, _ := msg.Input().GetOr("payload.name", "").(string)
        doc := data.MustNewDocument(map[string]any{"greeting": "Hello, " + name}, ctx)
        return registration.NewDocumentResult(doc), nil
    }
}
```

Registration:

```go
hestia.Run(hestia.SetupConfig{
    ExtraModules: []hestia.Module{mymodule.New()},
})
```

Built-in features follow the same pattern — there is no privileged path.

---

## CLI Tool

```bash
hestia generate                    # downstream: generate modules.go from module_sources
hestia generate --self             # library: also scan internal/app features
hestia scaffold cmd <name>         # new server binary
hestia module scaffold <name>      # new module skeleton
```

The `generate` command creates `internal/autogen/modules.go` from `module_sources` in `hestia.json`. The `--self` flag additionally scans `internal/app/*/feature.go` for the library's own feature wiring.

---

## Project Structure

```
├── hestia.go                  # public API — Setup, Run, BuildOrchestrators
├── hestia.json                # project config
├── cmd/
│   ├── hestia/                # CLI tool
│   ├── server/                # server binary
│   └── test-server/           # test server with programmatic config
├── internal/
│   ├── abstract/              # interfaces & envelope types (zero implementation)
│   ├── core/                  # framework implementations
│   │   ├── local-dispatcher.go
│   │   ├── secure-dispatcher.go
│   │   ├── access-log-dispatcher.go
│   │   ├── config.go
│   │   ├── errors.go
│   │   ├── blobstore/
│   │   └── identity/
│   ├── app/                   # default system module
│   │   ├── module.go
│   │   ├── auth/ users/ apikeys/ policies/ audit/ blobs/ collections/ operations/
│   │   └── gen_features.go    # auto-generated feature wiring
│   ├── interface/             # protocol adapters
│   │   ├── api/               # HTTP transport & orchestrator
│   │   └── cli/               # CLI transport
│   ├── boot/                  # startup wiring (config, DB, persistence, builder)
│   └── utility/persistest/    # in-memory persistence for model tests
├── module/
│   └── greeter/               # example module
└── migrations/                # generated migration files
```

---

## Dependencies

- **[go-anansi/v8](https://github.com/asaidimu/go-anansi)** — document store, schemas, queries, sanitization
- **[go-iam/v2](https://github.com/asaidimu/go-iam)** — access control (CEL-based rules)
- **[go-events/v2](https://github.com/asaidimu/go-events)** — event bus (Pebble-backed)
- **go-sqlite3** / **pebble** — embedded SQLite + LSM storage
- **golang-jwt/jwt/v5** — JWT tokens
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
