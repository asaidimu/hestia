package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/app/audit"
	"github.com/asaidimu/hestia/internal/app/auth"
	"github.com/asaidimu/hestia/internal/app/collections"
	"github.com/asaidimu/hestia/internal/app/operations"
	"github.com/asaidimu/hestia/internal/app/policies"
	"github.com/asaidimu/hestia/internal/app/users"
	blobutil "github.com/asaidimu/hestia/internal/core/blobstore"
	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/interface/api"
	"github.com/asaidimu/hestia/internal/abstract"
)



type SystemModule struct {
	opts      Options
	cfg       *core.Config
	disp      *core.LocalDispatcher
	persist   base.Persistence
	jwtSvc     core.JWTService
	sessionSvc core.SessionService

	userModel      *users.UserModel
	apiKeyModel    *apikeys.APIKeyModel
	policyModel    *policies.PolicyModel
	seedModel      *operations.SeedModel
	accessLogModel *audit.AccessLogModel
	blocklistSvc   *auth.TokenBlocklistService
	permMgr        *policies.DBPermissionManager
	ac             iam.AccessController
	policyBridge   *policies.PolicyStoreAdapter

	blobSvc *blobutil.Service

	bootstrapped   bool
	ephemeralKey   string
	adminUserID    string
	adminEmail     string

	messages []abstract.MessageRegistration
}

type Options struct {
	OnBootstrapped    func()
	OnReset           func()
	Logger            *zap.Logger
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool

	// DispatcherHooks wraps the dispatcher chain with additional layers.
	// Applied after the default chain (Secure→Blob→AccessLog→Local).
	// Each hook receives and returns a Dispatcher.
	DispatcherHooks []func(abstract.Dispatcher) abstract.Dispatcher
}

func New(cfg *core.Config, disp *core.LocalDispatcher, opts Options) *SystemModule {
	return &SystemModule{
		opts:       opts,
		cfg:        cfg,
		disp:       disp,
		jwtSvc:     auth.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.ResetTokenTTL),
		sessionSvc: api.NewService(cfg.JWTSecret),
	}
}

func (m *SystemModule) Name() string { return "system" }

func (m *SystemModule) Setup(ctx context.Context, persist base.Persistence) error {
	m.persist = persist
	m.blocklistSvc = auth.NewTokenBlocklistService(persist)

	m.initModels(persist)

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
	m.initAccessController(ctx)

	m.policyBridge = policies.NewPolicyStoreAdapter(m.policyModel, m.permMgr, m.ac)

	apiKeyAuth := auth.NewAPIKeyAuthenticator(m.apiKeyModel, m.userModel, m.ephemeralKey, m.adminUserID, m.adminEmail)

	if err := m.registerExistingDocumentHandlers(ctx); err != nil {
		return fmt.Errorf("register document handlers: %w", err)
	}
	if err := m.permMgr.Reload(ctx); err != nil {
		m.opts.Logger.Warn("Failed to reload permissions after doc handler registration", zap.Error(err))
	}

	m.messages = collectFeatureRegistrations(m, apiKeyAuth)

	go m.purgeBlocklistLoop()

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

func (m *SystemModule) initModels(persist base.Persistence) {
	m.userModel = users.NewUserModel(persist)
	m.apiKeyModel = apikeys.NewAPIKeyModel(persist)
	m.policyModel = policies.NewPolicyModel(persist)
	m.seedModel = operations.NewSeedModel(persist)
	m.accessLogModel = audit.NewAccessLogModel(persist)
}

func (m *SystemModule) seedData(ctx context.Context) error {
	adminID, adminEmail, bootstrapped, err := auth.SeedAdmin(ctx, m.userModel, m.seedModel, m.opts.Logger,
		auth.SeedAdminOptions{
			Email:            m.opts.AdminEmail,
			Password:         m.opts.AdminPassword,
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

	m.permMgr = policies.NewDBPermissionManager(m.policyModel)
	policies.PopulatePermissionManager(m.permMgr, allDefaultOperations)

	if err := policies.SeedPolicies(ctx, m.policyModel, allDefaultOperations); err != nil {
		return fmt.Errorf("seed policies: %w", err)
	}
	return nil
}

func (m *SystemModule) initPermissions(ctx context.Context) error {
	if err := m.permMgr.Reload(ctx); err != nil {
		m.opts.Logger.Warn("Failed to reload permissions from DB, using fallback", zap.Error(err))
		policies.PopulatePermissionManager(m.permMgr, allDefaultOperations)
	}
	return nil
}

func (m *SystemModule) initAccessController(ctx context.Context) {
	m.ac = iam.CreateAccessController(iam.AccessControllerOptions{
		CacheTTL: 5 * time.Second,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.ac.LoadRules(policies.GoDefaultRules())
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
	var disp core.Dispatcher = core.NewSecureDispatcher(next, m.permMgr, m.ac)
	return disp
}

func (m *SystemModule) DispatcherChain(next core.Dispatcher) core.Dispatcher {
	var disp core.Dispatcher = core.NewSecureDispatcher(next, m.permMgr, m.ac)
	disp = blobutil.NewDispatcher(m.blobSvc, disp)
	disp = core.NewAccessLogDispatcher(disp, m.accessLogModel)
	for _, hook := range m.opts.DispatcherHooks {
		disp = hook(disp)
	}
	return disp
}

func (m *SystemModule) AdminUserID() string  { return m.adminUserID }
func (m *SystemModule) AdminEmail() string   { return m.adminEmail }
func (m *SystemModule) Bootstrapped() bool   { return m.bootstrapped }
func (m *SystemModule) EphemeralKey() string { return m.ephemeralKey }

func (m *SystemModule) purgeBlocklistLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if err := m.blocklistSvc.PurgeExpired(context.Background()); err != nil {
			m.opts.Logger.Warn("failed to purge expired blocklist entries", zap.Error(err))
		}
	}
}
