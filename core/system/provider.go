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
	auditmodel "github.com/asaidimu/hestia/core/system/audit/model"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	notificationsmodel "github.com/asaidimu/hestia/core/system/notifications/model"
	"github.com/asaidimu/hestia/core/system/operations"
	"github.com/asaidimu/hestia/core/system/policies"
	policiesmodel "github.com/asaidimu/hestia/core/system/policies/model"
	"github.com/asaidimu/hestia/core/system/schedules"
	schedulesmodel "github.com/asaidimu/hestia/core/system/schedules/model"
	settingsmodel "github.com/asaidimu/hestia/core/system/settings/model"
	tenantsmodel "github.com/asaidimu/hestia/core/system/tenants/model"
	"github.com/asaidimu/hestia/core/system/updates"
	"github.com/asaidimu/hestia/core/system/users"
	"github.com/asaidimu/hestia/core/system/users/model"

	"github.com/asaidimu/hermes/pkg/actionlog"
	hermesruntime "github.com/asaidimu/hermes/pkg/runtime"
	hermesstore "github.com/asaidimu/hermes/pkg/store"
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
	OpModel       *policiesmodel.SystemOperationPolicys
	RuleModel     *policiesmodel.SystemIamRules
	Seed          *operations.SeedModel
	Audit         *auditmodel.SystemAuditLogs
	AuditBuffer   *runtime.AuditBuffer
	Tenants       *tenantsmodel.SystemTenants
	Settings      *settingsmodel.SystemSettingss
	Notifications *notificationsmodel.SystemNotificationss
	Schedules     *schedulesmodel.SystemScheduledMessagess

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

	WorkflowRuntime *hermesruntime.WorkflowRuntime
}

func NewProviderSet(persist base.Persistence, cfg *runtime.Config, logger *zap.Logger) *ProviderSet {
	return &ProviderSet{
		Persist: persist,
		Config:  cfg,
		Logger:  logger,
	}
}

// InitModels creates all persistence-backed models. The policies model pair is
// constructed later in initPolicyInfra, once the LiveRepositories exist — the
// generated ModelCollections are built OVER them so writes through PolicyModel
// stay coherent with the permission manager's live caches.
func (ps *ProviderSet) InitModels(ctx context.Context) error {
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
	auditModel, err := auditmodel.InitSystemAuditLogsModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system audit logs model: %w", err)
	}
	ps.Audit = auditModel
	// S-15: construct the audit buffer eagerly. The previous lazy init
	// in AuditDispatcher.Buffer() raced concurrent first requests into
	// competing buffers/flush loops, and because the dispatcher chain is
	// built from Wrap() clones, nobody held the instance that would ever
	// flush it. One shared buffer lives here and Sync()/Close() happen
	// from SystemModule.Stop.
	ps.AuditBuffer = runtime.NewAuditBuffer(ps.Audit, ps.Logger)
	tenantsModel, err := tenantsmodel.InitSystemTenantsModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system tenants model: %w", err)
	}
	ps.Tenants = tenantsModel
	settingsModel, err := settingsmodel.InitSystemSettingssModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system settings model: %w", err)
	}
	ps.Settings = settingsModel

	notifsModel, err := notificationsmodel.InitSystemNotificationssModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system notifications model: %w", err)
	}
	ps.Notifications = notifsModel
	schedulesModel, err := schedulesmodel.InitSystemScheduledMessagessModel(ps.Persist, ps.Logger)
	if err != nil {
		return fmt.Errorf("init system scheduled messages model: %w", err)
	}
	ps.Schedules = schedulesModel

	schedCtx, schedCancel := context.WithCancel(context.Background())
	ps.SchedCancel = schedCancel
	ps.Scheduler = scheduler.New(schedCtx, ps.Logger)

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

	// Initialize hermes workflow runtime.
	//
	// v1.3.0 removed the bundled Hestia scheduler adapter: Hermes now takes a
	// plain scheduler.Scheduler, so the shim lives here. Hermes calls
	// Shutdown on the scheduler when the runtime stops; Hestia's scheduler is
	// owned by SystemModule.Stop, so the shim deliberately does not stop it.
	schedAdapter := &hestiaScheduler{sched: ps.Scheduler}

	// The action log is the durable source of truth for run history and
	// restart recovery. Without it, ListRuns/GetEvents only reflect the
	// current process and paused runs cannot be recovered.
	actionLog, err := actionlog.NewAnansiStore(ctx, ps.Persist, "_pipeline_actionlog_")
	if err != nil {
		return fmt.Errorf("init hermes action log: %w", err)
	}

	stateModels, err := hermesstore.NewAnansiStateCollection(ctx, ps.Persist, "_pipeline_state_")
	if err != nil {
		return fmt.Errorf("init hermes state collection: %w", err)
	}

	ps.WorkflowRuntime = hermesruntime.NewWorkflowRuntime(hermesruntime.Options{
		Scheduler: schedAdapter,
		StoreFactory: func() (hermesstore.Store, error) {
			return hermesstore.NewPersistentStore(stateModels, nil), nil
		},
		ActionLog: actionLog,
		Logger:    ps.Logger.Sugar(),
	})

	return nil
}

// hestiaScheduler adapts Hestia's scheduler to hermes' scheduler.Scheduler.
type hestiaScheduler struct {
	sched *scheduler.Scheduler
}

func (h *hestiaScheduler) Schedule(id, cron string, callback func(ctx context.Context)) error {
	h.sched.Register(id, cron, func(ctx context.Context) error {
		callback(ctx)
		return nil
	})
	return nil
}

func (h *hestiaScheduler) Cancel(id string) error {
	h.sched.Remove(id)
	return nil
}

// Shutdown is intentionally a no-op: Hestia owns the scheduler lifetime and
// stops it in SystemModule.Stop, after the workflow runtime has shut down.
func (h *hestiaScheduler) Shutdown(_ context.Context) error {
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
	schedule := cfg.CheckSchedule
	if schedule == "" {
		schedule = "@daily"
	}
	ps.Scheduler.Register("updates:check", schedule, func(ctx context.Context) error {
		return ps.Updates.RunScheduledCheck(runtimecontext.SystemContext(ctx))
	})
	return nil
}

// CollectRegistrations returns the generated service registrations so they are
// visible to documentation and the docs/list handler. Streaming ops included:
// every feature registers via the generated services.go collector.
func (ps *ProviderSet) CollectRegistrations(svcRegs []abstract.MessageRegistration) []abstract.MessageRegistration {
	return svcRegs
}
