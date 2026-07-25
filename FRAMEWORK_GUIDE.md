# Hestia Framework Guide

## Architecture Overview

Hestia is a message-driven application framework. Every operation is converted into a **Message** that flows through a **dispatcher chain** to a registered **handler** that returns a **Result**.

The dispatcher chain is transport-agnostic — it knows nothing about HTTP, CLI, WebSocket, or any other interface. Transports sit in front of the chain, converting external input into Messages and serializing Results back to the caller.

```
┌─ Transport(s) ─────────────────────────────────────┐
│  HTTP    ──→ auth middleware ──→ Send(msg)          │
│  CLI     ──→ system identity  ──→ Send(msg)          │
│  Wails   ──→ system identity  ──→ Send(msg)          │
│  Custom  ──→ (any)            ──→ Send(msg)          │
└──────────────────────┬──────────────────────────────┘
                       │ msg
                       ▼
┌─ Dispatcher Chain ──────────────────────────────────┐
│  SecureDispatcher  (IAM enforcement)                 │
│  AuditDispatcher   (audit logging)                   │
│  RecoveryDispatcher (panic recovery)                 │
│  NamespacedDispatcher (prefix rewriting, e.g. blobs) │
│  Custom DispatcherHooks (user-provided)              │
│  LocalDispatcher (handler registry)                  │
└──────────────────────┬──────────────────────────────┘
                       │ result
                       ▼
                  Transport serializes Result
```

All transports are optional. You can run with only CLI (`DisableHTTP: true`), only HTTP (`DisableCLI: true`), both, or neither (programmatic dispatch).

---

## Core Abstractions

### Message (core/abstract/message.go)

```go
type Message interface {
    ID() string                    // UUIDv7
    Name() string                  // "module:feature:scope:action"
    Context() context.Context      // carries identity, audit, trace
    Input() *data.Document         // structured input
    InputChannel() <-chan *data.Document   // streaming input
    BlobInputChannel() <-chan Blob         // blob streaming
}
```

Messages follow `module:feature:scope:action` (4 colon-separated segments):

| Segment | Example |
|---------|---------|
| `module` | `system`, `blob` |
| `feature` | `auth`, `users`, `apikeys`, `collections` |
| `scope` | `session`, `user`, `key`, `document` |
| `action` | `create`, `read`, `update`, `delete`, `list`, `query` |

Examples: `system:auth:session:create`, `system:users:user:list`, `system:blobs:avatars:upload`.

Validated by `boot.validateMessageName()` — exactly 4 segments required.

**Internal messages** (`Internal: true`): not visible to any external consumer. Used for intra-system communication (e.g., `system:auth:apikey:validate`, `system:core:audit:log`).

**Bootstrap-safe messages** (`BootstrapSafe: true`): registered before the system is bootstrapped (first-run setup phase).

### Dispatcher (core/abstract/dispatcher.go)

```go
type Dispatcher interface {
    Send(msg Message) (*Result, error)
}
```

A single-method interface. Dispatchers are composable via the decorator pattern.

### Registry (core/abstract/dispatcher.go)

```go
type Registry interface {
    RegisterHandler(name string, handler MessageHandler, info HandlerInfo) error
    GetHandler(name string) (MessageHandler, error)
    DeleteHandler(name string) error
    ListHandlers() []HandlerInfo
    SetHandlerEnabled(name string, enabled bool) error
}
```

### MessageHandler

```go
type MessageHandler func(context.Context, Message) (*Result, error)
```

### Module (core/abstract/module.go)

```go
type Module interface {
    Name() string
    Setup(ctx context.Context, persist base.Persistence) error
    Capabilities() []Capability
}
```

A module provides `Capabilities`, each containing `MessageRegistration` entries that wire handlers to message names.

### MessageRegistration

```go
type MessageRegistration struct {
    Name          string
    Handler       MessageHandler
    Description   string
    Intent        Verb
    Enabled       bool
    BootstrapSafe bool
    Internal      bool       // Hidden from external consumers
    Input         Input
    Output        *definition.Schema
}
```

