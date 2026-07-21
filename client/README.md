# @asaidimu/hestia

TypeScript client SDK for the [Hestia](https://github.com/asaidimu/hestia) platform — a lightweight, modular backend framework.

## Features

- **Auth** — login, register, refresh, logout, session management with auto-refresh on 401
- **Collections** — generic CRUD over any named collection via `HestiaCollection<T>`
- **API Keys** — create, list, get, update, rotate, delete API keys
- **Policies** — manage policy operations, rules, validation, and reload
- **Audit Logs** — query and filter audit entries
- **Blobs** — upload, download, list, and manage blob storage namespaces
- **Capabilities** — query available capabilities
- **Reactive Paging** — observable paginated data with resize/navigate/subscribe

## Installation

```bash
npm install @asaidimu/hestia
# or
bun add @asaidimu/hestia
```

## Quick Start

```ts
import { HestiaClient, WailsTransport } from "@asaidimu/hestia"

// HTTP mode (browser/node)
const api = new HestiaClient({ baseUrl: "http://localhost:8070" })

// Wails desktop mode — same API, different transport
const desktop = new HestiaClient({
  transport: new WailsTransport(),
})
// Now api.auth.login(), api.users.find(), etc. call Go directly

// Login as admin
const { token, user } = await api.auth.login("admin@test.local", "password123")
console.log("Logged in as", user.email)

// List users
const { data: users } = await api.users.find()
console.log(`Found ${users.length} users`)

// Create an API key
const key = await api.keys.create({ name: "CI/CD Deploy Key" })
console.log("API key:", key.key)

// Reactive pager
const pager = api.users.page()
pager.subscribe((page) => console.log("Page:", page.page.number))
await pager.resize(25, 1)
```

## API

### `HestiaClient`

The main entry point. Construct with a `baseUrl` pointing to a Hestia server.

| Property | Type | Description |
|---|---|---|
| `auth` | `HestiaAuth` | Authentication (login, register, refresh, logout) |
| `users` | `HestiaUsers` | User management |
| `keys` | `HestiaKeyStore` | API key management |
| `policies` | `HestiaPolicies` | Policy operations and rules |
| `logs` | `HestiaLogs` | Audit log queries |
| `collections` | `HestiaCollections` | Generic collection metadata |
| `blobs` | `HestiaBlobClient` | Blob (file) storage |
| `capabilities` | `HestiaCapabilities` | Available capabilities |
| `client` | `Transport` | Low-level transport (HTTP or Wails) |

### Auth

```ts
// Login — stores tokens and identity in the reactive store
const result = await api.auth.login(email, password)

// Register a new user (requires admin session)
const user = await api.auth.register(email, password, name)

// Refresh tokens
const pair = await api.auth.refresh(refreshToken)

// Logout — clears stored tokens
await api.auth.logout()

// Health check (public endpoint, no auth required)
const health = await api.auth.health()
```

### Collections

```ts
// Get a typed collection handle
const docs = api.collection<MyType>("my_collection")

// CRUD operations
const list = await docs.find(query?)
const item = await docs.read(id)
const created = await docs.create(document)
const updated = await docs.update(id, partial)
await docs.delete(id)
```

### Reactive Paging

```ts
const pager = api.users.page()

// Subscribe to page changes
const unsub = pager.subscribe((page) => render(page.data))

// Navigate, resize, sort, filter
await pager.navigate(2)
await pager.resize(50, 1)
await pager.sort({ field: "email", order: "asc" })
await pager.filter({ email: { $like: "%@example.com" } })

// Manual refresh
await pager.refresh()

// Cleanup
unsub()
```

### Auto-Refresh

The client transparently refreshes expired JWT access tokens on 401 responses. Concurrent requests that receive 401 are deduplicated — only one refresh call is made. API-key requests (with `X-API-Key` header) are excluded from auto-refresh.


## Development

```bash
# Install dependencies
bun install

# Run tests (requires a running test-server on :8070)
./test-server &
bun test
```

## License

MIT
