package system

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/sanitize"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/blobs/staging"
	"github.com/asaidimu/updater"

	"github.com/asaidimu/hestia/core/abstract"
	auditmodel "github.com/asaidimu/hestia/core/system/audit/model"
	"github.com/asaidimu/hestia/core/system/auth"
	authsvc "github.com/asaidimu/hestia/core/system/auth"
	"github.com/asaidimu/hestia/core/system/blobs"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	"github.com/asaidimu/hestia/core/system/collections"
	"github.com/asaidimu/hestia/core/system/logs"
	operationsvc "github.com/asaidimu/hestia/core/system/operations"
	"github.com/asaidimu/hestia/core/system/policies"
	policiesmodel "github.com/asaidimu/hestia/core/system/policies/model"
	"github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/updates"
	"github.com/asaidimu/hestia/core/system/users"
	"github.com/asaidimu/hestia/core/system/users/model"

	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	module "github.com/asaidimu/hestia/core/runtime/module"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
	"github.com/asaidimu/hestia/core/runtime/scheduler"

	hermesruntime "github.com/asaidimu/hermes/pkg/runtime"
)

type SystemModule struct {
	module.BaseModule

	opts      dispatch.SystemOptions
	cfg       *runtime.Config
	disp      *runtime.LocalDispatcher
	providers *ProviderSet

	// chainedDisp holds the full dispatcher chain once DispatcherChain
	// builds it (post-boot). Schedule fires resolve it lazily so they
	// traverse authorization, rate limiting and audit (S-1).
	chainedDisp atomic.Value

	bootstrapped bool
	ephemeralKey string
	adminUserID  string
	adminEmail   string
	messages     []abstract.MessageRegistration
}

func New(cfg *runtime.Config, disp *runtime.LocalDispatcher, opts dispatch.SystemOptions) *SystemModule {
	return &SystemModule{
		opts: opts,
		cfg:  cfg,
		disp: disp,
	}
}

func (m *SystemModule) Name() string { return "system" }

func (m *SystemModule) Setup(ctx context.Context, rt abstract.Container) error {
	persist := abstract.MustResolve[base.Persistence](rt)
	ring := abstract.MustResolve[*logs.RingBuffer](rt)
	m.providers = NewProviderSet(persist, m.cfg, m.opts.Logger, ring)

	// Register feature-scoped sanitization rules. Each feature declares its
	// own rules via SanitizationRules(); the dispatcher sets the scope in
	// context based on the message name's feature segment.
	reg := sanitize.Registry()
	for scope, config := range allSanitizationRules {
		if err := reg.Register(scope, config); err != nil {
			m.opts.Logger.Warn("Failed to register sanitization rules", zap.String("scope", scope), zap.Error(err))
		}
	}

	if err := m.providers.InitModels(ctx); err != nil {
		return fmt.Errorf("init models: %w", err)
	}
	if err := m.initPolicyInfra(ctx); err != nil {
		return fmt.Errorf("init policy infra: %w", err)
	}

	m.providers.LiveSchedule = schedules.NewLiveSchedule(m.providers.Schedules, m.providers.Scheduler, m.disp, m.opts.Logger).
		WithAuthorizer(schedules.NewScheduleAuthorizer(
			m.providers.PermMgr,
			m.providers.AccessCtrl,
			func() collection.LiveCollection[*users.UserClaims] { return m.providers.LiveUsers },
		)).
		WithDispatcherProvider(m.ChainedDispatcher())
	if err := m.providers.LiveSchedule.Init(ctx); err != nil {
		return fmt.Errorf("init live schedule: %w", err)
	}

	// Derive independent per-purpose signing keys from the master secret via
	// HKDF (see runtime.DerivePurposeKey). This fails boot when no session
	// secret was provisioned — there is deliberately no default secret.
	sessionKey, err := runtime.DerivePurposeKey(m.cfg.SessionSecret, "session")
	if err != nil {
		return fmt.Errorf("derive session keys: %w", err)
	}
	resetKey, err := runtime.DerivePurposeKey(m.cfg.SessionSecret, "password-reset")
	if err != nil {
		return fmt.Errorf("derive session keys: %w", err)
	}
	// The blocklist backs session revocation (S-4) and reset-token
	// consumption (S-13); see auth.TokenBlocklist.
	blocklist, err := auth.NewTokenBlocklist(persist, m.opts.Logger)
	if err != nil {
		return fmt.Errorf("init token blocklist: %w", err)
	}

	sessionSvc := auth.NewSessionService(sessionKey)
	m.providers.CredProv = auth.NewCredentialsProviderWithVersion(sessionSvc, resetKey, func(ctx context.Context, userID string) (int, error) {
		user, err := m.providers.Users.GetByID(ctx, userID)
		if err != nil {
			return 0, nil
		}
		return user.GetTokenVersion(), nil
	}, blocklist)

	svc, err := blobutil.NewService(m.cfg.BlobsDir, m.opts.Logger)
	if err != nil {
		return fmt.Errorf("init blob service: %w", err)
	}
	m.providers.BlobSvc = svc

	if err := m.seedData(ctx); err != nil {
		return err
	}
	if err := m.initUserClaimsCache(ctx); err != nil {
		return fmt.Errorf("init user claims cache: %w", err)
	}

	m.providers.PolicyBridge = policies.NewPolicyStoreAdapter(m.providers.Policies, m.providers.PermMgr, m.providers.LiveRules)

	apiKeyAuth := auth.NewAPIKeyAuthenticator(m.providers.APIKeys, m.providers.LiveUsers, m.ephemeralKey, m.adminUserID, m.adminEmail, m.opts.Logger, m.Bootstrapped)

	if err := m.registerExistingDocumentHandlers(ctx); err != nil {
		return fmt.Errorf("register document handlers: %w", err)
	}
	if err := m.registerExistingBlobHandlers(ctx); err != nil {
		return fmt.Errorf("register blob handlers: %w", err)
	}

	m.providers.Bootstrapped = m.bootstrapped

	// Seed the system providers into the shared runtime, then register every
	// service provider. The two-phase boot seals the container (Rebuild) after
	// all modules have set up; registrations are resolved in Capabilities.
	if err := m.seedProviders(rt, apiKeyAuth); err != nil {
		return err
	}
	return RegisterServices(rt)
}

