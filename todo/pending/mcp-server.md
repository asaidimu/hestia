# MCP server: agent access to a live Hestia instance

**Status:** proposed — no implementation started.
**Opened:** 2026-11-13
**Owner:** unassigned
**Related:** `docs/reviews/2026-08-21-second-pass-architecture-review.md` §5,
`todo/active/secure-workflow-runtime.md` (agent/workflow direction)

## Goal

Give agents a first-class, low-friction way to **observe, operate on, and
remediate a running Hestia instance** over the Model Context Protocol, without
shelling out to `curl` and without bypassing Hestia's authorization, audit,
rate-limit, tenancy, or sanitization layers.

Two audiences, one engine:

1. **Coding / ops agents** (OpenCode, Claude Code, custom loops) — diagnose and
   fix the instance itself.
2. **Business-operations agents** — assist staff by reading and mutating
   business collections, running declared workflows, and answering questions
   about customers, orders, invoices, notifications.

The immediate motivating workflow for (1): an agent notices a symptom (e.g.
"hourly notification cleanup never runs"), then within one MCP session can:

1. read health, logs, audit, scheduler jobs, and schedules;
2. discover the relevant operation and its input schema;
3. invoke read-only diagnostics repeatedly;
4. apply a safe remediation (create/update/run a schedule);
5. verify the fix from logs/audit/jobs.

For (2) the shape is the same but the guardrails differ: domain-scoped tools,
the human's permissions, projected/redacted output, and workflow-mediated
writes. See "Two consumer classes" below.

## Why this is a good fit

- `abstract.MessageRegistration` (`core/abstract/module.go`) already carries
  `Name`, `Description`, `Intent`, `Internal`, `BootstrapSafe`, `Input.Schema`,
  and `Output *definition.Schema` — an MCP tool manifest already exists in
  memory for every operation.
- `core/interface/http/register.go` already implements the canonical
  build-document → validate → dispatch → serialize pipeline. MCP should reuse
  it, not fork it.
- `application.Registrations` is available in `core/hestia.go`, so an embedded
  interface can see every operation with no network hop.
- The following diagnostic operations are already exposed over HTTP today:
  `system:core:health:check`, `system:core:capability:list`,
  `system:core:docs:list`, `system:logs:list`, `system:audit:log:export`,
  `system:scheduler:job:list`, schedules CRUD, and
  `system:collections:document:query`.
- API-key auth (`X-API-Key`), the per-key operation allowlist enforced by
  `core/runtime/secure-dispatcher.go`, and the audit chain already exist.
  MCP must plug into these, never around them.
- The business surface already exists as operations too: collection CRUD and
  QDSL queries (`core/system/collections`), workflow definitions and runs
  (`core/system/workflows`), blobs, notifications, users, and audit. Nothing
  business-specific needs to be invented — only curated and governed.

## Two consumer classes (scope is wider than dev diagnostics)

The motivating incident is a coding agent debugging a scheduler, but the same
interface serves a second, likely larger audience: **agents assisting business
operations** — answering customer questions, looking up orders/documents,
preparing invoices, triggering fulfilment workflows, reconciling records.

Both consumers want the same thing (a governed RPC surface over registrations),
so the engine should be built once. What differs is everything *around* it:

| Concern | Dev / ops agent | Business-operations agent |
|---|---|---|
| Caller identity | Service agent, API key in env | Human-delegated (OAuth) or a per-domain service key |
| Scope of tools | Whole system, diagnostics + remediation | One business domain, mostly reads, tightly gated writes |
| Operation names | `system:scheduler:job:list` is fine | Needs business language ("find customer by email") |
| Blast radius | Can restart/repair the instance | Can move money, contact customers, mutate records |
| Confirmation | `confirm: true` is usually enough | Human approval, ideally via a workflow approval gate |
| Output shaping | Raw, verbose logs/schema OK | Paginated, redacted, PII-aware, bounded |
| Audit need | Trace a fix | Prove who authorized what, and why |
| Client | OpenCode / Claude Code | Chat apps, internal assistants, third-party agent platforms |

