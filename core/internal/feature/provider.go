package feature

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/notification"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	blobutil "github.com/asaidimu/hestia/core/internal/feature/blobs/store"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/audit"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/blobs"
	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/internal/feature/operations"
	"github.com/asaidimu/hestia/core/internal/feature/policies"
	"github.com/asaidimu/hestia/core/internal/feature/notifications"
	"github.com/asaidimu/hestia/core/internal/feature/schedules"
	"github.com/asaidimu/hestia/core/internal/feature/settings"
	"github.com/asaidimu/hestia/core/internal/feature/tenants"
	"github.com/asaidimu/hestia/core/internal/feature/users"
)

// ProviderSet groups all feature models and runtime state that were
// previously fields on SystemModule.
type ProviderSet struct {
	Persist      base.Persistence
	Config       *runtime.Config
	Logger       *zap.Logger
	Bootstrapped bool

	Users        *users.UserModel
	APIKeys      *apikeys.APIKeyModel
	Policies         *policies.PolicyModel
	Seed             *operations.SeedModel
	Audit            *audit.AuditModel
	Tenants          *tenants.TenantModel
	Settings         *settings.SettingsModel
	Notifications    *notifications.NotificationModel
	Schedules        *schedules.ScheduleModel

	BlobSvc      *blobutil.Service
	Notifier     abstract.Notifier
	Scheduler    *scheduler.Scheduler
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
	ps.Users = users.NewUserModel(ps.Persist)
	ps.APIKeys = apikeys.NewAPIKeyModel(ps.Persist)
	ps.Seed = operations.NewSeedModel(ps.Persist)
	ps.Audit = audit.NewAuditModel(ps.Persist)
	ps.Tenants = tenants.NewTenantModel(ps.Persist)
	ps.Settings = settings.NewSettingsModel(ps.Persist)

	ps.Notifications = notifications.NewNotificationModel(ps.Persist)
	ps.Schedules = schedules.NewScheduleModel(ps.Persist)

	ps.Scheduler = scheduler.New(ps.Logger)
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

	return nil
}

// CollectRegistrations returns all message registrations from every feature.
func (ps *ProviderSet) CollectRegistrations(
	apiKeyAuth *auth.APIKeyAuthenticator,
	adminUserID string,
	onBootstrap func(),
	onReset func(),
	disp *runtime.LocalDispatcher,
) []abstract.MessageRegistration {
	var all, allRegs []abstract.MessageRegistration

	all = append(all, apikeys.Registrations(apikeys.Dependencies{
		APIKeyModel: ps.APIKeys,
	})...)

	all = append(all, audit.Registrations(audit.Dependencies{
		Persist: ps.Persist,
	})...)

	all = append(all, auth.Registrations(auth.Dependencies{
		UserModel:           ps.Users,
		CredentialsProvider: ps.CredProv,
		APIKeyAuth:          apiKeyAuth,
		APIKeyModel:         ps.APIKeys,
		AdminUserID:         adminUserID,
		SessionTTL:          ps.Config.SessionTTL,
		Notifier:            ps.Notifier,
		AppURL:              ps.AppURL,
	})...)

	all = append(all, blobs.Registrations(blobs.Dependencies{
		BlobStore:    ps.BlobSvc,
		PolicyBridge: ps.PolicyBridge,
		Registry:     disp,
	})...)

	all = append(all, collections.Registrations(collections.Dependencies{
		Persist:      ps.Persist,
		Registry:     disp,
		Logger:       ps.Logger,
		PolicyBridge: ps.PolicyBridge,
	})...)

	all = append(all, operations.Registrations(operations.Dependencies{
		Logger:        ps.Logger,
		Disp:          disp,
		Bootstrapped:  func() bool { return ps.Bootstrapped },
		OnBootstrap:   onBootstrap,
		OnReset:       onReset,
		AuditModel:    ps.Audit,
		Persist:       ps.Persist,
		Registrations: &allRegs,
		Scheduler:     ps.Scheduler,
	})...)

	all = append(all, notifications.Registrations(notifications.Dependencies{
		NotificationModel: ps.Notifications,
	})...)

	all = append(all, schedules.Registrations(schedules.Dependencies{
		ScheduleModel: ps.Schedules,
		LiveSchedule:  ps.LiveSchedule,
	})...)

	all = append(all, settings.Registrations(settings.Dependencies{
		SettingsModel: ps.Settings,
	})...)

	all = append(all, policies.Registrations(policies.Dependencies{
		PolicyModel: ps.Policies,
		PermManager: ps.PermMgr,
		LiveRules:   ps.LiveRules,
	})...)

	all = append(all, users.Registrations(users.Dependencies{
		UserModel: ps.Users,
		Persist:   ps.Persist,
	})...)

	allRegs = all
	return all
}