// seedProviders registers the system-level providers into the shared runtime
// container that the scaffolded services resolve in their constructors. The
// base providers (persistence, logger, dispatcher) are pre-seeded by boot.
func (m *SystemModule) seedProviders(rt abstract.Container, apiKeyAuth *auth.APIKeyAuthenticator) error {
	if err := abstract.RegisterInstance[*schedules.LiveSchedule](rt, m.providers.LiveSchedule); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*policies.PolicyModel](rt, m.providers.Policies); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[runtime.ReloadablePermissionManager](rt, m.providers.PermMgr); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[iam.RuleSet[iam.FunctionRule]](rt, m.providers.LiveRules); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[abstract.BindingPolicyStore](rt, m.providers.PolicyBridge); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*blobutil.Service](rt, m.providers.BlobSvc); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*staging.Manager](rt, m.providers.BlobSvc.Staging()); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[operationsvc.BootstrappedFunc](rt, func() bool { return m.providers.Bootstrapped }); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[operationsvc.OnBootstrapFunc](rt, m.bootstrapCallback()); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[operationsvc.OnResetFunc](rt, m.resetCallback()); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*auditmodel.SystemAuditLogs](rt, m.providers.Audit); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*[]abstract.MessageRegistration](rt, &m.messages); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*scheduler.Scheduler](rt, m.providers.Scheduler); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[abstract.CredentialsProvider](rt, m.providers.CredProv); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[*auth.APIKeyAuthenticator](rt, apiKeyAuth); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[authsvc.AdminUserID](rt, authsvc.AdminUserID(m.adminUserID)); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[authsvc.SessionTTL](rt, authsvc.SessionTTL(m.cfg.SessionTTL)); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[abstract.Notifier](rt, m.providers.Notifier); err != nil {
		return err
	}
	if err := abstract.RegisterInstance[authsvc.AppURL](rt, authsvc.AppURL(m.providers.AppURL)); err != nil {
		return err
	}
	// Register updater-specific types so updates.NewService can resolve them via DI.
	if m.providers.Updater != nil {
		if err := abstract.RegisterInstance[*updater.Updater](rt, m.providers.Updater); err != nil {
			return err
		}
		if err := abstract.RegisterInstance[updates.AutoApply](rt, updates.AutoApply(m.cfg.SelfUpdate.AutoApply)); err != nil {
			return err
		}
		if err := abstract.RegisterInstance[updates.SystemdMode](rt, updates.SystemdMode(m.cfg.SelfUpdate.SystemdMode)); err != nil {
			return err
		}
		if m.opts.OnRestartRequired != nil {
			if err := abstract.RegisterInstance[updates.RestartHook](rt, updates.RestartHook(m.opts.OnRestartRequired)); err != nil {
				return err
			}
		}
		if err := abstract.RegisterInstance[updates.ExePath](rt, updates.ExePath(m.providers.UpdExe)); err != nil {
			return err
		}
		if err := abstract.RegisterInstance[updates.UpdateDataDir](rt, updates.UpdateDataDir(m.providers.UpdData)); err != nil {
			return err
		}
		if err := abstract.RegisterInstance[updates.HasMailer](rt, updates.HasMailer(m.cfg.Mailer.SMTPHost != "")); err != nil {
			return err
		}
		if err := abstract.RegisterInstance[updates.AppURL](rt, updates.AppURL(m.providers.AppURL)); err != nil {
			return err
		}
		if err := abstract.RegisterInstance[updates.AppVersion](rt, updates.AppVersion(m.cfg.Version)); err != nil {
			return err
		}
	}
	// Register hermes workflow runtime
	if err := abstract.RegisterInstance[*hermesruntime.WorkflowRuntime](rt, m.providers.WorkflowRuntime); err != nil {
		return err
	}
	return nil
}

