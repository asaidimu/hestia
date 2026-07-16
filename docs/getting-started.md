# Getting Started

## Running the Server

```bash
git clone https://github.com/asaidimu/hestia.git
cd hestia
go run ./cmd/test-server
```

The server starts on port `8070` by default and creates an admin user automatically.

## Installing the Client SDK

```bash
npm install @asaidimu/hestia
# or
bun add @asaidimu/hestia
```

## First API Call

```ts
import { HestiaClient } from "@asaidimu/hestia"

const api = new HestiaClient({ baseUrl: "http://localhost:8070" })

// Health check — no auth required
const health = await api.auth.health()
console.log("Server bootstrapped:", health.bootstrapped)

// Login with default credentials
const session = await api.auth.login("admin@test.local", "password123")
console.log("Access token:", session.token.access)

// List users
const { data: users } = await api.users.find()
console.log(`Found ${users.length} users`)
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8070` | HTTP server port |
| `DATA_DIR` | `./data` | Persistence directory |
| `JWT_SECRET` | auto-generated | JWT signing key |
| `ADMIN_EMAIL` | `admin@test.local` | Initial admin email |
| `ADMIN_PASSWORD` | `password123` | Initial admin password |

## Client Configuration

```ts
const api = new HestiaClient({
  baseUrl: "http://localhost:8070",
  // Optional: persistent token store (localStorage, IndexedDB, etc.)
  persistence: myTokenStore,
})
```
