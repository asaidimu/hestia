# Test Plan — ERP Template Server

## Convention

All tests go in `*_test.go` files **alongside the source** (same directory).
Use **black-box testing** via `package foo_test` so tests depend only on the exported API.

```
internal/module/system/feature/auth/
  handler.go          → package auth
  handler_test.go     → package auth_test   ← black-box
```

Only use `package foo` (white-box) when unexported internals genuinely need direct testing.
Both can coexist in the same directory.

---

## Existing Tests (7 files, 32 functions)

| File | Tests |
|---|---|
| `internal/app/database_test.go` | In-memory DB, file-based DB, idempotent close |
| `internal/core/permissions_test.go` | SecureDispatcher: anonymous denied, admin allowed, public allowed, system bypass |
| `internal/module/system/services/jwt_test.go` | Access/refresh/reset token generation & validation, expired & wrong signature |
| `internal/module/system/services/blocklist_test.go` | Blocklist CRUD, multiple entries, duplicate JTI |
| `internal/utility/auth/password_test.go` | Password hash & verify |
| `internal/utility/persistest/pagination_test.go` | Pagination totals, IncludeTotal default |
| `internal/orchestrator/api/register_test.go` | buildDoc, serializeResponse, route derivation, dispatcher registration |

---

## Phase 1 — Feature Handler Tests (highest priority)

All use black-box (`package auth_test` / etc.). Each handler constructor tested in isolation by passing mock dependencies.

### `internal/module/system/feature/auth/handler_test.go` (package auth_test)

| Test Function | What It Tests | Mock Strategy |
|---|---|---|
| `TestNewCreateSessionHandler_ValidCredentials` | Login with valid email/password returns tokens + sanitized user | Mock `UserModel.GetByEmail` returns user doc with bcrypt hash |
| `TestNewCreateSessionHandler_InvalidEmail` | Unknown email → `"invalid email or password"` | `GetByEmail` returns error |
| `TestNewCreateSessionHandler_WrongPassword` | Known email, bad password → `"invalid email or password"` | `auth.CheckPassword` fails |
| `TestNewRegisterHandler_Success` | Register with email/password/name returns sanitized user | Mock `UserModel.Register` returns user doc |
| `TestNewRegisterHandler_DuplicateEmail` | Registering existing email propagates model error | `Register` returns error |
| `TestNewRefreshSessionHandler_ValidJWT` | Refresh with valid JWT returns new token pair | Mock `JWTService.ValidateToken` returns valid claims |
| `TestNewRefreshSessionHandler_ValidSession` | Refresh with session token works via `SessionService` | Token fails JWT, `SessionService.Validate` succeeds |
| `TestNewRefreshSessionHandler_Blocklisted` | Refresh with blocklisted token → `"refresh token has been revoked"` | `IsBlocklisted` returns true |
| `TestNewRefreshSessionHandler_InvalidToken` | Bad refresh token → `"invalid refresh token"` | Both JWT and session validation fail |
| `TestNewDeleteSessionHandler_BlocklistsBoth` | Access + refresh tokens both blocklisted | Claims in context, refresh in body |
| `TestNewDeleteSessionHandler_NoClaims` | Logout without claims still succeeds (graceful) | No claims in context |
| `TestNewPasswordResetHandler_UserFound` | Reset always returns empty success (never reveals existence) | `GetByEmail` succeeds or fails — same result |
| `TestNewPasswordConfirmHandler_ValidResetToken` | Confirm with valid password_reset token changes password | Claims in context with `TokenType: "password_reset"` |
| `TestNewPasswordConfirmHandler_MissingAuth` | No claims → `"authorization required"` | No claims in context |
| `TestNewPasswordConfirmHandler_WrongTokenType` | Non-reset token → `"invalid token type"` | Claims present but wrong type |
| `TestNewSetBootstrapPasswordHandler_Success` | Valid caller_id + password + email updates password + email | Mock `ChangePassword` + `Update` |
| `TestNewSetBootstrapPasswordHandler_WrongCaller` | Wrong caller_id → `"only the seeded admin"` | caller_id mismatch |
| `TestNewValidateTokenHandler_Valid` | Token validation returns decoded claims | `ValidateToken` returns claims |
| `TestNewValidateTokenHandler_Invalid` | Bad token propagates error | `ValidateToken` returns error |
| `TestNewCheckBlocklistHandler_Blocklisted` | Token ID is blocklisted → true | `IsBlocklisted` returns true |
| `TestNewCheckBlocklistHandler_Clean` | Token ID not blocklisted → false | `IsBlocklisted` returns false |
| `TestNewCheckBlocklistHandler_NilBlocklist` | No blocklist service → false | blocklist arg is nil |
| `TestNewValidateAPIKeyHandler_Valid` | Valid API key returns claims | Mock `keyAuth.Authenticate` returns claims |
| `TestNewValidateAPIKeyHandler_Invalid` | Invalid API key → error | `Authenticate` returns error |

