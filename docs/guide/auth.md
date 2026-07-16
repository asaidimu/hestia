# Authentication

Hestia uses JWT-based authentication with access/refresh token pairs.

## Login

```
POST /system/auth/session
```

```json
{
  "email": "admin@test.local",
  "password": "password123"
}
```

Returns an access token (short-lived, 15 min) and a refresh token (long-lived, 7 days).

## Token Refresh

```
PATCH /system/auth/session
```

```json
{
  "refresh_token": "<refresh-token>"
}
```

Returns new access and refresh token pairs. The old refresh token is rotated.

## Auto-Refresh (Client SDK)

The TypeScript client automatically refreshes the access token when it receives a `401` response. Concurrent requests that fail with `401` are deduplicated — only one refresh call is made.

```ts
// All of this happens transparently:
const session = await api.auth.login("admin@test.local", "password123")

// The SDK stores tokens and auto-refreshes on 401
const { data: users } = await api.users.find()
```

## Logout

```
DELETE /system/auth/session
```

Revokes the current session token.

## Registration

```
POST /system/auth/user
```

```json
{
  "email": "newuser@example.com",
  "password": "SecurePass1",
  "name": "New User"
}
```

Requires an authenticated admin session.

## Token Validation

```
GET /system/auth/session    — Validate session token
GET /system/auth/token      — Validate JWT access token
```

## API Key Auth

```
GET /system/auth/apikey
Header: X-API-Key <api-key>
```

Authenticate using an API key instead of a JWT session.
