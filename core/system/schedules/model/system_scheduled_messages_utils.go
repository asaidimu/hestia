package model

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

// Create persists a new scheduled message, setting created_at to now.
func (m *SystemScheduledMessagess) CreateSchedule(ctx context.Context, doc data.Documenter) (*document.Document, error) {
	doc.Set("created_at", time.Now().UnixMilli())

	scheduled := document.New(&SystemScheduledMessages{
		UserID:   mustGetString(doc, "user_id"),
		Message:  mustGetString(doc, "message"),
		Cron:     mustGetString(doc, "cron"),
		Input:    mustGetMap(doc, "input"),
		TenantID: ptrStr(mustGetString(doc, "tenant_id")),
	})

	if v, err := doc.Get("disabled"); err == nil {
		if b, ok := v.(bool); ok {
			scheduled.Disabled = &b
		}
	}

	if v, err := doc.Get("protected"); err == nil {
		if b, ok := v.(*bool); ok && b != nil {
			scheduled.Protected = b
		} else if b, ok := v.(bool); ok {
			scheduled.Protected = &b
		}
	}

	created, err := m.Create(ctx, scheduled)
	if err != nil {
		return nil, err
	}
	return created.MustDocument(), nil
}

// GetSchedule returns a single scheduled message by ID, or nil if not found.
func (m *SystemScheduledMessagess) GetSchedule(ctx context.Context, id string) (*document.Document, error) {
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

// UpdateSchedule patches fields on a scheduled message by ID.
func (m *SystemScheduledMessagess) UpdateSchedule(ctx context.Context, id string, fields map[string]any) error {
	q := query.NewQueryBuilder().Where("_id_").Eq(id).Build()
	results, err := m.Read(ctx, &q)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("schedule not found")
	}
	existing := results[0]
	if v, ok := fields["message"]; ok {
		if s, ok := v.(string); ok {
			existing.Message = s
		}
	}
	if v, ok := fields["cron"]; ok {
		if s, ok := v.(string); ok {
			existing.Cron = s
		}
	}
	if v, ok := fields["user_id"]; ok {
		if s, ok := v.(string); ok {
			existing.UserID = s
		}
	}
	if v, ok := fields["input"]; ok {
		if m, ok := v.(map[string]any); ok {
			existing.Input = m
		}
	}
	if v, ok := fields["disabled"]; ok {
		if b, ok := v.(bool); ok {
			existing.Disabled = &b
		}
	}
	_, err = m.Update(ctx, id, existing)
	return err
}

// ListSchedules returns all scheduled messages.
func (m *SystemScheduledMessagess) ListSchedules(ctx context.Context) ([]*document.Document, error) {
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

// ListSchedulesByTenant returns scheduled messages for a user, ordered by created_at.
func (m *SystemScheduledMessagess) ListSchedulesByTenant(ctx context.Context, userID string, limit, offset int) ([]*document.Document, error) {
	qb := query.NewQueryBuilder().
		Where("user_id").Eq(userID).
		OrderByAsc("created_at")
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

// DeleteSchedule removes a scheduled message by ID.
func (m *SystemScheduledMessagess) DeleteSchedule(ctx context.Context, id string) error {
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