### `internal/module/system/feature/users/handler_test.go` (package users_test)

| Test Function | What It Tests |
|---|---|
| `TestNewGetUserHandler_Found` | Existing user → sanitized doc |
| `TestNewGetUserHandler_NotFound` | Missing user → error from model |
| `TestNewUpdateUserHandler_UpdatesFields` | Updates name/email/scopes/verified, returns updated user |
| `TestNewUpdateUserHandler_NoFields` | Empty body → `"no fields to update"` |
| `TestNewChangePasswordHandler_CorrectCurrent` | Valid current+new password → password changed |
| `TestNewChangePasswordHandler_WrongCurrent` | Wrong current → `ErrInvalidCredentials` |
| `TestNewChangePasswordHandler_DeletedUser` | Soft-deleted user → `ErrUserDeleted` |
| `TestNewDeleteUserHandler_Soft` | Default (no modifier) → `SoftDelete` called |
| `TestNewDeleteUserHandler_Hard` | `permanent=true` → `HardDelete` called |
| `TestNewUserCreateDocumentHandler_Valid` | Valid body → user registered + sanitized |
| `TestNewUserCreateDocumentHandler_MissingFields` | Missing email/password/name → validation error |
| `TestNewUserUpdateDocumentHandler_Valid` | Valid body → user updated + sanitized |
| `TestNewUserUpdateDocumentHandler_EmptyBody` | Empty body → error |

### `internal/module/system/feature/apikeys/handler_test.go` (package apikeys_test)

| Test Function | What It Tests |
|---|---|
| `TestNewListAPIKeysHandler_OwnKeys` | Lists keys for current user (from context) |
| `TestNewListAPIKeysHandler_AdminSpecifiesUser` | Lists keys for given user_id arg |
| `TestNewGetAPIKeyHandler_Found` | Returns key with prefix+suffix hint |
| `TestNewGetAPIKeyHandler_NotFound` | Not found → error |
| `TestNewCreateAPIKeyHandler_Success` | Creates key with body fields, returns document with raw key |
| `TestNewUpdateAPIKeyHandler_Success` | Updates name/status/scopes etc. |
| `TestNewDeleteAPIKeyHandler_Success` | Deletes key, returns empty |
| `TestNewRotateAPIKeyHandler_Success` | Rotates key, returns new key |

### `internal/module/system/feature/core/handler_test.go` (package core_test)

| Test Function | What It Tests |
|---|---|
| `TestNewSystemStatusHandler_Bootstrapped` | `bootstrapped()` returns true |
| `TestNewSystemStatusHandler_NotBootstrapped` | `bootstrapped()` returns false |
| `TestNewDocumentationHandler_ReturnsAll` | Returns docs for all registrations with correct HTTP methods |
| `TestNewLogAccessHandler` | Access log entry inserted via model `Insert` |
| `TestNewMarkBootstrappedHandler` | Calls `onBootstrapped` callback |
| `TestNewResetHandler` | Calls `onReset` callback |
| `TestExtractAccessLogEntry` | Extracts fields from input document |

### `internal/module/system/feature/policies/handler_test.go` (package policies_test)

| Test Function | What It Tests |
|---|---|
| `TestNewUpsertOperationHandler` | Upserts operation with name + body fields |
| `TestNewDeleteOperationHandler` | Deletes operation by name |
| `TestNewUpsertRuleHandler_Simple` | Upserts simple rule |
| `TestNewUpsertRuleHandler_WithSubRules` | Upserts rule with nested rules tree |
| `TestNewDeleteRuleHandler_Unprotected` | Deletes unprotected rule |
| `TestNewDeleteRuleHandler_Protected` | Protected rule → `"cannot be deleted"` |
| `TestNewGetOperationHandler` | Gets operation by name |
| `TestNewGetRuleHandler` | Gets rule by name |
| `TestNewReloadPoliciesHandler` | Reloads + compiles + loads rules, returns counts |

### `internal/module/system/feature/collections/handler_test.go` (package collections_test)

