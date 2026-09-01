# @asaidimu/hestia-mock

An **IndexedDB-backed mock server + transport** for the
[@asaidimu/hestia](https://github.com/asaidimu/hestia) TypeScript client SDK.
Drop it into `HestiaClient` and the entire platform — auth, collections,
users, API keys, policies, notifications, schedules, workflows, blobs, audit
logs, app logs, settings, updates, capabilities — runs **entirely in the
browser, offline, persisted in IndexedDB**.

```ts
import { createMockHestia } from "@asaidimu/hestia-mock";

const mock = await createMockHestia();
await mock.client.auth.login("admin@test.local", "password123");

const todos = mock.client.collection<{ title: string }>("todos");
await todos.create({ data: { title: "offline-first" } });
```

## Why

Hestia's client SDK is transport-agnostic: every call funnels through a
`Transport` interface (`dispatch(name, input)`, `openStream`, raw path
methods). Two transports ship today — `HttpTransport` (browser) and
`WailsTransport` (desktop). This package adds a third: an in-process server
emulator whose storage engine is IndexedDB.

Use it for:

- **Offline-first development** — build the whole UI before the backend exists
- **Storybook / demos / tests** — no server, no network, deterministic seeds
- **Prototyping** — sessions, documents, files, and realtime streams all
  survive page reloads

## Install

```bash
bun add @asaidimu/hestia @asaidimu/hestia-mock
# or
npm install @asaidimu/hestia @asaidimu/hestia-mock
```

> In this repository the package resolves `@asaidimu/hestia` via
> `file:../client`. When published to npm, it uses the released SDK.

## Quick start

### One-call wiring

```ts
import { createMockHestia } from "@asaidimu/hestia-mock";

const mock = await createMockHestia({
  database: "my-app-mock",            // IndexedDB name (isolates state)
  seed: { email: "me@x.dev", password: "s3cret", admin: true },
  serverOptions: {
    latency: 40,                      // simulated server latency (ms)
    updateAvailable: null,            // set to a version to exercise updates
  },
});

const api = mock.client;              // a REAL HestiaClient
await api.auth.login("me@x.dev", "s3cret");

// Typed collections with the full query DSL
const products = api.collection<{ title: string; price: number }>("products");
await products.create({ data: { title: "Widget", price: 9.99 } });
const page = await products.find({
  filters: { field: "price", operator: "lt", value: 10 },
  sort: [{ field: "title", direction: "asc" }],
  pagination: { type: "offset", offset: 0, limit: 20 },
} as never);

// Reactive pager — driven by the same query engine
const pager = products.page();
pager.subscribe((p) => console.log(p.page.number, p.data.length));
await pager.resize(25, 1);

// Realtime streams (SSE emulation over an in-process event bus)
const ctrl = api.notifications.stream({
  onMessage: (n) => console.log("notification:", n.subject),
});
```

### À la carte pieces

```ts
import { MockHestiaServer, IndexedDbTransport, createIdbPersistence } from "@asaidimu/hestia-mock";
import { HestiaClient } from "@asaidimu/hestia";

const server = await MockHestiaServer.create({ database: "app" });
const transport = new IndexedDbTransport(server);
const api = new HestiaClient({
  baseUrl: "idb://hestia-mock",
  transport,
  persistence: createIdbPersistence<{ identity: any }>(server),
});
```

You can also talk to the server directly (bypassing the SDK):

```ts
await server.dispatch("system:core:health:check");
await server.pushLog({ level: "info", msg: "background job done" });
await server.fireSchedule(scheduleId);       // manual cron tick
await server.tickSchedules();                // fire due `@every` schedules
```

## What is emulated

| Subsystem | Behavior |
|---|---|
| **Auth** | Login/logout with persisted sessions (survive reloads), registration, password reset (the mock returns the token — real servers email it), bootstrap via `X-API-Key`, session revocation on password change/delete |
| **Collections** | Schema-registered collection metadata; auto-registers schema-less collections on first document write (dev convenience, noted in README) |
| **Documents** | Full CRUD with `_id_` + `_metadata_` (checksum, created/updated, version), optimistic locking (`SYNC-003-VC` on stale version), the complete `@asaidimu/query` engine |
| **Users** | Admin/user gating, soft-delete tombstones, password change with session teardown, sanitized reads (password hash never leaves the server) |
| **API keys** | Create (secret shown once), list/get/update/rotate/delete; `X-API-Key` header authenticates as the key owner |
| **Policies** | Seeded operation policies + IAM rules; the policies pager works through the generic document query route, exactly like production |
| **Notifications** | Per-user create/list/read/unread-count + live SSE stream; admin-gated creation |
| **Schedules** | Persistent cron schedule CRUD; `@every` auto-ticking (opt-in), manual `fireSchedule()`, Go-template resolution for `{{ .schedule._id }}` / `{{ .now }}` |
| **Workflows** | Persistent definitions, stub executor that emits a full timeline (`pipeline:started` → node events → `pipeline:success`), run inspection (meta/outcome/events/store), registry + handles endpoints, SSE run streaming with replay |
| **Blobs** | Namespaces, direct upload, resumable protocol (`begin` → `chunk` → `progress` → `complete`/`abort`), binary round-trip via IndexedDB, download as real `Blob`, prefix listing, stats/verify/compact |
| **Audit logs** | Every dispatch writes a structured `_audit_log_` entry (actor, operation, resource, status, latency); query + admin-gated export/stream; capped at ~500 recent entries |
| **App logs** | Structured log store + `system:logs:list` query (level/from/to/search/paging) + level-filtered stream |
| **Settings** | Key/value documents with version bumps |
| **Updates** | Full state machine (status/changelog/check/stage/apply/discard); set `updateAvailable` to exercise it |
| **Core** | Health (with bootstrap flag), heartbeat (session check), capabilities, docs list, admin reset (wipe + re-seed) |

### Error semantics

The mock throws real `SystemError` instances with the codes the SDK checks:

| Code | Meaning |
|---|---|
| `SYNC-001-NF` | Not found (collections, documents, users…) |
| `NOT_FOUND` | Blob-store not found (matches the SDK's blob client) |
| `SYNC-002-DUP` | Duplicate key |
| `SYNC-003-VC` | Optimistic-locking version conflict |
| `AUTH-002-UNAUTH` | Unauthenticated (401 → triggers `onUnauthorized`, clearing identity) |
| `AUTH-001-DENIED` | Admin-only route as non-admin |
| `AUTH-003-CRED` | Invalid credentials |
| `NO_ACTIVE_SESSION` | Logout with no session (logout stays idempotent) |

## Query engine

Implements the standard tier of `@asaidimu/query`:

- **All 14 comparison operators** — `eq`, `neq`, `lt`, `lte`, `gt`, `gte`,
  `in`, `nin`, `contains`, `ncontains`, `startswith`, `endswith`,
  `exists`, `nexists`
- **Logical groups** — `and`, `or`, `not`, `nor`, `xor`, nested arbitrarily
- **Multi-field sort** with per-field direction (missing values last)
- **Offset pagination** with full `metadata.page` (`number/size/count/total/pages`)
- Dotted paths (`meta.rating`) and array indices (`items.0`)

Cursor pagination degrades gracefully (natural order + limit).

## Persistence model

Everything lives in one IndexedDB database (`hestia-mock` by default):

- documents (all collections, including system ones), sessions, API keys,
  notifications, schedules, workflow definitions/runs/events, namespaces +
  blob bytes, app logs, audit trail, settings, capabilities, seed flags
- the client's auth identity is hydrated from the same database through a
  `SimplePersistence` implementation, so **login state survives reloads**

Databases are namespaced: pass distinct `database` names to isolate state
(tests, storybook, different users in the same browser).

## Testing your app with it

```ts
import "fake-indexeddb/auto";          // Node/bun: install fake-indexeddb
import { createMockHestia } from "@asaidimu/hestia-mock";

const mock = await createMockHestia({
  database: `test-${Date.now()}`,      // fresh state per run
  seed: { email: "admin@test.local", password: "password123", admin: true },
});

// run assertions against mock.client — a real HestiaClient
await mock.destroy();
```

This repository's own suite (`test/`) runs the **real SDK** against the mock
across every subsystem — 69 tests covering auth flows, document CRUD with
versioning, pagers, streams, staged blob uploads, audit entries, and full
server-restart persistence.

## Differences from a real server

- No network: `routeUrl()` returns `idb://…` URLs (blob download URLs are not
  fetchable; use `blobs.namespace(ns).download(key)`)
- No cron daemon: schedules fire via `server.tickSchedules()` /
  `server.fireSchedule(id)`; only `@every` expressions auto-tick
- Workflow execution is a stub (it records runs and emits realistic timeline
  events; it does not interpret node semantics)
- Password hashing is salted SHA-256 (not bcrypt/argon2)
- Document writes to unknown collections auto-register them (real servers
  reject; the mock prioritizes dev ergonomics)
- Policy rules are stored/validated trivially — no Cedar-style evaluation

## API

| Export | Description |
|---|---|
| `createMockHestia(options?)` | Server + transport + wired `HestiaClient`; returns `{ client, transport, server, destroy }` |
| `IndexedDbTransport` | `Transport` implementation (named dispatch, path methods, `openStream`) |
| `MockHestiaServer` | The server emulator (`dispatch`, `pushLog`, `fireSchedule`, `tickSchedules`, `writeAuditEntry`, `close`) |
| `createIdbPersistence(server)` | `SimplePersistence` for the client auth store |
| `executeQuery / evaluateFilter / sortDocs` | Query engine (exported for inspection/testing) |
| `EventBus, Topics` | Realtime topic bus (inject events into streams) |
| `mockErrors` | SystemError factories with HTTP status mapping |
| `openMockDatabase, wipeAllStores, STORES` | Low-level schema utilities |
| `seedDatabase, DEFAULT_SEED` | Seeding primitives |

## Development

```bash
cd client-mock
bun install
bun run test        # vitest (fake-indexeddb) — real SDK vs mock
bun run typecheck
bun run build       # tsdown → dist/
bun run test/smoke-dist.ts   # smoke the built artifact
```

## License

MIT
