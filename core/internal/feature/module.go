package feature

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

	"github.com/asaidimu/hestia/core/abstract"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	module "github.com/asaidimu/hestia/core/runtime/module"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime"
	blobutil "github.com/asaidimu/hestia/core/internal/feature/blobs/store"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/blobs"
	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/internal/feature/policies"
	"github.com/asaidimu/hestia/core/internal/feature/users"
)

type SystemModule struct {
	module.BaseModule

	opts     dispatch.SystemOptions
	cfg      *runtime.Config
	disp     *runtime.LocalDispatcher
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

func (m *SystemModule) Setup(ctx context.Context, persist base.Persistence) error {
	m.providers = NewProviderSet(persist, m.cfg, m.opts.Logger)

	if err := m.providers.InitModels(ctx); err != nil {
		return fmt.Errorf("init models: %w", err)
	}

	emailCh := m.cfg.Mailer.SMTPHost != ""
	fmt.Printf("[mailer] host=%q port=%d auth=%q from=%q from_name=%q app_url=%q email_channel=%v\n",
		m.cfg.Mailer.SMTPHost, m.cfg.Mailer.SMTPPort, m.cfg.Mailer.SMTPAuthType,
		m.cfg.Mailer.FromAddress, m.cfg.Mailer.FromName, m.cfg.AppURL,
		emailCh)

	sessionSvc := auth.NewSessionService(m.cfg.SessionSecret)
	resetSecret := m.cfg.SessionSecret + ":reset"
	m.providers.CredProv = auth.NewCredentialsProviderWithVersion(sessionSvc, resetSecret, func(ctx context.Context, userID string) (int, error) {
		user, err := m.providers.Users.GetByID(ctx, userID)
		if err != nil {
			return 0, nil
		}
		v, _ := user.GetInt("token_version")
		return v, nil
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
	m.messages = m.providers.CollectRegistrations(
		apiKeyAuth, m.adminUserID,
		m.bootstrapCallback(), m.resetCallback(),
		m.disp,
	)
	m.providers.Policies.SetKnownOps(collectAllKnownOperations())

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

func (m *SystemModule) Capabilities() []abstract.Capability {
	return []abstract.Capability{
		{
			Name:     "system",
			Messages: m.messages,
		},
	}
}

func (m *SystemModule) seedData(ctx context.Context) error {
	rootTenantID, err := m.seedTenants(ctx)
	if err != nil {
		return fmt.Errorf("seed tenants: %w", err)
	}

	adminEmail := m.opts.AdminEmail
	if adminEmail == "" {
		adminEmail = m.cfg.AdminEmail
	}
	adminPassword := m.opts.AdminPassword
	if adminPassword == "" {
		adminPassword = m.cfg.AdminPassword
	}
	adminID, adminEmail, bootstrapped, err := auth.SeedAdmin(ctx, m.providers.Users, m.providers.Seed, m.opts.Logger,
		auth.SeedAdminOptions{
			Email:             adminEmail,
			Password:          adminPassword,
			ForceBootstrapped: m.opts.ForceBootstrapped,
			TenantID:          rootTenantID,
		})
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	m.adminUserID = adminID
	m.adminEmail = adminEmail
	m.bootstrapped = bootstrapped

	if m.ephemeralKey == "" {
		key := make([]byte, 16)
		if _, err := rand.Read(key); err == nil {
			m.ephemeralKey = hex.EncodeToString(key)
		}
	}

	if !m.bootstrapped {
		if err := policies.SeedPolicies(ctx, m.providers.Policies, allDefaultPolicyBindings); err != nil {
			return fmt.Errorf("seed policies: %w", err)
		}
	}

	return nil
}

func (m *SystemModule) seedTenants(ctx context.Context) (string, error) {
	col, err := m.providers.Persist.Collection(ctx, "_tenant_")
	if err != nil {
		return "", fmt.Errorf("open tenant collection: %w", err)
	}
	allQuery := query.NewQueryBuilder().Build()
	result, err := col.Read(ctx, &allQuery)
	if err != nil {
		return "", fmt.Errorf("list tenants: %w", err)
	}
	if result.Count > 0 {
		return result.Data[0].ID(), nil
	}

	doc, err := m.providers.Tenants.Create(ctx, "Platform", "", nil)
	if err != nil {
		return "", fmt.Errorf("create tenant: %w", err)
	}
	tenantID := doc.ID()
	m.opts.Logger.Info("seeded root tenant", zap.String("tenant_id", tenantID))
	return tenantID, nil
}

func (m *SystemModule) initPermissions(ctx context.Context) error {
	opColl, err := m.providers.Persist.Collection(ctx, "_operation_policy_")
	if err != nil {
		m.opts.Logger.Warn("Failed to open _operation_policy_ collection, using static defaults", zap.Error(err))
		m.providers.PermMgr = policies.NewLivePermissionManager(nil, allDefaultPolicyBindings)
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
		m.providers.PermMgr = policies.NewLivePermissionManager(nil, allDefaultPolicyBindings)
		return nil
	}
	m.providers.LivePolicies = livePolicies
	if liveColl, ok := livePolicies.(base.Collection); ok {
		m.providers.Policies.SetPolicyColl(liveColl)
	}
	m.providers.PermMgr = policies.NewLivePermissionManager(livePolicies, allDefaultPolicyBindings)
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

	live, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[*users.UserClaims]{
		Collection: userColl,
		Processor:  &users.UserClaimsDocProcessor{},
		QueryKey:   "_id_",
		AutoLoad:   false,
		QueryFunc: func(key string) query.Query {
			if key == "" {
				return query.NewQueryBuilder().
					Where("deleted").NotExists().
					Build()
			}
			return query.NewQueryBuilder().
				Where(data.DocumentIDField).Eq(key).
				Where("deleted").NotExists().
				Build()
		},
	})
	if err != nil {
		return fmt.Errorf("create live user claims repository: %w", err)
	}
	m.providers.LiveUsers = live
	if userColl, ok := live.(base.Collection); ok {
		m.providers.Users.UseLiveCollection(userColl)
	}
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
			if err := m.providers.PolicyBridge.EnsureOperation(ctx, opName, op.RuleKey, op.Intent, op.Desc+" in "+ns.ID); err != nil {
				return fmt.Errorf("seed operation %s: %w", opName, err)
			}
		}
		if err := blobs.RegisterBlobHandlers(m.disp, m.providers.BlobSvc, ns.ID); err != nil {
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
	chain := runtime.NewDispatcherChain(
		runtime.LinkEntry{Name: "bootstrap", Link: runtime.NewBootstrapDispatcher(nil, m.disp, func() bool { return m.bootstrapped })},
		runtime.LinkEntry{Name: "secure", Link: runtime.NewSecureDispatcher(nil, m.providers.PermMgr, m.providers.AccessCtrl)},
		runtime.LinkEntry{Name: "tenant", Link: runtime.NewTenantDispatcher(nil, func(ctx context.Context) string {
			if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
				return claims.TenantID
			}
			return ""
		})},
		runtime.LinkEntry{Name: "blob", Link: blobutil.NewDispatcherLink(m.providers.BlobSvc)},
		runtime.LinkEntry{Name: "recovery", Link: runtime.NewRecoveryDispatcher(nil, m.opts.Logger)},
		runtime.LinkEntry{Name: "audit", Link: runtime.NewAuditDispatcherWithLogger(nil, m.providers.Audit, m.opts.Logger)},
	)
	if m.opts.DispatcherChainFunc != nil {
		m.opts.DispatcherChainFunc(chain)
	}
	return chain.Build(next)
}

func (m *SystemModule) AdminUserID() string                           { return m.adminUserID }
func (m *SystemModule) AdminEmail() string                            { return m.adminEmail }
func (m *SystemModule) Bootstrapped() bool                            { return m.bootstrapped }
func (m *SystemModule) EphemeralKey() string                          { return m.ephemeralKey }
func (m *SystemModule) CredentialsProvider() abstract.CredentialsProvider { return m.providers.CredProv }
func (m *SystemModule) UserModel() *users.UserModel                   { return m.providers.Users }

func (m *SystemModule) Start(ctx context.Context) error {
	m.providers.Scheduler.Start()
	m.opts.Logger.Info("system module started", zap.Stringer("scheduler", m.providers.Scheduler))
	return nil
}

func (m *SystemModule) Stop(ctx context.Context) error {
	m.providers.Scheduler.Stop()
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

func (m *SystemModule) SeedPolicies(ctx context.Context) error {
	if err := policies.SeedPolicies(ctx, m.providers.Policies, allDefaultPolicyBindings); err != nil {
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