### Input

```go
type Input struct {
    Schema          *definition.Schema  // Input document schema
    Arguments       []ArgDef            // Path/arguments params: {name: type}
    Modifiers       map[string]definition.FieldType  // Query/filter params
    Payload         definition.FieldType  // Body type (object, bytes, record, or 0 = none)
    ResourceIDField string              // Which argument holds the resource ID (for audit)
}
```

When a transport builds the input document, it populates:
- `arguments` — from path/arg params (matched against `Arguments`)
- `modifiers` — from query/filter params (matched against `Modifiers`)
- `payload` — from the request body (when `Payload != 0`)

Handlers access these via:
```go
userID, _ := msg.Input().GetOr("arguments.user_id", "").(string)
body, _ := msg.Input().GetOr("payload", nil).(map[string]any)
filter, _ := msg.Input().GetOr("modifiers.filter", "").(string)
```

### Transport (core/abstract/transport.go)

```go
type Transport interface {
    Handle(pattern string, handler Handler)
    Start() error
    Shutdown(ctx context.Context) error
}
```

The `transport.go` file defines the abstract transport used by the HTTP interface. Other transports implement `runtime.Interface` directly.

### Interface (core/runtime/interface.go)

```go
type Interface interface {
    Start(bootstrapped bool)
    Restart(bootstrapped bool)
    Shutdown(ctx context.Context) error
}
```

A transport-layer interface that receives external input, converts it to Messages, dispatches them, and returns Results.

---

## Verbs (Intent)

`Verb` describes the semantic intent of an operation. Verbs are transport-agnostic — each transport maps them to its own protocol conventions.

### Verb Definitions

| Verb | Semantics |
|------|-----------|
| `Create` | Produce a new resource |
| `Read` | Retrieve an existing resource |
| `Update` | Modify an existing resource |
| `Delete` | Remove an existing resource |
| `Query` | Search or list with complex filters |
| `Stream` | Real-time events (server-pushed) |
| `Check` | Validate without side effects |

### Usage in Registrations

```go
// Create — produces a new resource
{Intent: registration.Create, Input: runtime.Input{Payload: definition.FieldTypeObject}}

// Read — retrieves by identifier
{Intent: registration.Read, Input: runtime.Input{
    Arguments: []abstract.ArgDef{{Name: "id", Type: definition.FieldTypeString}},
    ResourceIDField: "id",
}}

// Update — partial mutation
{Intent: registration.Update, Input: runtime.Input{
    Arguments: []abstract.ArgDef{{Name: "id", ...}},
    Payload:   definition.FieldTypeObject,
}}

// Delete — removal
{Intent: registration.Delete, Input: runtime.Input{
    Arguments: []abstract.ArgDef{{Name: "id", ...}},
}}

// Query — search with structured query DSL
{Intent: registration.Query, Input: runtime.Input{Payload: definition.FieldTypeRecord}}

// Stream — real-time push
{Intent: registration.Stream}

// Check — validation
{Intent: registration.Check, Input: runtime.Input{Payload: definition.FieldTypeObject}}
```

---

## Result Types (core/abstract/result.go, core/registration/registration.go)

Handlers return `*registration.Result`:

| Kind | Constructor | Transport Handling |
|------|-------------|-------------------|
| `ResultKindDocument` | `NewDocumentResult(doc)` | Single JSON object |
| `ResultKindDocuments` | `NewDocumentsResult(docs)` | JSON array |
| `ResultKindPage` | `NewPageResult(page)` | Paginated (documents + pagination info) |
| `ResultKindBlob` | `NewBlobResult(blob)` | Raw bytes with content type |
| `ResultKindDocumentChannel` | `NewDocumentChannelResult(ch)` | Stream of documents |
| `ResultKindBlobChannel` | `NewBlobChannelResult(ch)` | Stream of blobs |

Special field:
- `SessionToken string` — transport-specific; the HTTP transport uses this to set a session cookie. Other transports ignore it.
- `Result{}` (zero value) — empty response.

---

## Dispatcher Implementations