func (m *SystemModule) bootstrapCallback() func() {
	return func() {
		m.bootstrapped = true
		if m.opts.OnBootstrapped != nil {
			m.opts.OnBootstrapped()
		}
	}
}

func (m *SystemModule) resetCallback() func() {
	return func() {
		if m.opts.OnReset != nil {
			m.opts.OnReset()
		}
	}
}

// Capabilities resolves the service registrations from the sealed runtime
// container. It is called by boot after the two-phase seal so the services
// seeded during Setup can be resolved.
func (m *SystemModule) Capabilities(rt abstract.Container) ([]abstract.Capability, error) {
	svcRegs, err := CollectServiceRegistrations(rt)
	if err != nil {
		return nil, err
	}
	m.messages = m.providers.CollectRegistrations(svcRegs)
	knownBindings := collectAllPolicyBindings()
	m.providers.Policies.SetKnownBindings(knownBindings)
	return []abstract.Capability{
		{
			Name:     "system",
			Messages: m.messages,
		},
	}, nil
}

func (m *SystemModule) seedData(ctx context.Context) error {
	adminEmail := m.opts.AdminEmail
	if adminEmail == "" {
		adminEmail = m.cfg.AdminEmail
	}
	adminPassword := m.opts.AdminPassword
	if adminPassword == "" {
		adminPassword = m.cfg.AdminPassword
	}

	result, err := SeedAll(ctx, m.providers, SeedOptions{
		AdminEmail:        adminEmail,
		AdminPassword:     adminPassword,
		ForceBootstrapped: m.opts.ForceBootstrapped,
	}, m.allDefaultPolicies())
	if err != nil {
		return err
	}

	m.adminUserID = result.AdminUserID
	m.adminEmail = result.AdminEmail
	m.bootstrapped = result.Bootstrapped

	// The ephemeral bootstrap key exists solely to bootstrap an un-bootstrapped
	// system. Generate it only in that state and clear it otherwise, so a key
	// captured from boot logs (container/CI logs, scrollback) cannot retain
	// permanent admin access after bootstrap (todo/first_run_api_key.md).
	if m.bootstrapped {
		m.ephemeralKey = ""
	} else if m.ephemeralKey == "" {
		key := make([]byte, 16)
		if _, err := rand.Read(key); err == nil {
			m.ephemeralKey = hex.EncodeToString(key)
		}
	}

	return nil
}

