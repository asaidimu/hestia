package feature

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	blobutil "github.com/asaidimu/hestia/core/blobstore"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/audit"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/blobs"
	"github.com/asaidimu/hestia/core/internal/feature/collections"
	"github.com/asaidimu/hestia/core/internal/feature/operations"
	"github.com/asaidimu/hestia/core/internal/feature/policies"
	"github.com/asaidimu/hestia/core/internal/feature/settings"
	"github.com/asaidimu/hestia/core/internal/feature/tenants"
	"github.com/asaidimu/hestia/core/internal/feature/users"
	"github.com/asaidimu/hestia/core/internal/feature/notifications"
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
	Policies     *policies.PolicyModel
	Seed         *operations.SeedModel
	Audit        *audit.AuditModel
	Tenants      *tenants.TenantModel
	Settings     *settings.SettingsModel
	Notifications *notifications.NotificationModel

	BlobSvc      *blobutil.Service
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
		AdminUserID:         adminUserID,
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
