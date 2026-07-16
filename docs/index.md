# Hestia

Lightweight, modular backend framework built with Go and a TypeScript client SDK.

## Features

- **Modular** — Plug-and-play modules: auth, users, API keys, blobs, collections, policies, audit
- **Auto-documented** — Every registered handler exposes its schema via `GET /system/core/docs`
- **Policy-based access** — Fine-grained access control via CEL policy rules
- **Reactive client** — TypeScript SDK with auto-refresh JWT handling and observable pagination
- **Blob storage** — Namespaced file storage with metadata
- **Dynamic collections** — Schema-less document CRUD for any data model

## Architecture

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│  TypeScript   │────▶│   HTTP API      │────▶│   Modules    │
│  Client SDK   │     │  (Fiber/Chi)    │     │  Auth/Users  │
│               │     │                 │     │  API Keys    │
│  @asaidimu/   │     │  /system/*      │     │  Blobs       │
│  hestia       │◀────│  /data/*        │◀────│  Collections │
└──────────────┘     └─────────────────┘     │  Policies    │
                                              │  Audit       │
                                              └──────┬───────┘
                                                     │
                                              ┌──────▼───────┐
                                              │  Persistence  │
                                              │  (SQLite/     │
                                              │   Postgres)   │
                                              └──────────────┘
```

## Quick Start

```bash
# Start the server
go run ./cmd/server

# Install the client SDK
npm install @asaidimu/hestia
```

```ts
import { HestiaClient } from "@asaidimu/hestia"

const api = new HestiaClient({ baseUrl: "http://localhost:8070" })

const { token, user } = await api.auth.login("admin@test.local", "password123")
console.log(`Logged in as ${user.email}`)

const { data: users } = await api.users.find()
console.log(`Found ${users.length} users`)
```

## Next Steps

- [Getting Started](/getting-started) — set up a server and make your first API call
- [API Reference](/api/core) — full endpoint documentation
- [Client SDK](/client/) — TypeScript client reference
