# Add API Key Authentication to TypeScript Client

## Context

The TypeScript client (`@asaidimu/hestia`) only supports cookie-based session authentication (login via email/password). Server-side applications need to authenticate using API keys sent via the `X-Api-Key` header, which the Go server already supports but the TS client has no mechanism to use.

## Approach

Add an `apiKey` option to `HestiaConfig`. When provided, the client automatically injects `X-Api-Key` on every request via `defaultHeaders` and skips session-based auth state management (identity store, heartbeat, 401 clearing).

## Changes

### 1. `client/packages/core/core/client.ts` — `HttpTransport`

Add `defaultHeaders` support to the constructor and inject them into every request.

```typescript
constructor(
  private baseUrl: string,
  private apiPrefix: string,
  private onAuthStateChanged?: () => void,
  private defaultHeaders?: Record<string, string>,
)
```

In `createNetworkClient` call, pass `defaultHeaders`:

```typescript
this.raw = createNetworkClient({
  baseUrl,
  defaultResponseType: "json",
  defaultBodyType: "json",
  ...(this.defaultHeaders ? { defaultHeaders: this.defaultHeaders } : {}),
});
```

### 2. `client/packages/core/container.ts` — `HestiaConfig` + `HestiaClient`

Add `apiKey?: string` to `HestiaConfig`. When set:

- Build `defaultHeaders: { "X-Api-Key": apiKey }` and pass to `HttpTransport`
- Skip `tokenProvider` wiring (no identity store needed)
- Skip heartbeat setup
- `authenticated()` returns `true` if `apiKey` is set (trust the key; server rejects invalid ones)
- `onAuthStateChange` callbacks still fire on 401 from the server (key could be revoked)

### 3. `client/packages/core/system/auth/store.ts` — `HestiaAuth`

No changes needed. `login()` still works if called, but with API key auth it's unnecessary. The `bootstrap()` method already sends its own `X-API-Key` header per-call, which takes precedence over `defaultHeaders`.

### 4. `client/packages/core/core/wails-transport.ts` — `WailsTransport`

No changes. WailsTransport delegates to `HttpTransport` for HTTP fallback paths, so it inherits the default headers automatically.

## Usage

```typescript
// Server-side: authenticate with API key
const api = new HestiaClient({
  baseUrl: "http://localhost:8090",
  apiKey: "ak_abc123...",
});

// All requests automatically include X-Api-Key header
const { data: users } = await api.users.find();
const docs = api.collection<MyType>("articles");
await docs.create({ title: "Hello" });
```

## Files to Modify

1. `client/packages/core/core/client.ts` — add `defaultHeaders` param to `HttpTransport`
2. `client/packages/core/container.ts` — add `apiKey` to config, wire default headers

## Verification

1. `npm run build` in `client/` (or `turbo build`)
2. `npm run test` in `client/packages/core`
3. Manual: create a client with `apiKey`, call an endpoint, verify `X-Api-Key` header is sent