### LocalDispatcher (core/runtime/local-dispatcher.go)

The terminal dispatcher and handler registry. Holds `map[string]handlerEntry`. Thread-safe.

```go
disp := runtime.NewLocalDispatcher()
disp.RegisterHandler("my:feature:item:create", myHandler, runtime.HandlerInfo{...})
```

Checks: nil context, missing handler, disabled handler.

### SecureDispatcher (core/runtime/secure-dispatcher.go)

IAM enforcement. System identities (internal callers) bypass all checks. For all others:

1. **API key operation gate** — API key identities with an operation allowlist only pass named ops.
2. **Policy resolution** — `PermissionManager.Resolve(msg)` yields a rule key. If policy is disabled, deny.
3. **IAM evaluation** — `AccessController.Can(ctx, ruleKey, resource)` evaluates a CEL rule. Denied anonymous users get `ErrAuthRequired` (401); denied authenticated users get `ErrAccessDenied` (403).

### NamespacedDispatcher (core/runtime/namespaced-dispatcher.go)

Prefix-matching message rewriter. Transforms messages whose name starts with a prefix before passing them downstream.

Used by the blob system: `system:blobs:blob:list` → `system:blobs:{ns}:list` (per-namespace policy resolution).

### AuditDispatcher (core/runtime/access-log-dispatcher.go)

Records every dispatch to the audit log. Captures: operation type (derived from message name suffix), latency, actor identity, auth method, source IP, user agent, resource ID, error code. Persistence failures are non-fatal.

### RecoveryDispatcher (core/runtime/recovery-dispatcher.go)

Catches panics from downstream, logs stack trace, returns error. Prevents handler panics from crashing the process.

---

## Identity & Claims (core/identity/)

### Claims

```go
type Claims struct {
    UserID     string   // Empty = anonymous
    Email      string
    Scopes     []string // Permission scopes, e.g. ["administrator"]
    Operations []string // API key operation allowlist
    TokenType  string   // "session", "api_key", "password_reset", ""
    TokenID    string
    ExpiresAt  int64
}
```

### Context Functions

- `identity.ContextWithClaims(ctx, claims)` — stores claims + sets IAM identity
- `identity.ClaimsFromContext(ctx)` — retrieves claims
- `identity.SystemContext(ctx)` — sets system identity (bypasses IAM checks)

Auth methods tracked in audit: `password`, `api_key`, `service_account`, `none`.

### Session Token System (core/internal/feature/auth/)

HMAC-SHA256 tokens (not JWT):
```
token = base64(version + expiresAt + randomBytes) + ":" + base64(HMAC-SHA256(secret, data))
```

16-byte HMAC truncation for compact tokens (~72 chars).

The `CredentialsProvider`:
```go
CreateSession(userID, ttl)   → (token, info, error)
ValidateSession(token)        → (*SessionInfo, error)
RefreshSession(info)          → (newToken, error)
IssueResetToken(userID)       → (token, error)
ValidateResetToken(token)     → (userID, error)
```

Reset tokens use domain-separated key: `session_secret + ":reset"`.

---

## IAM / Policy System

### Resolution Flow

```
PermissionManager.Resolve(msg) → (ruleKey, enabled)
    ↓
AccessController.Can(ctx, ruleKey, resource) → bool
    ↓
CEL rule evaluation (go-iam)
```

### PermissionManager (core/runtime/permissions.go)

```go
type PermissionManager interface {
    Resolve(msg Message) (ruleKey string, enabled bool, err error)
    ListCapabilities() []CapabilityMetadata
}
```

Two implementations:
- `MapPermissionManager` — static, in-memory (testing)
- `LivePermissionManager` — backed by `LiveCollection[*Policy]`, auto-refreshes from DB

### Rules (CEL expressions)

| Rule | Expression |
|------|-----------|
| `public` | `true` |
| `authenticated` | `identity != null` |
| `password_reset` | `identity != null && identity.token_type == 'password_reset'` |
| `administrator` | `identity != null && 'administrator' in identity.permissions` |

Loaded from Go defaults + `_iam_rule_` collection.

