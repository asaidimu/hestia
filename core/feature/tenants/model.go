package tenants

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const tenantCollectionName = "_tenant_"

type TenantModel struct {
	persistence base.Persistence
}

func NewTenantModel(persistence base.Persistence) *TenantModel {
	return &TenantModel{persistence: persistence}
}

func (m *TenantModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persistence.Collection(ctx, tenantCollectionName)
}

func (m *TenantModel) Create(ctx context.Context, name, domain string, metadata map[string]any) (data.Documenter, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access tenant collection: %w", err)
	}
	doc := data.MustNewDocument(map[string]any{
		"name":     name,
		"domain":   domain,
		"status":   "active",
		"metadata": metadata,
	})
	result, err := col.CreateOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return result.Data, nil
}

func (m *TenantModel) GetByID(ctx context.Context, id string) (data.Documenter, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access tenant collection: %w", err)
	}
	q := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query tenant by id: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("tenant not found")
	}
	return result.Data[0], nil
}

func (m *TenantModel) GetByDomain(ctx context.Context, domain string) (data.Documenter, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access tenant collection: %w", err)
	}
	q := query.NewQueryBuilder().Where("domain").Eq(domain).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query tenant by domain: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("tenant not found")
	}
	return result.Data[0], nil
}
