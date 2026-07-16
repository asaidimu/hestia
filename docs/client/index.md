# TypeScript Client SDK

**Package:** `@asaidimu/hestia`

## Installation

```bash
npm install @asaidimu/hestia
```

## `HestiaClient`

Main entry point — configure once and access all modules.

```ts
import { HestiaClient } from "@asaidimu/hestia"

const api = new HestiaClient({ baseUrl: "http://localhost:8070" })
```

### Constructor Options

| Option | Type | Default | Description |
|---|---|---|---|
| `baseUrl` | `string` | required | Server base URL |
| `persistence` | `SimplePersistence<AuthState>` | `undefined` | Optional store for persisting auth tokens |

### Properties

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
| `client` | `HestiaNetworkClient` | Low-level HTTP client |
| `store` | `ReactiveDataStore<AuthState>` | Reactive auth state store |

## Core Types

```ts
interface Document<T> {
  _id_: string
  _metadata_: {
    created: string
    updated: string
    version: number
    checksum: string
    trace_id: string
  }
  [key: string]: any
}

interface Page<T> {
  data: Document<T>[]
  loading: boolean
  page: PaginationInfo
}

interface PaginationInfo {
  number: number
  size: number
  count: number
  total: number
  pages: number
}

type QueryDSL<T> = {
  pagination?: { type: "offset"; offset: number; limit: number }
  sort?: SortConfiguration<T>[]
  filters?: QueryFilter<T>
}
```

## `HestiaNetworkClient`

Low-level HTTP client with auto-refresh.

```ts
// Available methods
client.get<T>(path, options?)
client.post<T>(path, body?, options?)
client.patch<T>(path, body?, options?)
client.put<T>(path, body?, options?)
client.delete<T>(path, body?, options?)

// Auto-refresh on 401 — transparent JWT token refresh
```

## Reactive Paging

```ts
const pager = api.users.page()

// Subscribe to page changes
const unsub = pager.subscribe((page) => console.log(page.data))

// Navigate, resize
await pager.navigate(2)
await pager.resize(50, 1)

// Get current snapshot
const snapshot = pager.page()

// Cleanup
unsub()
```