### Policy Bindings

Each operation maps to a rule key:

```
Operation: "system:users:user:list" → Rule: "administrator" (enabled: true)
Operation: "system:auth:session:create" → Rule: "public" (enabled: true)
```

Stored in `_operation_policy_` collection.

---

## Configuration

### Load Order

1. `DefaultConfig()` struct defaults
2. `.env` file
3. `.env.dev` file
4. Environment variables
5. `SetupConfig` struct overrides (highest priority)

### Key Config Fields

| Go Field | Env Var | Default | Description |
|----------|---------|---------|-------------|
| `SessionSecret` | `SESSION_SECRET` | required | HMAC signing key |
| `BcryptCost` | `BCRYPT_COST` | `12` | Password hash cost |
| `SessionTTL` | `SESSION_TTL` | `8h` | Absolute session lifetime |
| `IdleTTL` | `SESSION_IDLE_TTL` | `30m` | Session idle timeout |
| `RefreshTTL` | `SESSION_REFRESH_TTL` | `15m` | Sliding refresh window |
| `DataDir` | `APP_DATA_DIR` | `~/.local/share/<project>` | Data directory |
| `DBPath` | `DB_PATH` | `<data_dir>/<project>.db` | SQLite path |
| `LogPath` | `LOG_PATH` | `<data_dir>/server.log` | Log file |
| `BlobsDir` | `BLOBS_DIR` | `<data_dir>/blobs` | Blob storage |
| `Port` | `PORT` | `8090` | HTTP port (HTTP transport only) |
| `APIPrefix` | `API_PREFIX` | `/api` | HTTP route prefix (HTTP transport only) |

When `APP_DATA_DIR` is explicitly set, the project name is NOT appended (unlike fallback paths).

---

## Bootstrap Flow

On first run, no admin user exists:

1. `SeedAdmin()` creates a random admin email + password (logged to stdout)
2. Only `BootstrapSafe` handlers are registered
3. The CLI `--bootstrap` flag or the `system:auth:bootstrap:password:set` + `system:core:bootstrap:mark` sequence completes setup
4. `OnBootstrapped` fires, restarting all interfaces with full registrations

Tracked in `_seed_` collection:
- `admin_user_id` — admin document ID
- `admin_password_hash` — bootstrapped = true when current hash differs from stored hash

---

## Built-in Features

All features are accessible via the dispatcher directly. When the HTTP transport is enabled, they are also exposed as auto-derived routes.

### 1. Authentication & Sessions (`system:auth:*`)

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:auth:session:create` | Create | `public` |
| `system:auth:user:register` | Create | `administrator` |
| `system:auth:session:delete` | Delete | `authenticated` |
| `system:auth:password:reset` | Create | `authenticated` |
| `system:auth:password:confirm` | Update | `public` |
| `system:auth:session:validate` | Read | `public` (internal) |
| `system:auth:apikey:validate` | Read | `public` (internal) |
| `system:auth:bootstrap:password:set` | Update | `administrator` |

### 2. User Management (`system:users:*`)

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:users:user:register` | Create | `administrator` |
| `system:users:user:list` | Query | `administrator` |
| `system:users:user:get` | Read | `administrator` |
| `system:users:user:update` | Update | `administrator` |
| `system:users:user:delete` | Delete | `administrator` |

Users stored in `_user_` collection. Passwords bcrypt-hashed (cost 12). Password field auto-redacted by sanitization.

### 3. API Keys (`system:apikeys:*`)

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:apikeys:key:list` | Read | `administrator` |
| `system:apikeys:key:create` | Create | `administrator` |
| `system:apikeys:key:get` | Read | `administrator` |
| `system:apikeys:key:update` | Update | `administrator` |
| `system:apikeys:key:delete` | Delete | `administrator` |
| `system:apikeys:key:rotate` | Create | `administrator` |

API keys: bcrypt-hashed, prefix-based lookup, operation allowlisting, expiry, environment scoping.

### 4. Policies & Rules (`system:policies:*`)

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:policies:operation:get` | Read | `administrator` |
| `system:policies:operation:list` | Read | `administrator` |
| `system:policies:rule:validate` | Check | `administrator` |
| `system:policies:rule:create` | Create | `administrator` |
| `system:policies:rule:get` | Read | `administrator` |
| `system:policies:rule:update` | Update | `administrator` |
| `system:policies:rule:list` | Read | `administrator` |
| `system:policies:rule:delete` | Delete | `administrator` |
| `system:policies:policy:create` | Create | `administrator` |
| `system:policies:policy:update` | Update | `administrator` |
| `system:policies:policy:list` | Read | `administrator` |
| `system:policies:reload` | Read | `administrator` |

