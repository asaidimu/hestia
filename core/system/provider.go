package system

import (
	"context"
	"fmt"
	"os"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-iam/v2/iam"
	"github.com/asaidimu/updater"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/notification"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	apikeysmodel "github.com/asaidimu/hestia/core/system/apikeys/model"
	"github.com/asaidimu/hestia/core/system/audit"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	notificationsmodel "github.com/asaidimu/hestia/core/system/notifications/model"
	"github.com/asaidimu/hestia/core/system/operations"
	"github.com/asaidimu/hestia/core/system/policies"
	"github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/settings"
	"github.com/asaidimu/hestia/core/system/tenants"
	"github.com/asaidimu/hestia/core/system/updates"
	"github.com/asaidimu/hestia/core/system/users"
	"github.com/asaidimu/hestia/core/system/users/model"
)

// ProviderSet groups all feature models and runtime state that were
// previously fields on SystemModule.
type ProviderSet struct {
	Persist      base.Persistence
	Config       *runtime.Config
	Logger       *zap.Logger
	Bootstrapped bool

	Users         *model.SystemUsers
	APIKeys       *apikeysmodel.SystemAPIKeys
	Policies      *policies.PolicyModel
	Seed          *operations.SeedModel
	Audit         *audit.AuditModel
	Tenants       *tenants.TenantModel
	Settings      *settings.SettingsModel
	Notifications *notificationsmodel.SystemNotificationss
	Schedules     *schedules.ScheduleModel

	UpdStore *updates.Store
	Updater  *updater.Updater
	Updates  *updates.UpdatesService
	UpdExe   string
	UpdData  string

	BlobSvc      *blobutil.Service
	RateStore    *ratestore.InMemoryStore
	Notifier     abstract.Notifier
	Scheduler    *scheduler.Scheduler
	SchedCancel  context.CancelFunc
	LiveSchedule *schedules.LiveSchedule
	AppURL       string
	CredProv     abstract.CredentialsProvider
	PolicyBridge *policies.PolicyStoreAdapter
	PermMgr      runtime.ReloadablePermissionManager
	AccessCtrl   iam.AccessController
	LiveRules    collection.LiveCollection[iam.FunctionRule]
	LivePolicies collection.LiveCollection[*policies.Policy]
	LiveUsers    collection.LiveCollection[*users.UserClaims]
}

func NewProviderSet(persist base.Persistence, cfg *runtime.Config, logger *zap.Logger) *ProviderSet {
	return &ProviderSet{
		Persist: persist,
		Config:  cfg,
		Logger:  logger,
	}
}

// InitModels creates all persistence-backed models.
func (ps *ProviderSet) InitModels(ctx context.Context) error {
	opColl, err := ps.Persist.Collection(ctx, "_operation_policy_")
	if err != nil {
		return err
	}
	ruleColl, err := ps.Persist.Collection(ctx, "_iam_rule_")
	if err != nil {
		return err
	}

	ps.Policies = policies.NewPolicyModel(opColl, ruleColl, nil)
	systemUsers, err := model.InitSystemUsersModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system users model: %w", err)
	}
	ps.Users = systemUsers
	apiKeys, err := apikeysmodel.InitSystemAPIKeysModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system api keys model: %w", err)
	}
	ps.APIKeys = apiKeys
	ps.Seed = operations.NewSeedModel(ps.Persist)
	ps.Audit = audit.NewAuditModel(ps.Persist)
	ps.Tenants = tenants.NewTenantModel(ps.Persist)
	ps.Settings = settings.NewSettingsModel(ps.Persist)

	notifsModel, err := notificationsmodel.InitSystemNotificationssModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system notifications model: %w", err)
	}
	ps.Notifications = notifsModel
	ps.Schedules = schedules.NewScheduleModel(ps.Persist)

	schedCtx, schedCancel := context.WithCancel(context.Background())
	ps.SchedCancel = schedCancel
	ps.Scheduler = scheduler.New(schedCtx, ps.Logger)
	ps.Scheduler.Register("notifications:cleanup", "@every 1h", func(ctx context.Context) error {
		deleted, err := ps.Notifications.DeleteExpired(ctx)
		if err != nil {
			ps.Logger.Warn("cleanup: expired notifications failed", zap.Error(err))
		} else {
			ps.Logger.Debug("cleanup: deleted expired notifications", zap.Int("count", deleted))
		}
		return nil
	})

	resolver := notification.NewSettingsResolver(ps.Persist)

	ps.Notifier = notification.New(resolver)
	ps.Notifier.RegisterChannel(notification.NewInAppChannel(ps.Persist, resolver))
	if cfg := ps.Config.Mailer; cfg.SMTPHost != "" {
		m, err := runtime.NewMailer(cfg)
		if err != nil {
			return fmt.Errorf("init mailer: %w", err)
		}
		ps.Notifier.RegisterChannel(notification.NewEmailChannel(m, resolver))
	}
	ps.AppURL = ps.Config.AppURL

	if err := ps.initUpdates(ctx); err != nil {
		return err
	}

	return nil
}

// initUpdates wires the self-update service when SelfUpdate is configured:
// builds the updater over the resolved provider, persists to the settings
// collection, reconciles any pre-boot swap remnants, registers the scheduled
// check, and exposes the service for registration.
func (ps *ProviderSet) initUpdates(ctx context.Context) error {
	cfg := ps.Config.SelfUpdate
	if cfg == nil {
		return nil
	}
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = ps.Config.DataDir
	}
	exe := cfg.ExecutablePath
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}
	}
	store := updates.NewStore(ps.Settings)
	u, err := updater.New(cfg.Provider, updater.Config{
		Version:          ps.Config.Version,
		DataDir:          dataDir,
		ExecutablePath:   exe,
		ForwardArguments: cfg.ForwardArguments,
		Store:            store,
	})
	if err != nil {
		return fmt.Errorf("init updater: %w", err)
	}
	ps.Updater = u
	ps.UpdStore = store
	ps.UpdExe = exe
	ps.UpdData = dataDir
	if err := ps.UpdStore.Reconcile(ctx, ps.Config.Version); err != nil {
		ps.Logger.Warn("updates: reconcile pending update failed", zap.Error(err))
	}
	ps.Updates = updates.NewServiceFromDeps(
		ps.Updater,
		ps.UpdStore,
		ps.Notifier,
		ps.Users,
		ps.Logger,
		ps.AppURL,
		ps.Config.Mailer.SMTPHost != "",
		cfg.AutoApply,
		ps.Config.Version,
		cfg.SystemdMode,
		exe,
		dataDir,
	)
	if cfg.CheckSchedule != "" {
		ps.Scheduler.Register("updates:check", cfg.CheckSchedule, func(ctx context.Context) error {
			return ps.Updates.RunScheduledCheck(runtimecontext.SystemContext(ctx))
		})
	}
	return nil
}

// CollectRegistrations returns all message registrations from every feature,
// plus any new-style service registrations (svcRegs) so they are visible to
// documentation and the docs/list handler.
func (ps *ProviderSet) CollectRegistrations(svcRegs []abstract.MessageRegistration) []abstract.MessageRegistration {
	// system:audit:log:stream cannot be generated yet — the generator skips
	// streaming annotations until dispatch.HandleInputStream lands — so
	// audit's stream registration is hand-written in core/system/audit/stream.go.
	// Every other feature's registrations (including updates) live in
	// core/system/* and arrive via svcRegs from CollectServiceRegistrations.
	all := audit.StreamRegistration(ps.Persist)
	return append(all, svcRegs...)
}
