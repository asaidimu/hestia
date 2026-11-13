# Secure Workflow Runtime (adopted refactor spec)

**Source:** root `refactor-spec.md` (draft, retired at adoption — this document is
authoritative).
**Status:** Active — Phase 0 landed; Phase 1 (durable runtime ownership) next.
**Adopted:** 2026-11-13 after validation against the codebase and Hermes v1.3.0.
**Phase 0 landed:** 2026-11-13 (`hermes v1.3.0` pinned, provider rewired,
`go test -race ./core/...` green).
**Targets:** Hestia `v1.16.x`, `github.com/asaidimu/hermes v1.3.0` (pinned),
`go-iam/v2 v2.0.0`, `go-anansi/v8 v8.10.3`.

**Goal:** make every registered workflow a tenant-scoped principal with a declared
capability manifest, durable execution, per-run budgets, tier isolation, and an
enforcement path for custom nodes that reuses Hestia's existing security
primitives. Hermes remains the process; `core/system/workflows` is the API
gateway and policy layer.

> Framing: **A workflow is a tenant-scoped application. Hermes is its process.
> `core/system/workflows` is the API gateway and policy layer that every call
> from that process must pass through.**

---

## 0. Validation summary (2026-11-13)

The draft was written against an assumed architecture that differs from the
repository in several load-bearing ways. The corrections below are part of the
adopted design, not optional polish.

| # | Draft claim | Verified reality | Consequence |
|---|---|---|---|
| V1 | Security primitives live in `core/security` | There is no `core/security` package. Middleware lives in `core/runtime` (`secure-dispatcher.go`, `rate-limit.go`, `throttle.go`, `tenant-dispatcher.go`, `audit/`, `ratestore/`, `permissions.go`). The chain is assembled in `core/system/module.go` (`SystemModule.DispatcherChain`). | Additive exports/registrations belong in `core/runtime` + `core/system`; do not invent a new package. |
| V2 | Workflows has a `Module` with `Start`/`Stop` in `core/system/workflows/module.go` | No such file or type. `WorkflowsService` is a service package; runtime is created in `ProviderSet.InitModels` (`core/system/provider.go`) and DI-registered in `SystemModule.seedProviders`. Lifecycle is `SystemModule.Start`/`Stop`. | Introduce a **runtime holder** registered in DI during `Setup`, initialized/recovered in `SystemModule.Start`, shut down in `Stop`. Do not add a parallel module. |
| V3 | Hermes options accept `StoreLoader`, `Timeline`, `Nodes`, `Hooks`, `Principal`, `Budget` | None of those exist in Hermes v1.3.0. `StoreLoader`/`Timeline` were removed; nodes register globally through `nodekit.Register`; hooks are per-registration (`RegisterOptions.OnPrepare/OnComplete/OnPause/OnResume/OnCleanup`), not global options. | Rephrase these as Hestia-side concepts. See §2 for the exact v1.3.0 API. |
| V4 | `runtime.Recover` returns an error; `Close` exists | `Recover(ctx)` returns nothing; there is no `Close`; shutdown is `Shutdown(ctx) error`. `Health(ctx)` returns `RuntimeHealth` (no error). | Fix signatures; `Recover` cannot fail the caller directly — log/health-report recover state instead. |
| V5 | `Options.Logger` accepts Hermes' `core.Logger` | v1.3.0 expects `*zap.SugaredLogger`; `pkg/core.Logger` no longer exists. | Delete `zapToHermesLogger` and pass `ps.Logger.Sugar()`. |
| V6 | `ProviderSet.WorkflowRuntime` can simply be removed and the runtime built in `Start` | Service constructors resolve dependencies during `Setup`, before any `Start`. `NewWorkflowsService` resolves `*hermesruntime.WorkflowRuntime` from DI. | Replace the concrete DI binding with a holder; services must tolerate a not-yet-started runtime. |
| V7 | Principal is "registered into `iam.AccessController`" | `iam.AccessController` is a stateless evaluator (`Has/Is/Can/Evaluate`); go-iam has no principal/identity store. Hestia carries actors as `iam.Identity{Permissions, Properties}` in context (`core/abstract/claims.go`, `core/runtime/context`). | Persist the principal in Anansi and project it into an `iam.Identity` on the run's execution context. |
| V8 | RuntimePrincipal can be written into Hermes `RunMeta` | `timeline.RunTimelineMeta.Metadata` exists but Hermes never populates it and exposes no setter; `RunOptions` has no metadata field. | Persist snapshots in a Hestia-owned schema keyed by run ID. Bind at `OnPrepare`, because Hermes mints the run ID internally. Do **not** put the snapshot in workflow state (authors can read it). |
| V9 | SSE replay "falls back to the in-memory timeline" | v1.3.0 has no timeline option; `GetEvents`/`ListRuns` derive from the action log. | The existing `Stream` replay becomes durable automatically once an action log is wired. Replace the task with a reconnect/dedupe regression test. |
| V10 | `RunBudget` rides on `ratestore` and survives restart | `ratestore.InMemoryStore` is an in-memory token bucket that resets on restart and is rate-shaped, not a cumulative counter. | Durable `RunBudget` belongs in Anansi; `ratestore` only gates burst/sustained rates. |
| V11 | Approval/grant is a signed snapshot stored beside the manifest | There is no signing key infrastructure for workflow grants today; `SessionSecret` HKDF is available (`runtime.DerivePurposeKey`). | Either derive a dedicated "workflow-grant" key via HKDF or store a content hash + approver identity in phase 2; do not block on asymmetric signing. |
| V12 | Per-tenant aggregate quota suspends runs via Hermes `Pause` | v1.3.0 exposes `Resume` but no `Pause(runID)`. Pause happens inside a pipeline (`pause` node). | Cannot suspend an arbitrary in-flight run from outside. Use fail-closed node gates and reject new runs; keep "pause in-flight" as an upstream request/open question. |
| V13 | New messages can be added directly to `registrations.go` | `registrations.go`/`policies.go` are generated (`cmd/hestia service generate --all`). | Add annotated handler methods, then regenerate. Never hand-edit generated files. |
| V14 | Audit is applied automatically to node calls | The audit link wraps message dispatch only; custom node calls bypass it. | The workflow audit shim must write `AuditEntry` values to `AuditBuffer` explicitly (and add `ActorTypeWorkflow`). |

