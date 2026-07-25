package settings

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const settingsCollectionName = "_settings_"

type SettingsModel struct {
	persistence base.Persistence
}

func NewSettingsModel(persistence base.Persistence) *SettingsModel {
	return &SettingsModel{persistence: persistence}
}

func (m *SettingsModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persistence.Collection(ctx, settingsCollectionName)
}

func (m *SettingsModel) queryBuilder(tenantID string) *query.QueryBuilder {
	b := query.NewQueryBuilder()
	if tenantID != "" {
		b = b.Where("tenant_id").Eq(tenantID)
	}
	return b
}

func (m *SettingsModel) Get(ctx context.Context, tenantID, key string) (any, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}
	q := m.queryBuilder(tenantID).Where("key").Eq(key).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("setting not found")
	}
	return result.Data[0].Get("value")
}

func (m *SettingsModel) Set(ctx context.Context, tenantID, key string, value any, updatedBy string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}
	q := m.queryBuilder(tenantID).Where("key").Eq(key).Build()
	existing, err := col.Read(ctx, &q)
	if err != nil {
		return err
	}
	if existing.Count > 0 {
		doc := data.Patch(map[string]any{
			"value":      value,
			"updated_by": updatedBy,
		}).Document(ctx)
		_, err = col.Update(ctx, &base.CollectionUpdate{
			Set:    doc,
			Filter: q.Filters,
		})
		return err
	}
	doc := data.MustNewDocument(map[string]any{
		"key":        key,
		"value":      value,
		"updated_by": updatedBy,
	})
	if tenantID != "" {
		doc.Set("tenant_id", tenantID)
	}
	_, err = col.CreateOne(ctx, doc)
	return err
}

func (m *SettingsModel) Unset(ctx context.Context, tenantID, key string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}
	q := m.queryBuilder(tenantID).Where("key").Eq(key).Build().Filters
	_, err = col.Delete(ctx, q, false)
	return err
}

func (m *SettingsModel) All(ctx context.Context, tenantID string) (map[string]any, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}
	q := m.queryBuilder(tenantID).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, result.Count)
	for _, doc := range result.Data {
		key, _ := doc.GetString("key")
		val, _ := doc.Get("value")
		out[key] = val
	}
	return out, nil
}