Consequences for this plan:

- Keep the generic registration-backed engine as the core — it already covers
  business collections, workflows, blobs and notifications with no new code.
- Add a **surface profile** (see D2) so a deployment can expose a
  business-facing subset with human-readable tool names and copy, instead of
  dumping internal namespaces on a sales assistant.
- Support **human-delegated auth** (OAuth / forwarded session) alongside API
  keys (see D3), so business tools reflect the *user's* permissions and the
  audit entry names the human, not just the agent.
- Prefer **workflows as the mutating action layer** (see D2/D4): a business
  agent should call one declared, capability-scoped workflow rather than
  compose a dozen raw CRUD calls. `todo/active/secure-workflow-runtime.md`
  already plans tenant-scoped principals, manifests, approval and budgets —
  those are exactly the guardrails business writes need.
- Tenancy is real: `core/runtime/tenant-dispatcher.go` derives the tenant from
  the caller's context. Tool lists and cached servers must be keyed by
  tenant **and** user as well as credentials, or one tenant's tool schema can
  leak into another's session.
- Non-developer clients often cannot run a local binary, and several only
  speak stdio or require public HTTPS. This raises the priority of the OAuth
  path and of the Phase 2 stdio bridge rather than lowering it.

## Corrections to the earlier review

`docs/reviews/2026-08-21-second-pass-architecture-review.md` §5 suggests
implementing MCP as an `abstract.Transport` alongside `HTTPTransport`. That
does not fit the current code: `http.Interface` hardcodes
`newHTTPTransport()` in `core/interface/http/server.go`, and `Transport` is an
HTTP-shaped route registrar (`Handle(pattern, handler)`), not an MCP session
abstraction.

**Recommended shape instead:** a sibling `core/interface/mcp` implementing
`runtime.Interface` (`Start/Restart/Shutdown`), owning its own HTTP listener
and the official MCP `StreamableHTTPHandler`. It resolves the same
registrations, dispatcher chain, and auth providers that `http.Interface`
uses. Extract the shared input/serialization helpers out of
`core/interface/http` so the two interfaces cannot drift.

## Design decisions (recommended, confirm before implementation)

### D1 — Transport

- **Primary:** embedded **Streamable HTTP** endpoint, opt-in via config.
  - Default address `127.0.0.1:8071`, path `/mcp`; never bind `0.0.0.0` by
    default.
  - Opt-in: `HESTIA_MCP_ENABLED=true` / `runtime.Config.MCP.Enabled`.
  - Disabled by default, including in the test server until explicitly enabled.
- **Secondary (only if needed):** a `hestia-mcp` stdio bridge that proxies to
  the Streamable HTTP endpoint for clients without remote-MCP support. Do not
  make it the primary path.

### D2 — Tool surface

Do **not** expose every registration as a tool (Hestia has hundreds; tool
schemas alone would blow up agent context). Use a hybrid:

- **Meta-tools (always present):**
  - `hestia_search_operations` — query by text, intent, tag, namespace.
  - `hestia_describe_operation` — name → description, intent, risk, input
    JSON Schema, output schema, policy rule, HTTP path.
  - `hestia_call_operation` — name + `arguments` + `payload` + optional
    `reason`, `dry_run`, `confirm`.
- **Curated first-class tools (thin wrappers over existing operations):**
  - `hestia_health`
  - `hestia_query_logs` (`system:logs:list`)
  - `hestia_query_audit` (`system:audit:log:export` / collections query)
  - `hestia_scheduler_status` (jobs + schedules in one compact view)
  - `hestia_list_schedules` / `hestia_get_schedule`
  - `hestia_create_schedule` / `hestia_update_schedule` / `hestia_delete_schedule`
  - `hestia_query_collection`
  - `hestia_run_schedule_now` (new operation, see Phase 1.5)
  - `hestia_run_scheduler_job` (new operation, see Phase 1.5)
  - `hestia_cleanup_notifications` (internal; only when maintenance is enabled)