// initPolicyInfra wires the policy/rule persistence stack: it opens both raw
// collections, layers the LiveRepositories (compiled *Policy / iam.FunctionRule
// caches) over them, then constructs the generated ModelCollections OVER those
// live repositories. Every write through PolicyModel therefore flows through
// the same instances the permission manager and access controller read from —
// DB and live caches stay coherent without manual invalidation.
func (m *SystemModule) initPolicyInfra(ctx context.Context) error {
	ps := m.providers

	opColl, err := ps.Persist.Collection(ctx, "_operation_policy_")
	if err != nil {
		return fmt.Errorf("open _operation_policy_ collection: %w", err)
	}
	ruleColl, err := ps.Persist.Collection(ctx, "_iam_rule_")
	if err != nil {
		return fmt.Errorf("open _iam_rule_ collection: %w", err)
	}

	// Live policy repository: compiled *Policy cache keyed by composite key.
	// Degrades to a raw-backed model plus static-default policies on failure.
	opBacking := opColl
	livePolicies, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[*policies.Policy]{
		Collection: opColl,
		Processor:  &policies.PolicyDocProcessor{},
		QueryKey:   "key",
		AutoLoad:   false,
	})
	if err != nil {
		ps.Logger.Warn("Failed to create live policy repository, using static defaults", zap.Error(err))
		ps.PermMgr = policies.NewLivePermissionManager(nil, m.allDefaultPolicies())
	} else {
		ps.LivePolicies = livePolicies
		opBacking = livePolicies
		ps.PermMgr = policies.NewLivePermissionManager(livePolicies, m.allDefaultPolicies())
	}

	// Live rule repository: compiled CEL functions keyed by rule name.
	liveRules, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[iam.FunctionRule]{
		Collection: ruleColl,
		Processor:  &policies.RuleDocProcessor{},
		QueryKey:   "name",
		AutoLoad:   false,
	})
	if err != nil {
		return fmt.Errorf("create live rule repository: %w", err)
	}
	ps.LiveRules = liveRules
	for name, fn := range policies.GoDefaultRules() {
		liveRules.Set(name, fn)
	}

	opModelColl, err := collection.NewModelCollection[*policiesmodel.SystemOperationPolicy](opBacking, ps.Logger)
	if err != nil {
		return fmt.Errorf("build operation policy model collection: %w", err)
	}
	ruleModelColl, err := collection.NewModelCollection[*policiesmodel.SystemIamRule](liveRules, ps.Logger)
	if err != nil {
		return fmt.Errorf("build iam rule model collection: %w", err)
	}
	ps.OpModel = &policiesmodel.SystemOperationPolicys{ModelCollection: opModelColl}
	ps.RuleModel = &policiesmodel.SystemIamRules{ModelCollection: ruleModelColl}
	ps.Policies = policies.NewPolicyModel(ps.OpModel, ps.RuleModel, nil)

	ps.AccessCtrl = iam.CreateAccessController(iam.AccessControllerOptions{
		Rules:    liveRules,
		CacheTTL: 0,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return nil
}

func (m *SystemModule) initUserClaimsCache(ctx context.Context) error {
	userColl, err := m.providers.Persist.Collection(ctx, "_user_")
	if err != nil {
		return fmt.Errorf("open _user_ collection: %w", err)
	}

	claims, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[*users.UserClaims]{
		Collection: userColl,
		Processor:  &users.UserClaimsDocProcessor{},
		QueryKey:   "_id_",
		AutoLoad:   false,
		QueryFunc: func(key string) query.Query {
			// disabled == -1 means active (see users GetActiveByID). The
			// previous Neq(0) predicate matched active (-1) AND disabled (1),
			// so disabled users stayed in the claims cache and their API keys
			// kept authenticating after the account was disabled.
			if key == "" {
				return query.NewQueryBuilder().
					Where("disabled").Eq(-1).
					Build()
			}
			return query.NewQueryBuilder().
				Where(data.DocumentIDField).Eq(key).
				Where("disabled").Eq(-1).
				Build()
		},
	})
	if err != nil {
		return fmt.Errorf("create live user claims repository: %w", err)
	}
	m.providers.LiveUsers = claims

	return nil
}

// failClosedChain rebuilds the canonical chain when embedder mutations
// violate the composition contract (audit A-2). Every violation is logged
// loudly: the embedder asked for an insecure chain and did not get one.
func (m *SystemModule) failClosedChain(canonical []runtime.LinkEntry, violations []string) *runtime.DispatcherChain {
	if m.opts.Logger != nil {
		for _, v := range violations {
			m.opts.Logger.Error("dispatcher chain: contract violation — falling back to canonical chain order", zap.String("violation", v))
		}
	}
	return runtime.NewDispatcherChain(canonical...)
}