### Compile-level validation of the Hermes upgrade

A throwaway worktree with `hermes v1.3.0` produced exactly these failures:

```
core/system/provider.go:167: undefined: hermesscheduler.NewHestiaSchedulerFunc
core/system/provider.go:185: undefined: hermesstore.NewAnansiStoreFactory
core/system/provider.go:196: unknown field StoreLoader in runtime.Options
core/system/provider.go:197: unknown field Timeline in runtime.Options
core/system/provider.go:197: undefined: timeline.NewMemoryTimelineStore
core/system/provider.go:198: cannot use *zapToHermesLogger as *zap.SugaredLogger
core/system/provider.go:225: undefined: hermescore.Logger
```

After a minimal provider-only patch (local scheduler adapter, `NewAnansiStateCollection`
+ `NewPersistentStore`, `actionlog.NewAnansiStore`, `ps.Logger.Sugar()`), the whole
`./core/...` tree built against v1.3.0. That bounds Phase 0: no other package needs
API changes. Baseline `go test ./core/system/workflows/` is green on v1.1.0.

---

## 1. Current state (verified)

- Dispatcher chain: `sanitization → bootstrap → secure → ratelimit → throttle →
  tenant → blob → audit → LocalDispatcher` (`runtime.DefaultChainOrder`;
  `SystemModule.DispatcherChain`). `LocalDispatcher` hosts recovery.
- `WorkflowsService` (`core/system/workflows/service.go`) compiles/runs/streams
  workflows. `Register` hard-codes `Mode{Type: "serialized"}` and performs no
  capability validation.
- Hermes runtime is created in `core/system/provider.go` with a scheduler
  adapter, an Anansi state-store factory, an in-memory timeline, and a logger
  adapter. There is **no** action log, retention, budgets, principal, or
  Hestia-aware node.
- Workflow definitions are Anansi documents (`_workflow_definition_`) with
  `name`, `description`, `nodes`, `edges`, `tenant_id`, timestamps. No
  manifest, status, tier, or approval fields.
