package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/app/core"
	blobutil "github.com/asaidimu/hestia/app/core/blobstore"
	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/app/audit"
	"github.com/asaidimu/hestia/internal/app/auth"
	"github.com/asaidimu/hestia/internal/app/blobs"
	"github.com/asaidimu/hestia/internal/app/collections"
	"github.com/asaidimu/hestia/internal/app/operations"
	"github.com/asaidimu/hestia/internal/app/policies"
	"github.com/asaidimu/hestia/internal/app/users"
)

type SystemModule struct {
	opts      Options
	cfg       *core.Config
	disp      *core.LocalDispatcher
	persist   base.Persistence
	credProv  abstract.CredentialsProvider

	userModel      *users.UserModel
	apiKeyModel    *apikeys.APIKeyModel
	policyModel    *policies.PolicyModel
	seedModel      *operations.SeedModel
	auditModel     *audit.AuditModel
	permMgr        core.ReloadablePermissionManager
	ac             iam.AccessController
	policyBridge   *policies.PolicyStoreAdapter
	liveRules      collection.LiveCollection[iam.FunctionRule]
	livePolicies   collection.LiveCollection[*policies.Policy]

	blobSvc *blobutil.Service

	bootstrapped bool
	ephemeralKey string
	adminUserID  string
	adminEmail   string

	messages []abstract.MessageRegistration
}

type Options struct {
	OnBootstrapped    func()
	OnReset           func()
	Logger            *zap.Logger
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool

	DispatcherHooks []func(abstract.Dispatcher) abstract.Dispatcher
}

func New(cfg *core.Config, disp *core.LocalDispatcher, opts Options) *SystemModule {
	return &SystemModule{
		opts: opts,
		cfg:  cfg,
		disp: disp,
	}
}

func (m *SystemModule) Name() string { return "system" }

func (m *SystemModule) Setup(ctx context.Context, persist base.Persistence) error {
	m.persist = persist
	if err := m.initModels(ctx, persist); err != nil {
		return err
	}

	sessionSvc := auth.NewSessionService(m.cfg.SessionSecret)
	resetSecret := m.cfg.SessionSecret + ":reset"
	m.credProv = auth.NewCredentialsProvider(sessionSvc, resetSecret)

	blobSvc, err := blobutil.NewService(m.cfg.BlobsDir, m.opts.Logger)
	if err != nil {
		return fmt.Errorf("init blob service: %w", err)
	}
	m.blobSvc = blobSvc

	if err := m.seedData(ctx); err != nil {
		return err
	}
	if err := m.initPermissions(ctx); err != nil {
		return err
	}
	if err := m.initAccessController(ctx); err != nil {
		return fmt.Errorf("init access controller: %w", err)
	}

	m.policyBridge = policies.NewPolicyStoreAdapter(m.policyModel, m.permMgr, m.liveRules)

	apiKeyAuth := auth.NewAPIKeyAuthenticator(m.apiKeyModel, m.userModel, m.ephemeralKey, m.adminUserID, m.adminEmail)

	if err := m.registerExistingDocumentHandlers(ctx); err != nil {
		return fmt.Errorf("register document handlers: %w", err)
	}
	if err := m.registerExistingBlobHandlers(ctx); err != nil {
		return fmt.Errorf("register blob handlers: %w", err)
	}
	m.messages = collectFeatureRegistrations(m, apiKeyAuth)
	m.policyModel.SetKnownOps(collectAllKnownOperations())

	return nil
}

func (m *SystemModule) Capabilities() []abstract.Capability {
	return []abstract.Capability{
		{
			Name:     "system",
			Messages: m.messages,
		},
	}
}

func (m *SystemModule) initModels(ctx context.Context, persist base.Persistence) error {
	opColl, err := persist.Collection(ctx, "_operation_policy_")
	if err != nil {
		return fmt.Errorf("open policy collection: %w", err)
	}
	ruleColl, err := persist.Collection(ctx, "_iam_rule_")
	if err != nil {
		return fmt.Errorf("open rule collection: %w", err)
	}

	m.policyModel = policies.NewPolicyModel(opColl, ruleColl, nil)
	m.userModel = users.NewUserModel(persist)
	m.apiKeyModel = apikeys.NewAPIKeyModel(persist)
	m.seedModel = operations.NewSeedModel(persist)
	m.auditModel = audit.NewAuditModel(persist)
	return nil
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
	adminID, adminEmail, bootstrapped, err := auth.SeedAdmin(ctx, m.userModel, m.seedModel, m.opts.Logger,
		auth.SeedAdminOptions{
			Email:            adminEmail,
			Password:         adminPassword,
			ForceBootstrapped: m.opts.ForceBootstrapped,
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
		if err := policies.SeedPolicies(ctx, m.policyModel, allDefaultPolicyBindings); err != nil {
			return fmt.Errorf("seed policies: %w", err)
		}
	}
	return nil
}

func (m *SystemModule) initPermissions(ctx context.Context) error {
	opColl, err := m.persist.Collection(ctx, "_operation_policy_")
	if err != nil {
		m.opts.Logger.Warn("Failed to open _operation_policy_ collection, using static defaults", zap.Error(err))
		m.permMgr = policies.NewLivePermissionManager(nil, allDefaultPolicyBindings)
		return nil
	}

	livePolicies, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[*policies.Policy]{
		Collection: opColl,
		Processor:  &policies.PolicyDocProcessor{},
		QueryKey:   "operation",
		Active:     false,
	})
	if err != nil {
		m.opts.Logger.Warn("Failed to create live policy repository, using static defaults", zap.Error(err))
		m.permMgr = policies.NewLivePermissionManager(nil, allDefaultPolicyBindings)
		return nil
	}
	m.livePolicies = livePolicies
	if liveColl, ok := livePolicies.(base.Collection); ok {
		m.policyModel.SetPolicyColl(liveColl)
	}
	m.permMgr = policies.NewLivePermissionManager(livePolicies, allDefaultPolicyBindings)
	return nil
}