- **Optional generated tools:** allow an explicit allowlist of operations to be
  emitted as dedicated typed tools (config file), for hot paths where the
  meta-tool indirection is too slow. Off by default.
- **Surface profiles:** the deployment selects what a given MCP surface exposes
  (e.g. `dev-diagnostics`, `ops`, `business:<domain>`). A profile declares
  allowed namespaces/operations, optional aliases with human-readable
  descriptions, preferred mutating path (workflow vs CRUD), and field
  projections. Profiles are config, not code, so a business deployment can
  ship `find_customer`, `get_order`, `create_invoice` without touching Go.
- **Workflow-first mutations:** for business profiles, prefer exposing
  `system:workflows:runtime:run` (and curated `hestia_run_workflow
  <name>`) over raw `document:create`/`update`. Workflows declare their
  capability manifest, so the mutation is reviewable, bounded and auditable in
  one place instead of an improvised sequence of CRUD calls.
- **Prompts and resources matter more for business use:** MCP prompts can
  encode procedures ("reconcile month-end", "handle a refund request") and
  resources can expose reference data (catalog, policy text). Treat both as
  part of the surface, not an afterthought.

MCP tool names must match `[A-Za-z0-9_-]`; normalize `system:logs:list` to
`system_logs_list` (keep a name ↔ registration map; never re-derive).