- All workflow messages are `root`-only (`core/system/workflows/policies.go`),
  including `system:workflows:runtime:run`.
- `go test ./core/system/workflows/` passes on the current pin.

---

## 2. Hermes v1.3.0 API map (authoritative for this work)

| Need | v1.3.0 surface |
|---|---|
| Construction | `runtime.NewWorkflowRuntime(runtime.Options{...})` |
| Durable log | `actionlog.NewAnansiStore(ctx, persist, "_pipeline_actionlog_")`; optional capabilities `RunEnumerator`, `FoldSnapshotStore`, `UsageReporter`, `LogPruner` |
| State store | `store.NewAnansiStateCollection(ctx, persist, "_pipeline_state_")` then `store.NewPersistentStore(models, nil)` per run |
| Scheduler | `scheduler.Scheduler` interface: `Schedule(id, cron, cb)`, `Cancel(id)`, `Shutdown(ctx)` — implement locally over `*hestia/scheduler.Scheduler` |
| Logger | `Options.Logger *zap.SugaredLogger` |
| Recovery | `Recover(ctx)` (no return). Pauses stage in `pendingPauses` until the owning workflow is registered (`Register` binds them). Re-register definitions **before or after** `Recover`, then verify binding. |
| Shutdown | `Shutdown(ctx) error` |
| Introspection | `Health(ctx) RuntimeHealth`; `StorageUsage(ctx) (actionlog.LogUsage, error)`; `GetRunMeta`, `GetRunOutcome`, `ListRuns`, `GetEvents`, `GetActionLog` |
| Hooks | `RegisterOptions.OnPrepare/OnComplete/OnPause/OnResume/OnCleanup`; `RunOptions` carries the same per-ad-hoc-run hooks |
| Node API | `nodekit.Register(NodeDefinition{Kind, Run, Effect, ConfigSchema, Requirements, ...})`; global registry, so register custom kinds during Setup (or via an explicit `nodes.Register(deps)` call) |
| Drift | `Options.DefinitionPolicy` (`warn` default | `require-same`); `RunTimelineMeta.DefinitionDrift` |
| Retention | `StoreGrace`, `RunHistoryTTL`, `MaxLogAge`, `MaxStateBytes`, `MaxActiveRuns`, `EventTTL`, `DisableFoldSnapshots` |

Notes that change the design:

1. **Run IDs are minted inside Hermes.** A run's store ID becomes the run ID.
   Use `RegisterOptions.OnPrepare` (receives `*RunHandle{RunID, Store, Event}`)
   to move an ephemeral, caller-supplied principal token from the trigger
   payload into the snapshot store and strip it from run state.
2. **Nodes get `context.Background()`.** The pipeline executes with a detached
   context (`runCtx.Run(context.Background())`), so the caller's `iam.Identity`
   is not available. Resolve authority by `nCtx.RunID` from the persisted
   snapshot, never from the node context.
3. **Custom nodes must set `Effect`** (`EffectPure`/`EffectSideEffecting`) or
   `nodekit.Register` panics. `hestia.collection` and `hestia.dispatch` are
   side-effecting.
4. **No external pause.** See V12. Quota consequences must be expressible as
   node failure, run rejection, or run abort.

---

## 3. Locked design decisions

### 3.1 Runtime ownership and lifecycle (replaces draft §7)

Create `core/system/workflows/runtime/` with:

```go
// Runtime is the DI-visible holder. Constructed (New) during SystemModule.Setup,
// started (Start) during SystemModule.Start, stopped in SystemModule.Stop.
type Runtime struct {
    mu      sync.RWMutex
    hermes  *hermesruntime.WorkflowRuntime
    log     actionlog.Store
    service *WorkflowsService
}

func New(cfg Config, deps Deps) (*Runtime, error) // build action log + state collection + hermes
func (r *Runtime) Start(ctx context.Context) error // Recover, health, bind service
func (r *Runtime) Shutdown(ctx context.Context) error
func (r *Runtime) Hermes() *hermesruntime.WorkflowRuntime
func (r *Runtime) Health(ctx context.Context) RuntimeHealthView
```

- `ProviderSet.WorkflowRuntime` is removed.
- `SystemModule.Setup` creates the holder and registers it as
  `*workflowsruntime.Runtime`; `NewWorkflowsService` resolves the holder, not the
  concrete Hermes runtime.