func (m *SystemModule) initAccessController(ctx context.Context) error {
	ruleColl, err := m.persist.Collection(ctx, "_iam_rule_")
	if err != nil {
		return fmt.Errorf("get _iam_rule_ collection: %w", err)
	}

	live, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[iam.FunctionRule]{
		Collection: ruleColl,
		Processor:  &policies.RuleDocProcessor{},
		QueryKey:   "name",
		Active:     false,
	})
	if err != nil {
		return fmt.Errorf("create live rule repository: %w", err)
	}
	m.liveRules = live

	if liveColl, ok := live.(base.Collection); ok {
		m.policyModel.SetRuleColl(liveColl)
	}

	for name, fn := range policies.GoDefaultRules() {
		live.Set(name, fn)
	}

	m.ac = iam.CreateAccessController(iam.AccessControllerOptions{
		Rules:    live,
		CacheTTL: 0,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return nil
}

func (m *SystemModule) registerExistingBlobHandlers(ctx context.Context) error {
	namespaces, err := m.blobSvc.ListNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, ns := range namespaces {
		for _, op := range blobs.BlobOps() {
			opName := "blob." + ns.ID + "." + op.Suffix
			if err := m.policyBridge.EnsureOperation(ctx, opName, op.RuleKey, op.Intent, op.Desc+" in "+ns.ID); err != nil {
				return fmt.Errorf("seed operation %s: %w", opName, err)
			}
		}
		if err := blobs.RegisterBlobHandlers(m.disp, m.blobSvc, ns.ID); err != nil {
			return fmt.Errorf("register blob handlers for %q: %w", ns.ID, err)
		}
	}
	if err := m.policyBridge.ReloadPolicies(ctx); err != nil {
		return fmt.Errorf("reload policies: %w", err)
	}
	return nil
}

func (m *SystemModule) registerExistingDocumentHandlers(ctx context.Context) error {
	names, err := m.persist.ListCollections(ctx)
	if err != nil {
		return err
	}
	for _, name := range names {
		if strings.HasPrefix(name, "_") {
			continue
		}
		if err := collections.RegisterDocumentHandlers(m.disp, m.persist, name); err != nil {
			return fmt.Errorf("register doc handlers for %q: %w", name, err)
		}
	}
	return nil
}

func (m *SystemModule) SecureDispatcher(next core.Dispatcher) core.Dispatcher {
	return core.NewSecureDispatcher(next, m.permMgr, m.ac)
}

func (m *SystemModule) DispatcherChain(next core.Dispatcher) core.Dispatcher {
	var disp core.Dispatcher = core.NewSecureDispatcher(next, m.permMgr, m.ac)
	disp = blobutil.NewDispatcher(m.blobSvc, disp)
	disp = core.NewAuditDispatcher(disp, m.auditModel)
	for _, hook := range m.opts.DispatcherHooks {
		disp = hook(disp)
	}
	return disp
}

func (m *SystemModule) AdminUserID() string                        { return m.adminUserID }
func (m *SystemModule) AdminEmail() string                         { return m.adminEmail }
func (m *SystemModule) Bootstrapped() bool                         { return m.bootstrapped }
func (m *SystemModule) EphemeralKey() string                        { return m.ephemeralKey }
func (m *SystemModule) CredentialsProvider() abstract.CredentialsProvider { return m.credProv }
func (m *SystemModule) UserModel() *users.UserModel                 { return m.userModel }


func (m *SystemModule) SeedPolicies(ctx context.Context) error {
	if err := policies.SeedPolicies(ctx, m.policyModel, allDefaultPolicyBindings); err != nil {
		return fmt.Errorf("seed policies: %w", err)
	}

	if m.liveRules != nil {
		dbRules, err := m.policyModel.ListRules(ctx)
		if err != nil {
			return fmt.Errorf("list rules after seed: %w", err)
		}
		count := 0
		for _, r := range dbRules {
			if r.Expression == "" {
				continue
			}
			fn, err := policies.CompileCEL(r.Expression)
			if err != nil {
				continue
			}
			m.liveRules.Set(r.Name, fn)
			count++
		}
		m.opts.Logger.Info("seeded rules", zap.Int("rules", count))
	}

	return nil
}