Tool annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`,
`openWorldHint`) must be set from `Intent`, policy rule, and internal/risk
metadata. Destructive tools are denied by default unless explicitly enabled.

### D3 — Auth and authorization

- Authenticate every MCP request with the existing API-key path:
  `X-API-Key` or `Authorization: Bearer <key>` → `auth.APIKeyAuthenticator` →
  `abstract.Claims` + `runtime.ContextWithAuditIdentity` (match
  `core/interface/http/middleware.go` exactly, including `TokenType`
  → actor/auth method mapping).
- Also support **human-delegated auth** for business assistants: a session
  token or OAuth access token resolved by the same credential provider, so the
  effective identity is the end user. Business tools must be filtered by that
  user's permissions, not the agent's.
- Build MCP tools per caller (cache by tenant + user + permission fingerprint +
  protocol version) and filter `tools/list` by the caller's actual
  permissions: `PermissionManager.Resolve` + `IAM AccessController.Can` with
  the caller's context. A forbidden operation must not appear in the list and
  must be rejected at call time with `isError: true`. Never cache a tool list
  across tenants or users.
- Reuse `secure-dispatcher.go`'s per-key `operations` allowlist; no second
  authorization model.
- Do not use a root key for the agent by default. Document creating a dedicated
  `agent-diagnostics` key whose `operations` allowlist covers diagnostics +
  schedules + collections, and excludes `system:core:reset`,
  `system:users:*` writes, `system:auth:*`, and API-key management. For
  business agents the same rule applies per domain: one key/profile per
  business function, never one omnipotent key shared by every assistant.
- Make the actor explicit in audit for agent-originated calls: keep the human
  (or service) as the actor and record the agent client/tool/`reason` as
  properties, or add a dedicated agent actor type. Existing `ActorType` values
  are `user`, `service`, `system`, `anonymous` (`core/runtime/audit`); agents
  should not silently pass as `user`.
- Bind MCP sessions to the authenticated user (the SDK supports
  `TokenInfo.UserID` session binding) and reject session-id reuse across keys.

### D4 — Safety defaults

- `Internal: true` operations are hidden unless `MCP.AllowInternal` is set.
  Even then, expose only a curated maintenance allowlist, never everything.
- Destructive calls require `confirm: true`; if the client supports MCP
  elicitation, also ask for interactive confirmation. Add `dry_run` to
  `hestia_call_operation` returning planned arguments/operation without
  dispatch.
- Per-call timeout, max concurrency, max page size (e.g. default 50, max 500),
  and a hard response-size cap.
- All tool outputs pass through the existing sanitization path; never return
  `_api_key`, password hashes, session tokens, or blob bytes inline. For
  business profiles add **field projection**: the profile declares which fields
  are returned, so a support assistant sees a customer's contact details but
  not internal notes, margins, or unrelated PII.
- High-risk business writes delegate approval instead of relying on
  `confirm: true`: create the pending workflow/approval record and return its
  id, letting a human approve through the workflow runtime
  (`todo/active/secure-workflow-runtime.md`). `confirm` remains for
  low-risk/ops actions.
- Idempotency: mutating `hestia_call_operation` accepts a client `idempotency_key`
  so an agent retrying after a timeout cannot double-create an order, invoice
  or customer. Replay returns the original result.
- Loopback bind is the default; validate `Origin` when bound to a non-loopback
  address and document the reverse-proxy/TLS expectation.

### D5 — SDK and protocol

- Use the official Go SDK: `github.com/modelcontextprotocol/go-sdk`
  (v1.7.0+ supports spec `2026-07-28`, `2025-11-25`, `2025-06-18`).
- Prefer `2026-07-28` (stateless, `server/discover`) with legacy fallback.
  OpenCode supports `protocol: "auto"` for probing.
- `StreamableHTTPHandler` + `StreamableHTTPOptions{Stateless: true}` for the
  embedded endpoint; per-request server construction/caching is the hook for
  per-caller tool filtering.
- Pin the SDK version in `go.mod`; record the supported protocol versions.

## Phase 0 — spike and fixtures

- [ ] Pin `github.com/modelcontextprotocol/go-sdk` in `go.mod` (verify Go
      1.27 compatibility, `StreamableHTTPHandler` + `AddTool` dynamic schema
      support, and `mcp.CallToolResult.StructuredContent`).
- [ ] Capture a real `system:core:docs:list` response from the local prod
      instance (`:8090`) as a fixture; use it to prove schema translation and
      tool-name normalization against real data.
- [ ] Spike: in-memory MCP client/server (`mcp.NewInMemoryTransports`) calling
      `system:core:health:check` through the runtime dispatcher. Exit criterion:
      a tool call reaches the handler with valid claims and returns structured
      JSON.
- [ ] Confirm whether `definition.Schema.AsMap()` is sufficient as JSON Schema
      for MCP, or whether a converter is needed (envelopes, `payload`/`arguments`,
      bytes/blobs, nullable/required mapping).

## Phase 1 — embedded MCP interface (Streamable HTTP)

### Package and wiring

- [ ] New `core/interface/mcp/` package:
  - `config.go` — `Config{Enabled, Addr, Path, AllowInternal, MaxPageSize,
    MaxResultBytes, AllowedOperations, DestructiveEnabled, Profiles}`.
  - `server.go` — implements `runtime.Interface`; owns `httptest`-friendly
    `net/http.Server` + `mcp.NewStreamableHTTPHandler`.
  - `auth.go` — credential extraction/validation + audit identity (API key and
    human-delegated token). Reuse `auth.APIKeyAuthenticator` and the session
    credential provider resolved from DI; do not duplicate credential logic.
  - `profile.go` — load surface profiles: operation allowlist, aliases +
    human-readable descriptions, workflow-first mode, output field projection.
  - `registry.go` — snapshot `[]abstract.MessageRegistration`; filter/annotate;
    prebuild `document.DocumentPool` per registration (as
    `core/interface/http/register.go` does).
  - `schema.go` — registration → MCP tool schema; flatten
    `payload`/`arguments`/`modifiers`; preserve required/optional.
  - `invoke.go` — build `abstract.Request{Body, PathParams, Query, Headers}`,
    call `http.BuildInputDocument` (exported, currently in
    `core/interface/http/doc.go`), validate, dispatch via the chained
    dispatcher, serialize `abstract.Result` to MCP content/structured content.
  - `resources.go` — optional Phase 1.5: `hestia://health`,
    `hestia://operations`, `hestia://operations/{name}`,
    `hestia://scheduler/jobs`.
  - `tools_curated.go` — the first-class tools from D2.
