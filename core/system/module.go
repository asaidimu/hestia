package system

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/blobs/staging"
	"github.com/asaidimu/updater"

	"github.com/asaidimu/hestia/core/abstract"
	featureaudit "github.com/asaidimu/hestia/core/system/audit"
	"github.com/asaidimu/hestia/core/system/auth"
	authsvc "github.com/asaidimu/hestia/core/system/auth"
	"github.com/asaidimu/hestia/core/system/blobs"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	"github.com/asaidimu/hestia/core/system/collections"
	operationsvc "github.com/asaidimu/hestia/core/system/operations"
	"github.com/asaidimu/hestia/core/system/policies"
	"github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/updates"
	"github.com/asaidimu/hestia/core/system/users"
	"github.com/asaidimu/hestia/core/system/users/model"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	module "github.com/asaidimu/hestia/core/runtime/module"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
)

type SystemModule struct {
	module.BaseModule

	opts      dispatch.SystemOptions
	cfg       *runtime.Config
	disp      *runtime.LocalDispatcher
	providers *ProviderSet

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
	m.providers = NewProviderSet(persist, m.cfg, m.opts.Logger)

	if err := m.providers.InitModels(ctx); err != nil {
		return fmt.Errorf("init models: %w", err)
	}

	m.providers.LiveSchedule = schedules.NewLiveSchedule(m.providers.Schedules, m.providers.Scheduler, m.disp, m.opts.Logger)
	if err := m.providers.LiveSchedule.Init(ctx); err != nil {
		return fmt.Errorf("init live schedule: %w", err)
	}

	sessionSvc := auth.NewSessionService(m.cfg.SessionSecret)
	resetSecret := m.cfg.SessionSecret + ":reset"
	m.providers.CredProv = auth.NewCredentialsProviderWithVersion(sessionSvc, resetSecret, func(ctx context.Context, userID string) (int, error) {
		user, err := m.providers.Users.GetByID(ctx, userID)
		if err != nil {
			return 0, nil
		}
		return user.GetTokenVersion(), nil
	})

	svc, err := blobutil.NewService(m.cfg.BlobsDir, m.opts.Logger)
	if err != nil {
		return fmt.Errorf("init blob service: %w", err)
	}
	m.providers.BlobSvc = svc

	if err := m.seedData(ctx); err != nil {
		return err
	}
	if err := m.initPermissions(ctx); err != nil {
		return err
	}
	if err := m.initAccessController(ctx); err != nil {
		return fmt.Errorf("init access controller: %w", err)
	}
	if err := m.initUserClaimsCache(ctx); err != nil {
		return fmt.Errorf("init user claims cache: %w", err)
	}

	m.providers.PolicyBridge = policies.NewPolicyStoreAdapter(m.providers.Policies, m.providers.PermMgr, m.providers.LiveRules)

	apiKeyAuth := auth.NewAPIKeyAuthenticator(m.providers.APIKeys, m.providers.LiveUsers, m.ephemeralKey, m.adminUserID, m.adminEmail, m.opts.Logger)

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
	if err := abstract.RegisterInstance[*featureaudit.AuditModel](rt, m.providers.Audit); err != nil {
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

	if m.ephemeralKey == "" {
		key := make([]byte, 16)
		if _, err := rand.Read(key); err == nil {
			m.ephemeralKey = hex.EncodeToString(key)
		}
	}

	return nil
}

func (m *SystemModule) initPermissions(ctx context.Context) error {
	opColl, err := m.providers.Persist.Collection(ctx, "_operation_policy_")
	if err != nil {
		m.opts.Logger.Warn("Failed to open _operation_policy_ collection, using static defaults", zap.Error(err))
		m.providers.PermMgr = policies.NewLivePermissionManager(nil, m.allDefaultPolicies())
		return nil
	}

	livePolicies, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[*policies.Policy]{
		Collection: opColl,
		Processor:  &policies.PolicyDocProcessor{},
		QueryKey:   "key",
		AutoLoad:   false,
	})
	if err != nil {
		m.opts.Logger.Warn("Failed to create live policy repository, using static defaults", zap.Error(err))
		m.providers.PermMgr = policies.NewLivePermissionManager(nil, m.allDefaultPolicies())
		return nil
	}
	m.providers.LivePolicies = livePolicies
	if liveColl, ok := livePolicies.(base.Collection); ok {
		m.providers.Policies.SetPolicyColl(liveColl)
	}
	m.providers.PermMgr = policies.NewLivePermissionManager(livePolicies, m.allDefaultPolicies())
	return nil
}

func (m *SystemModule) initAccessController(ctx context.Context) error {
	ruleColl, err := m.providers.Persist.Collection(ctx, "_iam_rule_")
	if err != nil {
		return fmt.Errorf("get _iam_rule_ collection: %w", err)
	}

	live, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[iam.FunctionRule]{
		Collection: ruleColl,
		Processor:  &policies.RuleDocProcessor{},
		QueryKey:   "name",
		AutoLoad:   false,
	})
	if err != nil {
		return fmt.Errorf("create live rule repository: %w", err)
	}
	m.providers.LiveRules = live

	if liveColl, ok := live.(base.Collection); ok {
		m.providers.Policies.SetRuleColl(liveColl)
	}

	for name, fn := range policies.GoDefaultRules() {
		live.Set(name, fn)
	}

	m.providers.AccessCtrl = iam.CreateAccessController(iam.AccessControllerOptions{
		Rules:    live,
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
			if key == "" {
				return query.NewQueryBuilder().
					Where("disabled").Neq(0).
					Build()
			}
			return query.NewQueryBuilder().
				Where(data.DocumentIDField).Eq(key).
				Where("disabled").Neq(0).
				Build()
		},
	})
	if err != nil {
		return fmt.Errorf("create live user claims repository: %w", err)
	}
	m.providers.LiveUsers = claims

	return nil
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

	chain := runtime.NewDispatcherChain(
		runtime.LinkEntry{Name: "bootstrap", Link: runtime.NewBootstrapDispatcher(nil, m.disp, func() bool { return m.bootstrapped })},
		runtime.LinkEntry{Name: "secure", Link: runtime.NewSecureDispatcher(nil, m.providers.PermMgr, m.providers.AccessCtrl)},
		runtime.LinkEntry{Name: "ratelimit", Link: runtime.NewRateLimitDispatcher(rateLimitLookup, sharedRateStore)},
		runtime.LinkEntry{Name: "throttle", Link: runtime.NewThrottleDispatcher(throttleLookup, m.disp, m.opts.Logger, sharedRateStore)},
		runtime.LinkEntry{Name: "tenant", Link: runtime.NewTenantDispatcher(nil, func(ctx context.Context) string {
			if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
				return claims.TenantID
			}
			return ""
		})},
		runtime.LinkEntry{Name: "blob", Link: blobutil.NewDispatcherLink(m.providers.BlobSvc)},
		runtime.LinkEntry{Name: "audit", Link: runtime.NewAuditDispatcherWithLogger(nil, m.providers.Audit, m.opts.Logger)},
	)
	if m.opts.DispatcherChainFunc != nil {
		m.opts.DispatcherChainFunc(chain)
	}
	return chain.Build(next)
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