| Test Function | What It Tests |
|---|---|
| `TestIsSystemCollection_True` | `_user_` → true |
| `TestIsSystemCollection_False` | `custom_collection` → false |
| `TestNewCollectionCreateHandler_Valid` | Creates collection from JSON schema body |
| `TestNewCollectionCreateHandler_SystemName` | Reserved name → error |
| `TestNewCollectionCreateHandler_Duplicate` | Existing collection → error |
| `TestNewDocumentCreateHandler_Valid` | Creates doc in collection |
| `TestNewDocumentCreateHandler_EmptyBody` | Empty → `DOCUMENT_REQUIRED` |
| `TestNewDocumentGetHandler_Found` | Gets doc by ID |
| `TestNewDocumentGetHandler_NotFound` | Not found → empty result |
| `TestNewDocumentUpdateHandler_Valid` | Updates doc by ID |
| `TestNewCollectionListHandler` | Lists non-system collections |
| `TestNewCollectionQueryHandler_WithQuery` | Reads with query DSL |

### `internal/module/system/feature/audit/handler_test.go` (package audit_test)

| Test Function | What It Tests |
|---|---|
| `TestLogQueryHandler_WithQuery` | Queries access log with DSL |
| `TestLogQueryHandler_DefaultPagination` | No query → default pagination applied |

### `internal/module/system/feature/blobs/handler_test.go` (package blobs_test)

| Test Function | What It Tests |
|---|---|
| `TestNewListNamespacesHandler` | Lists all namespaces |
| `TestNewCreateNamespaceHandler_Valid` | Creates namespace with display_name |
| `TestNewCreateNamespaceHandler_MissingID` | No ns ID → validation error |
| `TestNewDeleteNamespaceHandler` | Deletes namespace |
| `TestNewListBlobsHandler` | Lists blobs with prefix/limit |
| `TestNewHeadBlobHandler` | Returns blob metadata |
| `TestNewUploadBlobHandler` | Uploads blob from body |
| `TestNewDownloadBlobHandler` | Downloads blob as registration.Blob |
| `TestNewDeleteBlobHandler` | Deletes blob |

---

## Phase 2 — Model Layer Tests

Black-box (`package models_test`), using `persistest.NewPersistence(t)`.

### `internal/module/system/models/user_test.go`

| Test Function |
|---|
| `TestUserModel_Register` |
| `TestUserModel_GetByEmail` |
| `TestUserModel_GetByID` |
| `TestUserModel_Update` |
| `TestUserModel_ChangePassword` |
| `TestUserModel_SoftDelete` |
| `TestUserModel_HardDelete` |
| `TestUserModel_IsDeleted` |
| `TestUserModel_List` |

### `internal/module/system/models/api_key_test.go`

| Test Function |
|---|
| `TestAPIKeyModel_Generate` |
| `TestAPIKeyModel_Create` |
| `TestAPIKeyModel_List` |
| `TestAPIKeyModel_Get` |
| `TestAPIKeyModel_Update` |
| `TestAPIKeyModel_Delete` |
| `TestAPIKeyModel_Rotate` |
| `TestAPIKeyModel_ValidateKey` |

### `internal/module/system/models/policy_test.go`

| Test Function |
|---|
| `TestPolicyModel_UpsertOperation` |
| `TestPolicyModel_GetOperation` |
| `TestPolicyModel_DeleteOperation` |
| `TestPolicyModel_ListOperations` |
| `TestPolicyModel_UpsertRule` |
| `TestPolicyModel_GetRule` |
| `TestPolicyModel_DeleteRule` |
| `TestPolicyModel_ListRules` |

### `internal/module/system/models/access_log_test.go`

| Test Function |
|---|
| `TestAccessLogModel_Insert` |

### `internal/module/system/models/seed_test.go`

| Test Function |
|---|
| `TestSeedModel_SetAndGet` |

---

## Phase 3 — Core Dispatcher & Orchestrator Tests

### `internal/core/dispatcher_test.go` (package core_test)

| Test Function |
|---|
| `TestLocalDispatcher_RegisterAndSend` |
| `TestLocalDispatcher_ListHandlers` |
| `TestLocalDispatcher_SetHandlerEnabled` |
| `TestLocalDispatcher_DeleteHandler` |
| `TestNamespacedDispatcher_PrefixesMessages` |
| `TestNewMessage_CreatesMessage` |

### `internal/orchestrator/api/orchestrator_test.go` (package api_test)

| Test Function |
|---|
| `TestOrchestrator_Start` |
| `TestOrchestrator_Shutdown` |
| `TestOrchestrator_Restart` |

---

## Phase 4 — Utility & Transport Tests

### `internal/utility/jwt/jwt_test.go` (package jwt_test)

(Note: `internal/module/system/services/jwt_test.go` already tests the wrapper.
This tests the raw `utility/jwt.Service`.)

| Test Function |
|---|
| `TestService_GenerateAndValidateAccessToken` |
| `TestService_GenerateRefreshToken` |
| `TestService_GenerateResetToken` |
| `TestService_ValidateToken_Expired` |
| `TestService_ValidateToken_BadSignature` |

