package system

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/query"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/system/auth"
	"github.com/asaidimu/hestia/core/system/notifications"
	"github.com/asaidimu/hestia/core/system/policies"
)

// SeedOptions groups optional parameters for SeedAll.
type SeedOptions struct {
	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool
}

// SeedResult holds values produced by SeedAll that callers may need.
type SeedResult struct {
	AdminUserID  string
	AdminEmail   string
	Bootstrapped bool
	TenantID     string
}

// SeedAll seeds all initial data in dependency order:
// 1. root tenant (if none exist)
// 2. admin user + bootstrapped flag
// 3. policies + rules (only when not already bootstrapped)
// 4. notification templates.
// All operations are idempotent.
func SeedAll(ctx context.Context, ps *ProviderSet, opts SeedOptions, defaults []policies.Policy) (result SeedResult, err error) {
	tenantID, err := seedTenant(ctx, ps)
	if err != nil {
		return result, fmt.Errorf("seed tenant: %w", err)
	}

	adminID, adminEmail, bootstrapped, err := auth.SeedAdmin(ctx, ps.Users, ps.Seed, ps.Logger,
		auth.SeedAdminOptions{
			Email:             opts.AdminEmail,
			Password:          opts.AdminPassword,
			ForceBootstrapped: opts.ForceBootstrapped,
			TenantID:          tenantID,
		})
	if err != nil {
		return result, fmt.Errorf("seed admin: %w", err)
	}

	if !bootstrapped {
		if err := policies.SeedPolicies(ctx, ps.Policies, defaults); err != nil {
			return result, fmt.Errorf("seed policies: %w", err)
		}
	}

	if err := notifications.SeedNotificationTemplates(ctx, ps.Persist); err != nil {
		return result, fmt.Errorf("seed notification templates: %w", err)
	}

	return SeedResult{
		AdminUserID:  adminID,
		AdminEmail:   adminEmail,
		Bootstrapped: bootstrapped,
		TenantID:     tenantID,
	}, nil
}

func seedTenant(ctx context.Context, ps *ProviderSet) (string, error) {
	col, err := ps.Persist.Collection(ctx, "_tenant_")
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

	doc, err := ps.Tenants.CreateTenant(ctx, "Platform", "", nil)
	if err != nil {
		return "", fmt.Errorf("create tenant: %w", err)
	}
	ps.Logger.Info("seeded root tenant", zap.String("tenant_id", doc.ID))
	return doc.ID, nil
}
