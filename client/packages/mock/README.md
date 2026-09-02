# @asaidimu/hestia-mock

> IndexedDB-backed mock server and transport for the Hestia TypeScript client SDK — full API parity, persistent offline emulation, and realistic error semantics.

[![Version](https://img.shields.io/npm/v/@asaidimu/hestia-mock)](https://www.npmjs.com/package/@asaidimu/hestia-mock)
[![License](https://img.shields.io/npm/l/@asaidimu/hestia-mock)](LICENSE.md)

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

### Detailed Description

`@asaidimu/hestia-mock` is an in-process server emulator for the [Hestia](https://github.com/asaidimu/hestia) platform. It implements the full Hestia server API using IndexedDB as the storage engine, enabling offline-first development, testing, and prototyping without a running backend.

The mock plugs directly into `HestiaClient` via the `Transport` interface — the same abstraction used by `HttpTransport` (browser) and `WailsTransport` (desktop). Every SDK call — auth, collections, documents, blobs, streams — is served entirely in the browser with no network dependency.

### Key Features

- **Full SDK Compatibility**: Uses the real `HestiaClient` — not a mock or stub — ensuring production-parity behavior
- **IndexedDB Persistence**: Sessions, documents, blobs, and settings survive page reloads
- **Deterministic Seeding**: Pre-populate users, collections, and policies for consistent test/dev environments
- **Realistic Error Semantics**: Throws `SystemError` instances with production error codes (`SYNC-001-NF`, `AUTH-002-UNAUTH`, etc.)
- **SSE Stream Emulation**: In-process event bus simulates server-sent events for notifications, audit logs, and workflows
- **Simulated Latency**: Configurable server delay for realistic UX testing
- **Query Engine**: Full `@asaidimu/query` support — 14 comparison operators, logical groups, multi-field sort, offset pagination

---

## Installation & Setup

### Prerequisites

- Node.js >= 18.0.0 or Bun >= 1.0.0
- `@asaidimu/hestia` (peer dependency)

### Installation

```bash
# Bun
bun add @asaidimu/hestia @asaidimu/hestia-mock

# npm
npm install @asaidimu/hestia @asaidimu/hestia-mock
```

### Node/Bun Test Setup

For Node.js or Bun environments, install `fake-indexeddb` to polyfill IndexedDB:

```bash
bun add -d fake-indexeddb
# or
npm install -D fake-indexeddb
```

Then import the polyfill at the top of your test files:

```ts
import "fake-indexeddb/auto";
```

---

## Usage Documentation

### Basic Usage

```ts
import { createMockHestia } from "@asaidimu/hestia-mock";

const mock = await createMockHestia({
  database: "my-app-mock",
  seed: { email: "admin@test.local", password: "password123", admin: true },
});

// Login — session persists across reloads
await mock.client.auth.login("admin@test.local", "password123");

// Typed collections with full CRUD
const todos = mock.client.collection<{ title: string; done: boolean }>("todos");
await todos.create({ data: { title: "Ship feature", done: false } });
const page = await todos.find();

// Cleanup
await mock.destroy();
```

### À la carte Wiring

```ts
import { MockHestiaServer, IndexedDbTransport, createIdbPersistence } from "@asaidimu/hestia-mock";
import { HestiaClient } from "@asaidimu/hestia";

const server = await MockHestiaServer.create({ database: "app" });
const transport = new IndexedDbTransport(server);

const api = new HestiaClient({
  baseUrl: "idb://hestia-mock",
  transport: transport as any,
  persistence: createIdbPersistence(server),
});
```

### Direct Server Access

Bypass the SDK and talk to the mock server directly:

```ts
await server.dispatch("system:core:health:check");
await server.pushLog({ level: "info", msg: "background job done" });
await server.fireSchedule(scheduleId);   // manual cron tick
await server.tickSchedules();            // fire all due @every schedules
```

### Configuration Options

| Option | Type | Default | Description |
|:---|:---|:---|:---|
| `database` | `string` | `"hestia-mock"` | IndexedDB database name (isolates state between instances) |
| `seed` | `SeedConfig` | `DEFAULT_SEED` | Initial admin user and seed data |
| `reset` | `boolean` | `false` | Wipe the database before seeding |
| `serverOptions.latency` | `number` | `0` | Simulated server latency in milliseconds |
| `serverOptions.updateAvailable` | `string \| null` | `null` | Set to a version string to enable the update flow |

---

## Project Architecture

### Core Components

- **`MockHestiaServer`**: The in-process server emulator. Handles all message dispatch, manages IndexedDB stores, and emits realtime events via an internal `EventBus`
- **`IndexedDbTransport`**: Implements the Hestia `Transport` interface. Routes `dispatch()`, raw path methods, and `openStream()` calls to the mock server
- **`createMockHestia()`**: One-call factory that wires server, transport, persistence, and a real `HestiaClient` together
- **Query Engine**: `executeQuery()` / `evaluateFilter()` / `sortDocs()` — implements the `@asaidimu/query` DSL against IndexedDB collections
- **`EventBus`**: Topic-based pub/sub for SSE stream emulation (notifications, audit logs, workflows)

### Persistence Model

All data lives in a single IndexedDB database (configurable name). Stores include:

| Store | Contents |
|:---|:---|
| Collections | All documents across all collections (system + user-defined) |
| Sessions | Auth sessions (survive reloads) |
| API Keys | Key metadata and hashes |
| Notifications | Per-user notification documents |
| Schedules | Cron schedule definitions |
| Workflows | Definitions, runs, events, registry |
| Blobs | Namespaced blob metadata and binary data |
| Logs | Application logs and audit trail |
| Settings | Key-value configuration |
| Capabilities | Feature flags |
| Seed | Seeding state flags |

### Extension Points

- **`MockServerOptions`**: Hook into server behavior (latency, update state, workflow execution)
- **`EventBus`**: Inject custom events into streams for testing realtime features
- **`seedDatabase()`**: Custom seeding logic beyond the default admin user

---

## Development & Contributing

### Available Scripts

```bash
cd client-mock
bun install

bun run test          # Run tests with vitest (fake-indexeddb)
bun run typecheck     # TypeScript type checking
bun run build         # Build dist/ with tsdown
```

### Testing & Quality Standard

```bash
bun run test
```

Tests use the **real SDK** (`@asaidimu/hestia`) against the mock server, covering auth flows, document CRUD with versioning, pagers, streams, blob uploads, audit entries, and persistence across simulated restarts.

### Contributing Guidelines

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'feat: add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## Troubleshooting & FAQ

### Troubleshooting

- **Issue**: `IndexedDB is not defined` in Node.js/Bun  
  **Solution**: Install and import the fake-indexeddb polyfill: `import "fake-indexeddb/auto"`

- **Issue**: Tests share state between runs  
  **Solution**: Use a unique database name per test: `database: \`test-${Date.now()}\``

- **Issue**: Blob download URLs are not fetchable  
  **Solution**: Use `blobs.namespace(ns).download(key)` instead of `routeUrl()` — the mock does not serve HTTP

### FAQ

- **Q**: Does this replace a real Hestia server?  
  **A**: No. The mock emulates the API surface for development and testing. It does not implement cron daemons, real password hashing (uses SHA-256), or Cedar-style policy evaluation.

- **Q**: Can I use this in Storybook?  
  **A**: Yes. Call `createMockHestia()` in a decorator or preview init. The IndexedDB persistence means state survives HMR updates.

- **Q**: How do I test error paths?  
  **A**: The mock throws real `SystemError` codes. Trigger them by querying non-existent documents (`SYNC-001-NF`), using wrong credentials (`AUTH-003-CRED`), or writing stale versions (`SYNC-003-VC`).

---

## License

Distributed under the [AGPL-3.0](LICENSE.md) License. See `LICENSE.md` for more information.

## Acknowledgments

- [Hestia](https://github.com/asaidimu/hestia) — the platform this mock emulates
- [@asaidimu/query](https://www.npmjs.com/package/@asaidimu/query) — the query DSL implemented by the mock engine
- [fake-indexeddb](https://www.npmjs.com/package/fake-indexeddb) — IndexedDB polyfill for Node.js/Bun testing
