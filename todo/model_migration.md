# Hand-rolled model → generated anansi model migration

Replace hand-rolled persistence code in `core/system/` with generated
`ModelCollection` types. Reference pattern: `core/system/<name>/users` service.
Each feature keeps a thin `<feature>_utils.go` / orchestration layer over its
generated collection singleton (`Init...Model(persist, logger)`).

- [*] TenantModel → `*tenantsmodel.SystemTenants`
  - **Context:** hand-rolled `core/system/tenants/model.go` duplicated schema binding logic.
  - **Details:** utils in `core/system/tenants/model/system_tenant_utils.go`
    (`CreateTenant`, `GetByID`, `GetByDomain`); provider/seeds/auth_test rewired.
- [*] AuditModel → `*auditmodel.SystemAuditLogs`
  - **Details:** `Insert(ctx, AuditEntry)` maps domain enums → generated enums;
    local `auditCollectionName` const in `audit/handler.go`.
- [*] SettingsModel → `*settingsmodel.SystemSettingss`
  - **Details:** `Get/Set/Unset/All` with value-envelope mapping
    (`toRecord`/`fromRecord`); removed duplicate legacy `model/model.go`.
- [*] ScheduleModel → `*schedulesmodel.SystemScheduledMessagess`
  - **Details:** utils return `*document.Document` for `data.Documenter` compat;
    duplicate legacy `model/model.go` removed; LiveSchedule wiring unchanged.
- [*] PolicyModel → generated models as building blocks (kept as orchestration layer)
  - **Context:** NOT a plain swap — `_operation_policy_` writes must stay coherent
    with two live caches: `LivePermissionManager` (compiled `*Policy` keyed by
    composite key) and the access controller (CEL-compiled `iam.FunctionRule`).
    The old design swapped raw collections for LiveRepositories post-construction
    via `SetPolicyColl`/`SetRuleColl`. Generated singletons bound to raw
    persistence would have silently broken cache coherence.
  - **Resolution:** construct the generated ModelCollections OVER the
    LiveRepositories — `collection.NewModelCollection[*SystemOperationPolicy](liveRepo, ...)`
    works because `liveRepository` embeds `base.Collection`. All PolicyModel
    writes flow ModelCollection → LiveRepository → DB + caches atomically.
  - **Files:** `policies/model.go` rewritten (typed API, exact original query
    semantics preserved incl. composite-key fallback and protected/in-use rule
    checks); `policies/docprocessor.go` keeps `docToPolicy` (raw-document
    compilation for the live cache — legitimately different concern);
    `system/module.go`: `initPermissions`+`initAccessController` merged into
    `initPolicyInfra`, runs BEFORE `seedData` (SeedAll dereferences ps.Policies);
    `SetPolicyColl`/`SetRuleColl` deleted.
  - **Tests:** reset+init singleton pattern per test
    (`DangerouslyReset...Model()` then `Init...Model(p)`) — same as schedules.