- [ ] Extract shared helpers from `core/interface/http`:
  - input envelope/building (`BuildInputDocument` is already exported; decide
    whether to move it to `core/runtime/dispatch` to avoid an MCP→HTTP import);
  - result serialization (`serializeResponse`/`serializeOutputField` are
    unexported) → add a neutral `ResultToJSON`/`SerializeResult` helper usable
    by HTTP, CLI, and MCP;
  - `addAuditContext` (unexported in `middleware.go`) → shared auth helper.
- [ ] `core/runtime/config.go` — add `MCPConfig` with env overrides
      (`HESTIA_MCP_ENABLED`, `HESTIA_MCP_ADDR`, `HESTIA_MCP_PATH`,
      `HESTIA_MCP_ALLOW_INTERNAL`, `HESTIA_MCP_ALLOW_DESTRUCTIVE`).
- [ ] `core/hestia.go` — add `NewMCPInterface(cfg)` mirroring
      `NewHTTPInterface`; resolve `*auth.APIKeyAuthenticator`,
      `ReloadablePermissionManager`, `iam.AccessController`, registrations, and
      chained dispatcher. Wire into `Setup` only when enabled.
- [ ] `cmd/test-server/main.go` — enable MCP on a free port when
      `MCP_PORT` is set, so e2e tests can connect.
- [ ] Graceful lifecycle: `Start` after boot; `Restart` on bootstrap/state
      changes; `Shutdown` with a short context; no goroutine leaks (use
      `httptest`/`go test -race`).

### Tool behavior