func (m *SystemModule) registerExistingBlobHandlers(ctx context.Context) error {
	namespaces, err := m.providers.BlobSvc.ListNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, ns := range namespaces {
		for _, op := range blobs.BlobOps() {
			opName := "system:blobs:" + ns.ID + ":" + op.Suffix
			if err := m.providers.PolicyBridge.EnsureBinding(ctx, opName, op.RuleKey); err != nil {
				return fmt.Errorf("seed operation %s: %w", opName, err)
			}
		}
		if err := blobs.RegisterBlobHandlers(m.disp, m.providers.BlobSvc, m.providers.BlobSvc.Staging(), ns.ID); err != nil {
			return fmt.Errorf("register blob handlers for %q: %w", ns.ID, err)
		}
	}
	if err := m.providers.PolicyBridge.ReloadPolicies(ctx); err != nil {
		return fmt.Errorf("reload policies: %w", err)
	}
	return nil
}

func (m *SystemModule) registerExistingDocumentHandlers(ctx context.Context) error {
	names, err := m.providers.Persist.ListCollections(ctx)
	if err != nil {
		return err
	}
	for _, name := range names {
		if strings.HasPrefix(name, "_") {
			continue
		}
		if err := collections.RegisterDocumentHandlers(m.disp, m.providers.Persist, name); err != nil {
			return fmt.Errorf("register doc handlers for %q: %w", name, err)
		}
	}
	return nil
}

func (m *SystemModule) DispatcherChain(next abstract.Dispatcher) abstract.Dispatcher {
	if m.providers.RateStore == nil {
		m.providers.RateStore = ratestore.New()
	}
	sharedRateStore := m.providers.RateStore
	rateLimitLookup := func(op string) *runtime.RateLimitPolicy {
		if m.providers.LivePolicies == nil {
			return nil
		}
		p, ok := m.providers.LivePolicies.Get(":" + op)
		if !ok || p == nil {
			return nil
		}
		return p.RateLimit
	}

	throttleLookup := func(op string) *runtime.ThrottlePolicy {
		if m.providers.LivePolicies == nil {
			return nil
		}
		p, ok := m.providers.LivePolicies.Get(":" + op)
		if !ok || p == nil {
			return nil
		}
		return p.Throttle
	}

	// Chain order: recovery (via LocalDispatcher) → sanitization → bootstrap → secure → ... → audit
	// Recovery is outermost (built into LocalDispatcher). Sanitization sets the
	// scope context and sanitizes outgoing documents for all downstream links.
	// The composition order is validated data (runtime.DefaultChainOrder,
	// audit A-2): embedder mutations via DispatcherChainFunc go through a
	// GuardedChainEditor and the final chain is validated — an invalid
	// chain is discarded and the canonical order is built instead
	// (fail closed), with the violation logged.
	canonicalEntries := []runtime.LinkEntry{
		{Name: "sanitization", Link: runtime.NewSanitizationDispatcher(nil)},
		{Name: "bootstrap", Link: runtime.NewBootstrapDispatcher(nil, m.disp, func() bool { return m.bootstrapped })},
		{Name: "secure", Link: runtime.NewSecureDispatcher(nil, m.providers.PermMgr, m.providers.AccessCtrl)},
		{Name: "ratelimit", Link: runtime.NewRateLimitDispatcher(rateLimitLookup, sharedRateStore, m.opts.Logger)},
		{Name: "throttle", Link: runtime.NewThrottleDispatcher(throttleLookup, m.disp, m.opts.Logger, sharedRateStore)},
		{Name: "tenant", Link: runtime.NewTenantDispatcher(nil, func(ctx context.Context) string {
			if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
				return claims.TenantID
			}
			return ""
		})},
		{Name: "blob", Link: blobutil.NewDispatcherLink(m.providers.BlobSvc)},
		{Name: "audit", Link: runtime.NewAuditDispatcherWithLogger(nil, m.providers.Audit, m.opts.Logger, m.providers.AuditBuffer)},
	}
	chain := runtime.NewDispatcherChain(canonicalEntries...)
	if m.opts.DispatcherChainFunc != nil {
		guarded := runtime.NewGuardedChainEditor(chain)
		m.opts.DispatcherChainFunc(guarded)
		if violations := guarded.Violations(); len(violations) > 0 {
			chain = m.failClosedChain(canonicalEntries, violations)
		}
		if err := chain.Validate(); err != nil {
			chain = m.failClosedChain(canonicalEntries, []string{err.Error()})
		}
	} else if err := chain.Validate(); err != nil {
		// Even the canonical construction must validate (defense in depth).
		chain = m.failClosedChain(canonicalEntries, []string{err.Error()})
	}
	built := chain.Build(next)
	// A-15: restart-required outcomes (bootstrap credential rotation, update
	// swap) are observed outermost so the transport's completion callback
	// flushes the client response before the host's restart hook fires.
	if m.opts.OnRestartRequired != nil {
		built = runtime.NewRestartLink(m.opts.OnRestartRequired).Wrap(built)
	}
	m.chainedDisp.Store(built)
	return built
}

