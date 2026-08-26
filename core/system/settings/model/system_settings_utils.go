package model

import (
	"context"
	"errors"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

// ErrNotFound is returned by Get when no setting matches the key.
var ErrNotFound = errors.New("setting not found")

// Get returns the value for the given tenant+key pair.
func (m *SystemSettingss) Get(ctx context.Context, tenantID, key string) (any, error) {
	q := m.buildQuery(tenantID).Where("key").Eq(key).Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results[0].Value, nil
}

// Set creates or updates a setting for the given tenant+key.
func (m *SystemSettingss) Set(ctx context.Context, tenantID, key string, value any, updatedBy string) error {
	q := m.buildQuery(tenantID).Where("key").Eq(key).Build()
	existing, err := m.Read(ctx, &q)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		doc := document.New(&SystemSettings{
			Key:       key,
			Value:     toRecord(value),
			UpdatedBy: &updatedBy,
			TenantID:  ptrStr(tenantID),
		})
		_, err = m.Update(ctx, existing[0].ID, doc)
		return err
	}

	doc := document.New(&SystemSettings{
		Key:       key,
		Value:     toRecord(value),
		UpdatedBy: &updatedBy,
		TenantID:  ptrStr(tenantID),
	})
	_, err = m.Create(ctx, doc)
	return err
}

// Unset deletes a setting by tenant+key.
func (m *SystemSettingss) Unset(ctx context.Context, tenantID, key string) error {
	q := m.buildQuery(tenantID).Where("key").Eq(key).Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return err
	}
	for _, s := range results {
		if err := m.DeleteByID(ctx, s.ID); err != nil {
			return err
		}
	}
	return nil
}

// All returns all settings for the given tenant as a key→value map.
func (m *SystemSettingss) All(ctx context.Context, tenantID string) (map[string]any, error) {
	q := m.buildQuery(tenantID).Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(results))
	for _, s := range results {
		out[s.Key] = fromRecord(s.Value)
	}
	return out, nil
}

func (m *SystemSettingss) buildQuery(tenantID string) *query.QueryBuilder {
	b := query.NewQueryBuilder()
	if tenantID != "" {
		b = b.Where("tenant_id").Eq(tenantID)
	}
	return b
}

func toRecord(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{"_": v}
}

func fromRecord(v map[string]any) any {
	if v == nil {
		return nil
	}
	if inner, ok := v["_"]; ok && len(v) == 1 {
		return inner
	}
	return v
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