### `internal/utility/session/session_test.go` (package session_test)

| Test Function |
|---|
| `TestService_Generate` |
| `TestService_Validate_Valid` |
| `TestService_Validate_Expired` |
| `TestService_Validate_Tampered` |

### `internal/transport/http/transport_test.go` (package httpserver_test)

| Test Function |
|---|
| `TestHTTPTransport_Handle` |
| `TestHTTPTransport_CORSMiddleware` |
| `TestHTTPTransport_CorrelationID` |

### `internal/transport/transport_test.go` (package transport_test)

| Test Function |
|---|
| `TestCodeToStatus` |
| `TestSystemErrorToStatus` |

### `internal/utility/blobs/service_test.go` (package blobs_test)

| Test Function |
|---|
| `TestService_CreateNamespace` |
| `TestService_DeleteNamespace` |
| `TestService_ListNamespaces` |

---

## Phase 5 — Integration & E2E Tests

### Integration: Auth flow (in `internal/module/system/feature/auth/`)

Full in-process test with real in-memory Anansi persistence:

```
TestAuthFlow_RegisterLoginRefreshLogout:
  seed admin → register user → login → access token works →
  refresh → old token blocklisted → new tokens work → logout →
  both tokens blocklisted → access denied
```

### Integration: Policy enforcement

```
TestPolicyFlow_CRUDRules:
  create operation → create rule → query operation → verify access granted →
  delete rule → verify access denied
```

### Integration: Collection lifecycle

```
TestCollectionLifecycle_CreateQueryDelete:
  create collection → insert doc → query → update doc → delete doc →
  delete collection → collection gone
```

### E2E: HTTP API (`cmd/test-server` or `httptest.Server` in `tests/` dir)

Full HTTP tests using the app builder with in-memory DB:

```
TestE2E_AuthEndpoints            — POST /api/auth/login, /api/auth/register, etc.
TestE2E_UserEndpoints            — GET /api/users/{id}, PATCH, DELETE
TestE2E_APIKeyEndpoints          — CRUD + rotate
TestE2E_PolicyEndpoints          — CRUD operations + rules
TestE2E_CollectionEndpoints      — CRUD collections + documents
TestE2E_HealthEndpoint           — GET /api/health
TestE2E_CapabilitiesEndpoint     — GET /api/admin/capabilities
TestE2E_DocumentationEndpoint    — GET /api/admin/documentation
TestE2E_AuthMiddleware           — missing token, expired token, bad token, API key auth
TestE2E_CORSHeaders              — OPTIONS requests
TestE2E_BootstrapFlow            — PATCH /api/bootstrap/password
```

---

## CI Pipeline (`server/.github/workflows/test.yaml`)

Replace `echo test` with:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.26'
    check-latest: true

- run: go vet ./...
- run: go test ./... -v -count=1 -race -coverprofile=coverage.out
- run: go tool cover -func=coverage.out | tail -1
```

---

## Test Helper: `internal/utility/webtest/`

Create a new package `webtest` with:

- `NewInMemoryApp(t) (*app.Application, *system.SystemModule)` — builds full app with in-memory SQLite
- `NewRequest(method, path string, body any) *http.Request` — builds test requests
- `ServeRequest(app, req) *httptest.ResponseRecorder` — serves through the API orchestrator handler

This lets E2E tests be written concisely:

```go
a, mod := webtest.NewInMemoryApp(t)
orch, _ := app.BuildOrchestrators(a, mod, "http")
go orch.Start(false)
defer orch.Shutdown(context.Background())

resp := webtest.ServeRequest(t, a, "POST", "/api/auth/login", map[string]any{
    "email": "admin@test.com", "password": "password",
})
assert.Equal(t, 200, resp.Code)
```

---

## Priority Order Summary

| Order | What | Est. Tests | Est. Effort |
|---|---|---|---|
| 1 | Auth handler tests | 24 | Medium |
| 2 | Users handler tests | 12 | Medium |
| 3 | API keys handler tests | 7 | Small |
| 4 | Core handler tests | 7 | Small |
| 5 | Policies handler tests | 9 | Medium |
| 6 | Collections handler tests | 11 | Medium |
| 7 | Audit + Blobs handler tests | 10 | Small |
| 8 | Model layer tests | 25 | Medium |
| 9 | Core dispatcher tests | 5 | Small |
| 10 | Utility tests (jwt, session, blob) | 10 | Small |
| 11 | Transport tests | 5 | Small |
| 12 | Integration tests (auth flow, policy, collection) | 3 | Large |
| 13 | E2E HTTP tests | 12 | Large |
| 14 | CI pipeline + linter | — | Small |
| 15 | `webtest` helper package | — | Small |