// ChainedDispatcher returns a late-binding accessor for the full dispatcher
// chain. The chain is built after boot (BuildInterfaces / desktop wiring),
// while schedules register during boot — schedule fires resolve the chain at
// tick time and fall back to the raw local dispatcher until it exists.
func (m *SystemModule) ChainedDispatcher() func() abstract.Dispatcher {
	return func() abstract.Dispatcher {
		if d, ok := m.chainedDisp.Load().(abstract.Dispatcher); ok && d != nil {
			return d
		}
		return m.disp
	}
}

func (m *SystemModule) AdminUserID() string  { return m.adminUserID }
func (m *SystemModule) AdminEmail() string   { return m.adminEmail }
func (m *SystemModule) Bootstrapped() bool   { return m.bootstrapped }
func (m *SystemModule) EphemeralKey() string { return m.ephemeralKey }
func (m *SystemModule) CredentialsProvider() abstract.CredentialsProvider {
	return m.providers.CredProv
}
func (m *SystemModule) UserModel() *model.SystemUsers { return m.providers.Users }

func (m *SystemModule) Start(ctx context.Context) error {
	m.providers.Scheduler.Start()
	m.opts.Logger.Info("system module started", zap.Stringer("scheduler", m.providers.Scheduler))
	return nil
}

func (m *SystemModule) Stop(ctx context.Context) error {
	m.providers.Scheduler.Stop()
	if m.providers.SchedCancel != nil {
		m.providers.SchedCancel()
	}
	if m.providers.RateStore != nil {
		m.providers.RateStore.Close()
		m.providers.RateStore = nil
	}
	if m.providers.BlobSvc != nil {
		if err := m.providers.BlobSvc.Close(); err != nil {
			m.opts.Logger.Warn("close blob service", zap.Error(err))
		}
		m.providers.BlobSvc = nil
	}
	// Shutdown hermes workflow runtime
	if m.providers.WorkflowRuntime != nil {
		if err := m.providers.WorkflowRuntime.Shutdown(ctx); err != nil {
			m.opts.Logger.Warn("shutdown workflow runtime", zap.Error(err))
		}
	}
	// S-15: flush and stop the audit buffer. The chain is built from
	// Wrap() clones, so without this the queued compliance entries (up
	// to 4096) silently vanished on every shutdown.
	if m.providers.AuditBuffer != nil {
		m.providers.AuditBuffer.Sync()
		m.providers.AuditBuffer.Close()
		m.providers.AuditBuffer = nil
	}
	m.opts.Logger.Info("system module stopped")
	return nil
}

func (m *SystemModule) Health(ctx context.Context) any {
	return map[string]any{
		"module":       "system",
		"bootstrapped": m.bootstrapped,
		"admin_email":  m.adminEmail,
	}
}

// allDefaultPolicies returns the static default policies plus, when self-update
// is configured, the updates service bindings.
func (m *SystemModule) allDefaultPolicies() []policies.Policy {
	return allDefaultPolicyBindings
}

func (m *SystemModule) SeedPolicies(ctx context.Context) error {
	if err := policies.SeedPolicies(ctx, m.providers.Policies, m.allDefaultPolicies()); err != nil {
		return fmt.Errorf("seed policies: %w", err)
	}

	if m.providers.LiveRules != nil {
		dbRules, err := m.providers.Policies.ListRules(ctx)
		if err != nil {
			return fmt.Errorf("list rules after seed: %w", err)
		}
		goRules := policies.GoDefaultRules()
		count := 0
		for _, r := range dbRules {
			if r.Expression == "" {
				continue
			}
			if _, hasGo := goRules[r.Name]; hasGo {
				continue
			}
			fn, err := policies.CompileCEL(r.Expression)
			if err != nil {
				continue
			}
			m.providers.LiveRules.Set(r.Name, fn)
			count++
		}
		m.opts.Logger.Info("seeded rules", zap.Int("rules", count))
	}

	return nil
}