### 5. Collections & Documents (`system:collections:*`)

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:collections:collection:list` | Read | `administrator` |
| `system:collections:collection:get` | Read | `administrator` |
| `system:collections:collection:create` | Create | `administrator` |
| `system:collections:collection:delete` | Delete | `administrator` |
| `system:collections:document:query` | Query | `authenticated` |
| `system:collections:document:create` | Create | `authenticated` |
| `system:collections:document:get` | Read | `authenticated` |
| `system:collections:document:update` | Update | `authenticated` |
| `system:collections:document:delete` | Delete | `authenticated` |

Generic document store with schema validation and QDSL queries.

### 6. Blob Storage (`system:blobs:*`)

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:blobs:namespace:list` | Query | `administrator` |
| `system:blobs:namespace:create` | Create | `administrator` |
| `system:blobs:namespace:delete` | Delete | `administrator` |
| `system:blobs:{ns}:list` | Query | `administrator` |
| `system:blobs:{ns}:head` | Query | `administrator` |
| `system:blobs:{ns}:upload` | Create | `administrator` |
| `system:blobs:{ns}:download` | Read | `administrator` |
| `system:blobs:{ns}:delete` | Delete | `administrator` |
| `system:blobs:{ns}:update` | Update | `administrator` |

Blob namespaces are dynamic — creating one auto-registers per-namespace handlers and policies.

### 7. Audit

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:core:audit:log` | Create | `authenticated` (internal) |

Auto-logged by `AuditDispatcher` for every dispatch. Stores: actor, operation, resource, latency, status, error, source IP, user agent, trace ID, session ID, auth method.

### 8. Core

| Message | Intent | Default Policy |
|---------|--------|---------------|
| `system:core:health:check` | Read | `public` |
| `system:core:docs:list` | Read | `public` |
| `system:core:capability:list` | Read | `administrator` |
| `system:core:capability:set` | Update | `administrator` |
| `system:core:heartbeat` | Read | `authenticated` |
| `system:core:reset` | Read | `administrator` |
| `system:core:bootstrap:mark` | Create | `public` (internal) |

`system:core:docs:list` enumerates all registered messages with their intents, schemas, and HTTP route derivations (when HTTP transport is used).

---

## Data Sanitization

Pattern-based field masking before any transport serializes:

- **Redacted** (`[REDACTED]`): `password`, `hash`, `secret`, `token`, `api_key`, `credential`
- **Hashed** (SHA-256): `auth`
- **Per-collection**:
  - `_user_`: password redacted, email preserved
  - `_api_key_`: hash redacted

---

## Error Handling

### Error Sentinels (core/runtime/errors.go)

```go
var ErrAccessDenied       // 403 — authenticated user denied
var ErrForbidden          // 403 — business logic denial
var ErrUnauthorized       // 401 — invalid/expired session
var ErrInvalidCredentials // 401 — bad password or key
var ErrAuthRequired       // 401 — anonymous on protected op
var ErrNotFound           // 404
var ErrAlreadyExists      // 409
var ErrValidation         // 400
var ErrInternal           // 500
```

Fluent API:
```go
runtime.ErrNotFound
runtime.ErrValidation.WithOperation("my:feature:action")
runtime.ErrAccessDenied.WithIssues(common.Issues{...})
runtime.ErrInternal.WithCause(err)
```

HTTP status mapping is transport-specific. See the HTTP transport section for the default mapping.

---

## Boot Sequence (core/internal/boot/)

```
hestia.Setup(cfg)
  │
  ├─ runtime.LoadConfig(projectName)
  ├─ cfg.applyTo(conf)
  ├─ Validate SessionSecret
  │
  ├─ boot.BuildApp(conf, opts)
  │   ├─ boot.Create(conf)
  │   │   ├─ NewLoggers()
  │   │   └─ NewPersistenceManager(conf)
  │   ├─ migrations.Apply()
  │   └─ SystemModule.Setup()
  │       ├─ initModels()
  │       ├─ CredentialsProvider
  │       ├─ BlobService
  │       ├─ seedData()
  │       ├─ initPermissions()
  │       ├─ initAccessController()
  │       ├─ registerDocumentHandlers()
  │       ├─ registerBlobHandlers()
  │       └─ collectRegistrations()
  │
  ├─ User migrations (cfg.Migrate)
  ├─ Register user modules
  ├─ BuildInterfaces()
  │   ├─ api.New() (HTTP, optional)
  │   └─ cli.New() (optional)
  ├─ Custom interfaces
  └─ Return *Application