- `SystemModule.Start` loads persisted definitions, registers them with Hermes,
  then calls `Recover(ctx)` (or `Recover` then register and confirm pending
  pauses bind), and publishes health.
- `SystemModule.Stop` calls `Shutdown` before closing persistence.
- No `core/system/workflows/module.go` is introduced.

### 3.2 Enforcement substrate (replaces draft §4.5 / `core/security`)

The interceptor lives in `core/system/workflows/interceptor/` and reuses:

- `iam.AccessController` (`Can/Is/Evaluate`) with a projected
  `iam.Identity{Permissions: effective capabilities, Properties: {user_id:
  workflow:<id>, tenant_id, actor_type: workflow}}`.
- `runtime.RateLimitStore` (`CheckAndConsume`/`Increment`) for burst gates.
- `*runtime.AuditBuffer` for `AuditEntry` emission.

It does **not** re-enter `SecureDispatcher` blindly: that dispatcher resolves a
rule key from a registered message name (`PermissionManager.Resolve`), and node
calls have no such message. Where a capability maps to a real message
(`hestia.dispatch`), the node dispatches that message through the normal chain
so downstream auth/rate-limit/audit still apply. For collection ops, the
interceptor calls the collections/persistence API with an explicit policy check.

Additive registrations in `SystemModule.seedProviders` (do not create a new
package):

```go
abstract.RegisterInstance[iam.AccessController](rt, m.providers.AccessCtrl)
abstract.RegisterInstance[runtime.RateLimitStore](rt, m.providers.RateStore)
abstract.RegisterInstance[*runtime.AuditBuffer](rt, m.providers.AuditBuffer)
```

Also add `ActorTypeWorkflow` to `core/runtime/audit/types.go`.

### 3.3 Identity and manifest

- `WorkflowPrincipal` is persisted (`_workflow_principals_`): definition ID,
  owner ID, name, tenant ID, status (`active|suspended|revoked|pending_approval`),
  tier (`T0..T3`), manifest ref/hash, timestamps.
- At trigger time, resolve a `RuntimePrincipal` (workflow ID, effective caps,
  delegated-by, manifest hash, budget ref, snapshot time) and persist it as
  `_workflow_run_snapshots_` keyed by run ID.
- Effective caps:
  - client credentials (cron/internal): `manifest ∩ workflow_grants`
  - on-behalf-of: `manifest ∩ workflow_grants ∩ invoker_grants`
  - dispatch chains inherit the caller's effective caps as an upper bound.
- Manifest tokens are opaque. The token→resource mapping (`registry/`) is
  platform-owned and hot-reloadable; collection and message names are never
  accepted from workflow authors or input.
- Tier is computed from the manifest by the registration validator; T3 tokens
  are only resolvable for operator-approved manifests.
- Approval (phase 2) records approver identity + manifest hash + timestamp.
  Signed snapshots are optional (derive a key via
  `runtime.DerivePurposeKey(cfg.SessionSecret, "workflow-grant")` if needed).

### 3.4 Data model

