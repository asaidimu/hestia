package model

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

// CreateTenant creates a new tenant with the given name, domain, and metadata.
func (m *SystemTenants) CreateTenant(ctx context.Context, name, domain string, metadata map[string]any) (*SystemTenant, error) {
	doc := document.New(&SystemTenant{
		Name:     name,
		Domain:   &domain,
		Metadata: metadata,
	})
	created, err := m.Create(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return created, nil
}

// GetByID returns the tenant with the given ID.
func (m *SystemTenants) GetByID(ctx context.Context, id string) (*SystemTenant, error) {
	tenant, err := m.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("query tenant by id: %w", err)
	}
	return tenant, nil
}

// GetByDomain returns the active tenant with the given domain.
func (m *SystemTenants) GetByDomain(ctx context.Context, domain string) (*SystemTenant, error) {
	q := query.NewQueryBuilder().
		Where("domain").Eq(domain).
		Limit(1).
		Build()

	tenants, err := m.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query tenant by domain: %w", err)
	}
	if len(tenants) == 0 {
		return nil, fmt.Errorf("tenant not found")
	}
	return tenants[0], nil
}