```

---

## Extending Hestia

### Extension Points

```go
type SetupConfig struct {
    Modules          []Module                          // Feature handlers
    Middlewares      []Middleware                      // HTTP-specific middleware
    DispatcherHooks  []func(Dispatcher) Dispatcher     // Transport-agnostic decorators
    Interfaces       []func(Dispatcher) Interface      // Custom transports
    PersistenceFactory func(cfg) (base.Persistence, error)
    Migrate          func(ctx, base.Persistence) error
    OnBootstrapped   func()
    OnReset          func()
    DisableHTTP       bool
    DisableCLI       bool
}
```

### Adding a Feature Module

```go
package mymodule

func Registrations(deps Dependencies) []abstract.MessageRegistration {
    return []abstract.MessageRegistration{
        {
            Name:        "mymodule:greeting:hello:create",
            Handler:     NewHelloHandler(deps.Store),
            Description: "Say hello",
            Enabled:     true,
            Intent:      registration.Create,
        },
    }
}

func NewHelloHandler(store *MyStore) runtime.MessageHandler {
    return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
        return registration.NewDocumentResult(
            data.MustNewDocument(map[string]any{"message": "Hello!"}, ctx),
        ), nil
    }
}
```

Register via `SetupConfig.Modules`.

### Dispatcher Hook

```go
SetupConfig{
    DispatcherHooks: []func(abstract.Dispatcher) abstract.Dispatcher{
        func(next abstract.Dispatcher) abstract.Dispatcher {
            return MyCustomDispatcher{next: next}
        },
    },
}
```

### Custom Transport

```go
type MyTransport struct {
    disp runtime.Dispatcher
}

func (t *MyTransport) Start(bootstrapped bool) { ... }
func (t *MyTransport) Restart(bootstrapped bool) { ... }
func (t *MyTransport) Shutdown(ctx) error { ... }

