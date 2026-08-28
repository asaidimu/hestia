package model

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

// CreateDefinition persists a new workflow definition, setting timestamps.
func (m *SystemWorkflowDefinitions) CreateDefinition(ctx context.Context, doc data.Documenter) (*document.Document, error) {
	now := time.Now().UnixMilli()
	doc.Set("created_at", now)
	doc.Set("updated_at", now)

	def := document.New(&SystemWorkflowDefinition{
		Name:        mustGetString(doc, "name"),
		Description: ptrStr(mustGetString(doc, "description")),
		Nodes:       mustGetMap(doc, "nodes"),
		Edges:       mustGetMap(doc, "edges"),
		TenantID:    ptrStr(mustGetString(doc, "tenant_id")),
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})

	created, err := m.Create(ctx, def)
	if err != nil {
		return nil, err
	}
	return created.MustDocument(), nil
}

// GetDefinition returns a single workflow definition by ID, or nil if not found.
func (m *SystemWorkflowDefinitions) GetDefinition(ctx context.Context, id string) (*document.Document, error) {
	q := query.NewQueryBuilder().Where("_id_").Eq(id).Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0].MustDocument(), nil
}

// ListDefinitions returns all workflow definitions.
func (m *SystemWorkflowDefinitions) ListDefinitions(ctx context.Context) ([]*document.Document, error) {
	q := query.NewQueryBuilder().Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	out := make([]*document.Document, len(results))
	for i, s := range results {
		out[i] = s.MustDocument()
	}
	return out, nil
}

// ListDefinitionsByTenant returns workflow definitions for a tenant.
func (m *SystemWorkflowDefinitions) ListDefinitionsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*document.Document, error) {
	qb := query.NewQueryBuilder().OrderByAsc("created_at")
	if tenantID != "" {
		qb = qb.Where("tenant_id").Eq(tenantID)
	}
	if limit > 0 {
		qb = qb.Limit(limit)
	}
	if offset > 0 {
		qb = qb.Offset(offset)
	}

	q := qb.Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	out := make([]*document.Document, len(results))
	for i, s := range results {
		out[i] = s.MustDocument()
	}
	return out, nil
}

// UpdateDefinition patches fields on a workflow definition by ID.
func (m *SystemWorkflowDefinitions) UpdateDefinition(ctx context.Context, id string, fields map[string]any) error {
	q := query.NewQueryBuilder().Where("_id_").Eq(id).Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("workflow definition not found")
	}
	existing := results[0]
	now := time.Now().UnixMilli()
	existing.UpdatedAt = &now

	if v, ok := fields["name"]; ok {
		if s, ok := v.(string); ok {
			existing.Name = s
		}
	}
	if v, ok := fields["description"]; ok {
		if s, ok := v.(string); ok {
			existing.Description = &s
		}
	}
	if v, ok := fields["nodes"]; ok {
		if m, ok := v.(map[string]any); ok {
			existing.Nodes = m
		}
	}
	if v, ok := fields["edges"]; ok {
		if m, ok := v.(map[string]any); ok {
			existing.Edges = m
		}
	}

	_, err = m.Update(ctx, id, existing)
	return err
}

// DeleteDefinition removes a workflow definition by ID.
func (m *SystemWorkflowDefinitions) DeleteDefinition(ctx context.Context, id string) error {
	return m.DeleteByID(ctx, id)
}

func mustGetString(d data.Documenter, key string) string {
	v, _ := d.GetString(key)
	return v
}

func mustGetMap(d data.Documenter, key string) map[string]any {
	v, _ := d.Get(key)
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