- [ ] `tools/list` returns only operations the caller may invoke **and** that
      the active surface profile allows; omit `Internal` unless allowed; set
      annotations; include a short server instruction string ("search →
      describe → call; read-only first; confirm destructive").
- [ ] Business profiles expose aliases and human-readable copy, and can hide
      the meta-tools entirely so a non-technical assistant sees only a small,
      domain-shaped toolset.
- [ ] Workflow-first profiles route mutations to `hestia_run_workflow` /
      `system:workflows:runtime:run`; raw document writes are absent unless the
      profile opts in.
- [ ] `hestia_describe_operation` returns one schema, not the whole catalog.
- [ ] `hestia_call_operation`:
  - validates name against registrations and caller permissions;
  - maps `arguments` → `PathParams`, `payload` → `Body` JSON,
    `modifiers` → `Query`;
  - validates with `dispatch.ValidateInputDocument` before dispatch;
  - maps `common.SystemError` codes to MCP `isError: true` with code/message/
    issues in structured content;
  - honors `dry_run`, `confirm`, and `idempotency_key`;
  - applies the profile's output field projection before serializing.
- [ ] Pagination: every list tool accepts `limit`/`offset` (or QDSL pagination),
      enforces default/max, and returns `has_more`.
- [ ] Streaming intent (`abstract.Stream`) is **not** callable via `tools/call`;
      return a clear error pointing at bounded `query_logs`/`query_audit` tools.
      Live streaming can later use MCP resources/subscriptions.
- [ ] Blobs: return metadata + a resource URI, not base64 by default.
- [ ] Result metadata (rate-limit remaining, timings) included in structured
      content where useful.

### Tests

- [ ] Unit: schema conversion (envelope, required fields, unknown types),
      name normalization/collision, per-key tool filtering, internal filtering,
      missing/invalid key rejection, audit context mapping.
- [ ] Unit: `dry_run` performs no dispatch; `confirm` required for destructive;
      forbidden operation denied before dispatch; idempotency replay returns the
      original result without a second dispatch.
- [ ] Unit: profile filtering (aliases resolve to the right registration,
      disallowed operation absent, meta-tools hidden for business profiles) and
      output field projection strips excluded fields.
- [ ] Integration: boot `hestia.Setup` with MCP enabled, connect an official
      SDK client over `httptest`, list tools, call health/logs/scheduler status.
- [ ] Integration (business path): a `business:*` profile with a scoped user
      lists only domain tools, runs a workflow, and cannot see another
      tenant's or user's tools after the cache is warmed by a different caller.
- [ ] Regression scenario: with the current scheduler bug fixture, an agent can
      list jobs, see cleanup never dispatched in logs, run the schedule now,
      and verify a new dispatch log line.
- [ ] `go test -race ./core/interface/mcp/...` and
      `go test ./core/system/...` green.

## Phase 1.5 — operations the agent actually needs

These are useful beyond MCP and close the current gap between "observe" and
"fix":

- [ ] `system:scheduler:job:run` — run a registered scheduler job immediately
      (root rule). Input: `name`. Emits the same dispatch path/logging as a
      normal tick.
- [ ] `system:schedules:schedule:run` — run one persisted schedule immediately
      (root rule). Input: `id`. Useful for verifying a schedule without waiting
      for `@every 1h`.
- [ ] Optional: `system:scheduler:job:pause` / `resume` for maintenance.
- [ ] Curated MCP tools for the above; annotate as mutating/destructive.
- [ ] Add policies (`core/system/operations/policies.go`,
      `core/system/schedules/policies.go`), registrations, and docs for every
      new operation. Keep the anti-escalation invariants: only root can run
      arbitrary jobs; a non-root schedule can only run its owner's schedule.

## Phase 2 — stdio bridge and agent integration

- [ ] `cmd/hestia-mcp/main.go` (or `hestia mcp serve --stdio`): an stdio MCP
      server that connects to the remote Streamable HTTP endpoint using the
      official Go client SDK and forwards tools/resources/prompts. Config:
      `HESTIA_MCP_URL` (default `http://127.0.0.1:8090/mcp`),
      `HESTIA_API_KEY`. No Hestia boot, no duplicated tool logic.
- [ ] Add an OpenCode example to docs and (optionally) the project config:

  ```jsonc
  {
    "$schema": "https://opencode.ai/config.json",
    "mcp": {
      "servers": {
        "hestia": {
          "type": "remote",
          "url": "http://127.0.0.1:8090/mcp",
          "oauth": false,
          "protocol": "auto",
          "headers": { "X-API-Key": "{env:HESTIA_API_KEY}" },
          "timeout": { "execution": 120000 }
        }
      }
    }
  }
  ```

  CLI equivalent:
  `opencode mcp add hestia --url http://127.0.0.1:8090/mcp --header 'X-API-Key={env:HESTIA_API_KEY}'`
- [ ] Add `skills/references/mcp-agent-playbook.md` (or extend
      `skills/SKILL.md`): standard diagnosis loop, safe-remediation rules,
      which tools to use for scheduler/log/notification problems, and when to
      prefer `dry_run`.
- [ ] Add `skills/references/mcp-business-playbook.md`: how to design a
      business surface profile, choose workflow-first mutations, and handle
      PII/approval correctly.
- [ ] Add `docs/guide/mcp.md`: setup, auth (API key and delegated user), tool
      catalog, surface profiles, security model, OpenCode/Claude/Cursor
      examples, troubleshooting (`opencode mcp list`).

## Phase 3 — hardening and observability

- [ ] Per-key rate limits and concurrency caps; reuse
      `core/runtime/ratestore` rather than a new limiter.
- [ ] Audit every MCP call with actor, operation, argument fingerprint (not
      secrets), duration, outcome; link to trace id from MCP `_meta` /
      `ai.opencode/sessionID` without treating it as auth.
- [ ] Metrics: calls, errors, latency, denied operations, result sizes.
- [ ] Negative tests: key with no operations (must see empty tool list), deleted/
      disabled user, expired key, forbidden operation, internal op when disabled,
      oversized results, malformed payloads, tool-list cache warmed by a
      different tenant/user, and a business profile requesting a
      non-projected field.
- [ ] Origin/CSRF checks when non-loopback; document TLS termination.
- [ ] Kill switch and a startup log line stating bind address, auth mode,
      active profile, internal/destructive exposure, and protocol versions.
- [ ] Verify no secrets leak: `_api_key`, `_user`, sessions, blob contents.
- [ ] Add MCP e2e to CI (or a make target) against `cmd/test-server`.

## Open questions (answer before Phase 1)

1. **Enablement:** opt-in per deployment (recommended) or automatically on
   `serve`?
2. **Consumers:** dev/ops diagnostics only for now (smaller first cut), or both
   dev/ops and business-operations surfaces from day one? The engine is shared
   either way; the difference is whether profiles and delegated auth are in
   Phase 1 or a later phase.
3. **Agent identity:** dedicated least-privilege API key (recommended) or the
   existing root key? Which operations must be excluded? For business agents,
   one profile key per function — confirm the domain split.
4. **Delegated auth:** is human-delegated OAuth required by the business
   assistants you have in mind, or is a per-domain service key acceptable for
   the first iteration?
5. **Tool surface:** hybrid meta-tools + curated tools (recommended) or one
   tool per operation? Should business profiles hide meta-tools entirely?
6. **Mutating path:** workflow-first (recommended) or allow raw document CRUD in
   business profiles? If CRUD is allowed, which collections?
7. **Approval:** delegate high-risk writes to the workflow approval runtime
   (recommended), or rely on `confirm`/elicitation only for the first cut?
8. **Internal ops:** expose a curated maintenance allowlist (recommended) or
   none? Is `hestia_cleanup_notifications` wanted?
9. **Confirmation:** `confirm: true` only, or also MCP elicitation for
   destructive calls?
10. **stdio:** needed for any client you use, or is remote Streamable HTTP
    enough? Chat-app clients may force this earlier than expected.
11. **Protocol:** target `2026-07-28` with `auto` fallback, or stay on
    `2025-11-25` for widest client compatibility?

## Acceptance criteria

- An agent can connect with a scoped API key and `opencode mcp list` shows
  `connected`.
- `tools/list` contains only operations that key may invoke; internal operations
  are absent unless explicitly enabled.
- `hestia_health`, `hestia_query_logs`, `hestia_scheduler_status`,
  `hestia_query_audit`, and collection queries work and return compact,
  structured, sanitized JSON.
- A create/update/delete schedule call is dispatched through the full chain and
  produces an audit entry.
- A forbidden call returns `isError: true` with `ERR_ACCESS_DENIED`; it is not
  dispatched.
- `hestia_run_schedule_now` can reproduce/verify a scheduler tick without waiting
  for the cron window.
- A business profile exposes only its domain tools, with human-readable names,
  for the connecting user's permissions and tenant; a different user/tenant
  never sees them.
- A business mutation runs through a workflow or an allowed operation, produces
  an audit entry naming the human actor plus the agent client, and is safe to
  retry with the same `idempotency_key`.
- `go test -race ./core/interface/mcp/...` and the MCP e2e target are green.
- Docs cover setup, security posture, tool catalog, surface profiles, and the
  OpenCode config.

## References

- Existing registration model: `core/abstract/module.go`
  (`MessageRegistration`, `Input`, `Capability`).
- HTTP install pipeline: `core/interface/http/register.go`
  (`installRegistration`, `buildDoc`, `serializeResponse`).
- Docs/capability introspection: `core/system/operations/service.go`
  (`DocsList`, `ListCapabilities`).
- Logs: `core/system/logs/service.go`; audit:
  `core/system/collections` audit query + `system:audit:log:export`.
- Scheduler/schedules: `core/runtime/scheduler/scheduler.go`,
  `core/system/schedules/liveschedule.go`, `core/system/schedules/service.go`.
- Auth/audit: `core/system/auth/adapters.go`,
  `core/interface/http/middleware.go`, `core/runtime/secure-dispatcher.go`.
- Earlier review: `docs/reviews/2026-08-21-second-pass-architecture-review.md` §5.
- MCP Go SDK: <https://github.com/modelcontextprotocol/go-sdk>.
- MCP spec (2026-07-28): <https://modelcontextprotocol.io/specification/2026-07-28>.
- OpenCode MCP config: <https://opencode.ai/docs/mcp-servers/>.