SetupConfig{
    Interfaces: []func(runtime.Dispatcher) runtime.Interface{
        func(disp runtime.Dispatcher) runtime.Interface {
            return &MyTransport{disp: disp}
        },
    },
}
```

### Feature Checklist

1. Create `core/internal/feature/<name>/`
2. Collection schema: `schema/*.schema.json`
3. Input/output schemas: `schema.go`
4. Model: `model.go`
5. Handlers: `handler.go` (factory functions → `runtime.MessageHandler`)
6. Feature: `feature.go` (`Registrations(Dependencies)`)
7. Default policies: `defaults.go` (`DefaultOperations()`)
8. Wire in `gen_features.go` (operations + registrations)
9. Migration: `core/migrations/`
10. Tests

---

## HTTP Transport (core/interface/api/)

The HTTP transport is one implementation of `runtime.Interface`. It is optional — controlled by `DisableHTTP` (default: enabled).

### Auth Middleware (core/interface/api/middleware.go)

Runs on every HTTP request before the dispatcher chain:

1. **Existing claims** — skip if context already authenticated
2. **Session cookie** — validate HMAC-signed token, enforce TTLs, slide refresh window. Failure clears cookie → 401.
3. **API key header** (`X-Api-Key`) — authenticate via `system:auth:apikey:validate`
4. **Anonymous** — empty `Claims{}`, pass through

**Cookie refresh**:
- Idle timeout (`SESSION_IDLE_TTL`, default 30m)
- Refresh window (`SESSION_REFRESH_TTL`, default 15m)
- Absolute TTL (`SESSION_TTL`, default 8h)
- Session create/delete ops never trigger refresh

### Route Derivation

Routes are derived from message registrations:

```
system:auth:session:create → POST /api/system/auth/session
system:users:user:get     → GET  /api/system/users/user/{user_id}
```

Logic (`core/interface/api/derive.go`):
- Split message name by `:` → `[module, feature, scope, action]`
- Path: `/{apiPrefix}/{module}/{feature}/{scope}`
- Append `/{argName}` for each `ArgDef`
- Append verb suffix: `/query`, `/stream`, `/check`

Verb to HTTP mapping:

| Verb | Method | Path Suffix | Status | Notes |
|------|--------|-------------|--------|-------|
| `Create` | `POST` | — | 201 | `Location` header |
| `Read` | `GET` | — | 200 | |
| `Update` | `PATCH` | — | 200 | |
| `Delete` | `DELETE` | — | 204 | |
| `Query` | `POST` | `/query` | 200 | QDSL body |
| `Stream` | `GET` | `/stream` | 200 | SSE |
| `Check` | `POST` | `/check` | 200 | |

### Error to HTTP Status Mapping

| Error Code | HTTP Status |
|-----------|-------------|
| `ERR_ACCESS_DENIED` | 403 |
| `FORBIDDEN` | 403 |
| `UNAUTHORIZED` | 401 |
| `INVALID_CREDENTIALS` | 401 |
| `AUTH_REQUIRED` | 401 |
| `NOT_FOUND` | 404 |
| `ALREADY_EXISTS` | 409 |
| `VALIDATION_ERROR` | 400 |
| `EMAIL_EXISTS` | 409 |
| `USER_DELETED` | 410 |
| `INTERNAL_ERROR` | 500 |

### Response Format

Success (non-blob):
```json
{
    "data": { ... },
    "metadata": { "timestamp": "...", "request": "...", "page": { ... } }
}
```

Error:
```json
{
    "error": { "code": "NOT_FOUND", "message": "...", "details": "..." },
    "metadata": { "timestamp": "...", "request": "..." }
}
```

Streaming uses SSE (`text/event-stream`):
```
data: {"data": {...}}\n\n
```

### Static Files

When `SetupConfig.StaticFS` is set, the HTTP transport serves static files for paths outside the API prefix. 404s within the API prefix return JSON errors; 404s outside fall through to static directory lookup.

### CORS

Set per-request: `Access-Control-Allow-Origin: {Origin}` (mirrors request), methods, headers, credentials. Preflight `OPTIONS` returns 204.

---

## Testing

```go
import "github.com/asaidimu/hestia/core/internal/testutil"

func TestMyHandler(t *testing.T) {
    p := testutil.NewPersistence(t)  // In-memory go-anansi
    model := mymodel.NewModel(p)
    handler := NewMyHandler(model)

    ctx := identity.ContextWithClaims(context.Background(), &identity.Claims{
        UserID: "test-user",
        Scopes: []string{"administrator"},
    })

    msg := abstract.NewMessage("my:feature:do:thing", ctx,
        data.MustNewDocument(map[string]any{"arguments": map[string]any{"id": "123"}}, ctx))

    result, err := handler(ctx, msg)
}
```

---

## Client Library (TypeScript)

Located at `client/`, published as a separate npm package:
- `client/auth/` — Session management
- `client/blobs/` — Blob storage
- `client/collections/` — Document CRUD + query
- `client/system/` — API keys, capabilities, identity, operations, policies, rules
- `client/core/` — Base client, collection abstraction, errors, pagination, Wails transport