New Anansi collections (follow the existing `_workflow_definition_` schema
conventions; regenerate model bindings with the repo's anansi codegen workflow):

| Collection | Purpose |
|---|---|
| `_workflow_principals_` | identity, owner, tenant, status, tier, manifest ref |
| `_workflow_manifests_` | versioned capability manifests |
| `_workflow_grants_` | approval snapshots (approver, hash, version, scope) |
| `_workflow_budgets_` | per-run counters and limits |
| `_workflow_quota_counters_` | per-workflow/per-tenant rate + violation counters |
| `_workflow_run_snapshots_` | frozen `RuntimePrincipal` per run |
| `_pipeline_actionlog_` | Hermes action log (created by `actionlog.NewAnansiStore`) |

Extend `_workflow_definition_` with `manifest_ref`, `manifest_hash`, `tier`,
`status`, `approved_by`, `approved_at`, `owner_id`.

Migration is additive. Definitions without a manifest become **T0 with an empty
manifest** and cannot use `hestia.*` nodes until re-approved.

### 3.5 API surface

Handlers go in `service.go` with `@hestia.register` annotations, then
`go run ./cmd/hestia service generate --all` regenerates `registrations.go` and
`policies.go`.

| Message | Notes |
|---|---|
| `system:workflows:definition:approve` | admin/owner per tier |
| `system:workflows:definition:reject` | records reason |
| `system:workflows:principal:suspend` / `resume` / `revoke` | lifecycle |
| `system:workflows:runtime:health` | wraps `Health` + budget/circuit/pool data |
| `system:workflows:runtime:storage` | wraps `StorageUsage` |
| `system:workflows:run:budget` | budget view |
| `system:workflows:run:actionlog` | `GetActionLog` view |
| `system:workflows:run:snapshot` | principal snapshot view (audit/debug) |

Changed behavior:

- `Compile` rejects unknown `hestia.*` kinds (nodekit lookup) and unresolvable
  manifest references.
- `Register` validates manifest/tier/author grants, creates a
  `WorkflowPrincipal` (`pending_approval`, or active when policy auto-approves
  T0/T1), and sets per-registration hooks.
- `Update` requires re-approval when the manifest changes.
- `Run` resolves the principal, stages it for `OnPrepare`, and applies the burst
  gate.
- `Resume` re-validates the snapshot against current policy and fails closed on
  revoked capabilities.
- `Stream` requires no replay change; verify reconnect dedupe once the action
  log is durable.

### 3.6 Custom nodes

`nodes/` registers two side-effecting kinds during `Setup`:

- `hestia.collection` — `{cap, op, args}`; validates cap/op/limit, applies
  policy, reserves budget, executes through the collections service/persistence,
  returns `truncated`, audits `workflow.collection.<op>`.
- `hestia.dispatch` — `{cap, args}`; cap maps to one concrete message + schema,
  validates args, enforces `max_depth`/`max_fanout` from run lineage, reserves
  budget, dispatches through the normal chain (so downstream auth applies),
  records a correlation handle, audits `workflow.dispatch`.

Reject raw collection names, raw query strings, unbounded limits, arbitrary
projections, and unlisted messages. A query AST can come later.

### 3.7 Quotas and tiers

- `RunBudget` is durable (`_workflow_budgets_`), with a hot in-memory cache
  hydrated on demand: `Calls`, `CollectionDocs`, `Dispatches`, `WallClock`,
  `Suspensions`. Reserve before, commit after; zero returns `BudgetExhausted`.
- Burst/sustained gates use `ratestore` keys
  (`workflow:<id>:burst`, `workflow:<id>:docs:<window>`).
- Consequence matrix (corrected for V12):
  - per-run budget → node fails `RateLimited{scope:"run", retry_after}`
  - per-workflow burst → `Run` rejects new starts
  - sustained abuse → circuit breaker suspends the principal, notifies owner
  - per-tenant aggregate → fail-closed node gate + reject new starts;
    in-flight pause requires an upstream Hermes API (open question)
  - global → `Options.MaxActiveRuns` + scheduler backpressure
- Tiers: T0 orchestration/shared/small, T1 scoped data/shared/medium, T2
  privileged/reserved/large, T3 system/isolated/dedicated. Tier pools are a
  scheduler-adapter concern; reserve capacity per tier and expose saturation.

### 3.8 Lifecycle hooks and audit

- `OnPause`/`OnResume` are passed in `RegisterOptions`: audit
  `workflow.run.paused`/`resumed`, notify the owner when human input is needed,
  update suspension counters.
- Workflow audit entries use `ActorTypeWorkflow`, `ActorID = workflow:<id>`,
  `OnBehalfOfID = invoker`, metadata `{run_id, capability, outcome}`.

---

## 4. Phased implementation

### Phase 0 — Hermes v1.3.0 baseline

- [*] Pin `github.com/asaidimu/hermes v1.3.0`; `go mod tidy`.
  - **Context:** v1.3.0 was already in the local module cache. `go mod tidy`
    succeeds offline (`GOPROXY=off`). Go toolchain in use is 1.27.4.
- [*] Replace `hermesscheduler.NewHestiaSchedulerFunc` with a local adapter over
  `*hestia/scheduler.Scheduler` (`Schedule`/`Cancel`/`Shutdown`).
  - **Files:** `core/system/provider.go` (`hestiaScheduler`). `Shutdown` is a
    no-op because `SystemModule.Stop` owns the scheduler lifetime.
- [*] Replace `hermesstore.NewAnansiStoreFactory` with
  `NewAnansiStateCollection(ctx, ps.Persist, "_pipeline_state_")` +
  `NewPersistentStore(models, nil)`.
- [*] Drop `StoreLoader` and `Timeline`; wire
  `actionlog.NewAnansiStore(ctx, ps.Persist, "_pipeline_actionlog_")`. Retention
  options stay zero (documented defaults) until Phase 1 exposes config.
- [*] Drop `zapToHermesLogger`/`hermescore.Logger`; pass `ps.Logger.Sugar()`.
- [*] Remove the now-unused `github.com/asaidimu/hermes/pkg/timeline` import.
- [*] Update `workflows_test.go` to use `zap.NewNop().Sugar()`.
- [*] Run `go build ./...`, then the full suite with `-race`.
  - **Gate met:** `go test -race ./core/...` green; `./core/system` and
    `./core/system/workflows` pass.

### Phase 1 — Durable runtime and ownership

- [ ] Add `core/system/workflows/runtime` holder (`New`/`Start`/`Shutdown`/
  `Hermes`/`Health`) and move Hermes construction out of
  `ProviderSet.InitModels`.
  - **Context:** service constructors run in `Setup`, before `Start`; the holder
    must be usable (not necessarily started) at construction time.
- [ ] Remove `ProviderSet.WorkflowRuntime`; register `*workflowsruntime.Runtime`
  in `seedProviders`; update `NewWorkflowsService` and `SystemModule.Stop`.
- [ ] Wire action-log retention (`StoreGrace`, `RunHistoryTTL`, `MaxLogAge`,
  `MaxStateBytes`, `MaxActiveRuns`, `EventTTL`, `DefinitionPolicy`) from
  config with documented defaults.
- [ ] Implement startup rebuild: re-register persisted definitions, then
  `Recover(ctx)`; verify that `pendingPauses` bind (Hermes stages pauses it
  cannot arm and `Register` later consumes them).
- [ ] Expose `runtime:health` and `runtime:storage` handlers (annotated methods,
  regenerate).
  - **Files:** `service.go`, `inputs.go`, `outputs.go`, `registrations.go`
    (generated).
  - **Gate:** a paused run survives a process restart and resumes; `ListRuns`
    and `GetEvents` replay after restart; SSE reconnect does not duplicate
    terminal events.
- [ ] Add a restart integration test that boots the runtime twice within one
  process (note: generated model collections are process-wide singletons; use
  the same pattern as existing boot tests or exercise the holder directly).

### Phase 2 — Identity, manifest, approval

- [ ] Add schemas + generated models for principals, manifests, grants; extend
  the definition schema.
- [ ] Implement `registry/` token→resource mapping with tier scoping and
  hot reload from the policies store.
- [ ] Implement `identity/`: status lifecycle, tier derivation, IAM projection
  helper (`iam.Identity` from a persisted principal).
- [ ] Implement registration validation: token existence, tenant scope, schema
  validity, derived tier, author grants; reject T3 for non-operators.
- [ ] Implement approval handlers + grant snapshots (approver, hash, version).
- [ ] Change `Register`/`Update` to create/require approval and version
  manifests.
- [ ] Seed/migrate existing definitions as T0 with an empty manifest.
  - **Gate:** unapproved workflows cannot register; manifest edits require
    re-approval; T3 manifest is rejected for non-operators; existing
    definitions keep working as T0.

### Phase 3 — Runtime principal and interceptor

- [ ] Implement `session/`: resolve `RuntimePrincipal` at trigger time
  (intersection rules) and persist it in `_workflow_run_snapshots_`.
- [ ] Bind the snapshot at `OnPrepare`: move the ephemeral trigger token from
  the event payload into the snapshot row and strip it from run state.
  - **Context:** Hermes mints the run ID internally; `OnPrepare` is the first
    point where it is known and execution can still be aborted.
- [ ] Implement `interceptor/`: authorize (`iam.AccessController.Can`), quota,
  execute, audit. Resolve the principal by `nCtx.RunID` — node contexts are
  `context.Background()`.
- [ ] Register `iam.AccessController`, `RateLimitStore`, and `*AuditBuffer` in
  DI; add `ActorTypeWorkflow`.
- [ ] Re-validate snapshots on `Resume`; fail closed if any capability was
  revoked or the manifest hash changed.
  - **Gate:** node call without a capability fails; snapshot survives restart;
  revoked capability fails resume; audit rows carry actor/run/capability/outcome.

### Phase 4 — Custom nodes

- [ ] Implement `nodes/hestia.collection` and `nodes/hestia.dispatch`;
  register them during Setup with `EffectSideEffecting`.
- [ ] Enforce op allowlists, `max_docs_per_call`, schema validation, dispatch
  depth/fanout, and budget reservations.
- [ ] Audit every call; return structured `truncated` results.
- [ ] Make `Compile` reject unknown `hestia.*` kinds before registration.
  - **Gate:** raw collection/message names are rejected; depth/fanout enforced;
  invalid dispatch args fail before execution; audit events emitted.

### Phase 5 — Quotas, tiers, circuit breaker

- [ ] Implement durable `RunBudget` storage + hot cache; reserve/commit/exhaust
  semantics.
- [ ] Implement per-workflow burst and sustained gates over `ratestore`.
- [ ] Implement consequence matrix (V12-corrected) and circuit breaker that
  suspends the principal and notifies the owner.
- [ ] Add tier pools to the scheduler adapter with reserved concurrency and
  saturation exposure.
  - **Gate:** per-run budget fails the node; burst rejects run starts;
  suspension survives restart; tier isolation is demonstrable under load.

### Phase 6 — Observability and hardening

- [ ] Extend health with budget usage, suspensions, circuit state, pool
  saturation, and action-log retention posture.
- [ ] Emit workflow lifecycle audit + operator notifications.
- [ ] Add action-log prune/`StorageUsage` monitoring and document retention
  defaults.
- [ ] Client SDK: add approve/reject/suspend/resume/revoke/health/storage/
  budget/snapshot messages and types (`client/packages/core/system/workflows/`).
- [ ] Docs: workflow security model, manifest authoring, approvals, quotas.
  - **Gate:** docs/sdk cover every new message; operator runbook has a
  quarantine/recover procedure.

---

## 5. Test plan

Unit:

- [ ] Manifest validation: token existence, schema validity, tier derivation,
  author grants.
- [ ] Capability intersection: client credentials vs on-behalf-of vs dispatch
  chain.
- [ ] Budget reserve/commit/exhaust/restart persistence.
- [ ] Tier pool selection and saturation.
- [ ] Principal projection into `iam.Identity` and policy evaluation.

Integration:

- [ ] Restart recovery: paused run listed and resumable after restart.
- [ ] Retention: terminal eviction, history expiry, log pruning.
- [ ] Budget exhaustion: node fails with `BudgetExhausted` or maps to
  `RateLimited{scope:"run"}` consistently.
- [ ] Burst gate: `Run` rejected when the workflow burst limit is exceeded.
- [ ] Circuit breaker: repeated violations suspend the workflow and emit an
  operator notification.
- [ ] Drift: `warn` surfaces drift; `require-same` rejects drifted resume.
- [ ] Capability revocation: resume fails closed.
- [ ] SSE replay: reconnect replays from the action log without duplicate
  terminal events.
- [ ] Node schema: invalid dispatch args rejected before execution.
- [ ] Depth/fanout: recursive dispatch blocked beyond manifest limits.

Security:

- [ ] Raw collection name injection → rejected.
- [ ] Unlisted dispatch message → rejected.
- [ ] Privilege escalation via a dispatch chain → intersection enforced.
- [ ] Registering a T3 manifest as a non-operator → rejected.
- [ ] Audit trail contains `actor_type`, `actor_id`, `run_id`, capability, and
  outcome for every node call.

---

## 6. Rollout, rollback, migration

Rollout:

1. Phases 0–1 with no user-facing behavior change.
2. Identity/approval in **shadow mode**: validate and log would-be rejections
   without blocking.
3. Enforce for new workflows; existing definitions remain T0 until re-approved.
4. Enable `hestia.*` nodes for T1+ after approval.
5. Enable quotas/tier pools with conservative defaults.

Rollback:

- Hermes pin can revert to v1.1.0; the provider patch is small and reversible.
- Action-log wiring is additive (reverting loses restart durability, not data).
- Manifest enforcement can be disabled by a config flag; schemas remain.

Migration:

- Additive schemas only.
- Backfill a T0 principal + empty manifest for existing definitions; mark
  custom-node use `pending_approval`.
- New runs carry snapshots; old runs are not backfilled.

---

## 7. Risks and open questions

| Risk | Mitigation |
|---|---|
| Hermes v1.3.0 behavior drift in `Register`/`Recover` | Phase 0 compile+race audit; pin exact version; hold recovery test. |
| `Recover` cannot fail the caller | Surface recover state in health; log; fail closed when a paused run cannot be armed. |
| Action-log growth in embedded deployments | Retention from day one; monitor `StorageUsage`; expose prune status. |
| Row-level policy cost on large collections | Cap `max_docs_per_call`; indexed filters; budget docs per call. |
| Tier pools starve the shared pool | Reserve capacity; expose saturation; conservative defaults. |
| Approval UX bottleneck | Auto-approve T0/T1 by policy; human approval only for T2/T3 and manifest expansion. |
| Snapshot binding race (run ID minted late) | Bind in `OnPrepare` and abort the run if binding fails; test. |
| Existing embedders resolve `ProviderSet.WorkflowRuntime` | Replace with the holder in the same change; grep consumers before removal. |

Open questions:

1. `hestia.dispatch`: synchronous completion only, or async with a correlation
   handle consumed by a later node?
2. May T0 workflows dispatch to T1/T2 workflows, or only to other T0s and
   notifications?
3. Capability registry location: its own sub-package over the policies store
   (recommended) or folded into `core/system/policies`?
4. Per-tenant aggregate suspension: accept fail-closed node failure until
   Hermes exposes an external `Pause(runID)`? (v1.3.0 has no such API.)
5. Grant signing: HKDF-derived key + HMAC, or approver identity + manifest hash
   only for phase 2?

---

## 8. File map (expected touch points)

| Path | Change |
|---|---|
| `go.mod` / `go.sum` | Hermes v1.3.0 pin |
| `core/system/provider.go` | Remove runtime construction; state collection/action log setup |
| `core/system/module.go` | Holder DI registration; create/start/stop lifecycle; seed new primitives |
| `core/system/services.go` (generated) | Regenerated registrations |
| `core/system/workflows/service.go` | Manifest-aware Register/Compile/Run/Resume, new handlers |
| `core/system/workflows/inputs.go` / `outputs.go` | New message contracts and views |
| `core/system/workflows/registrations.go` / `policies.go` | Generated — regenerate, never hand-edit |
| `core/system/workflows/model/` | Definition schema extension + new schemas/models |
| `core/system/workflows/runtime/` | Holder, options, recovery, health |
| `core/system/workflows/identity/` | Principal, status, tier, IAM projection |
| `core/system/workflows/registry/` | Token→resource mapping, versioning, tier scope |
| `core/system/workflows/approval/` | Approval/rejection, grant snapshots |
| `core/system/workflows/session/` | RuntimePrincipal resolution/snapshot/delegation |
| `core/system/workflows/interceptor/` | Authorize → quota → execute → audit |
| `core/system/workflows/quota/` | Durable RunBudget + ratestore gates |
| `core/system/workflows/sandbox/` | Tier pools, timeouts, dispatch depth |
| `core/system/workflows/audit/` | Workflow actor shim over `AuditBuffer` |
| `core/system/workflows/nodes/` | `hestia.collection`, `hestia.dispatch` |
| `core/runtime/audit/types.go` | `ActorTypeWorkflow` |
| `core/runtime/ratestore/` | Per-workflow/per-run key helpers (reuse store) |
| `client/packages/core/system/workflows/` | SDK messages/types |

---

## 9. Definition of done

A workflow is an application: it has an owner, tenant, lifecycle, declared
capabilities, a frozen per-run authority snapshot, durable execution, quotas,
audit, and isolation — and every `hestia.*` node call is authorized, quota-checked,
executed, and audited through Hestia's existing security substrate.
