package notifications

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const (
	collectionName = "_notifications_"
	readTTL        = 30 * 24 * time.Hour
)

type NotificationModel struct {
	persist base.Persistence
}

func NewNotificationModel(persist base.Persistence) *NotificationModel {
	return &NotificationModel{persist: persist}
}

func (m *NotificationModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persist.Collection(ctx, collectionName)
}

func (m *NotificationModel) filterExpired(docs data.DocumentSet) data.DocumentSet {
	now := time.Now().UnixMilli()
	filtered := make(data.DocumentSet, 0, len(docs))
	for _, d := range docs {
		expiresAt, err := d.Get("expires_at")
		if err != nil || expiresAt == nil {
			filtered = append(filtered, d)
			continue
		}
		exp, ok := expiresAt.(int64)
		if !ok {
			filtered = append(filtered, d)
			continue
		}
		if exp > now {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (m *NotificationModel) List(ctx context.Context, userID, tenantID string, limit, offset int) (data.DocumentSet, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

	qb := query.NewQueryBuilder().Where("user_id").Eq(userID).OrderByDesc("created_at")
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
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	return m.filterExpired(result.Data), nil
}

func (m *NotificationModel) CountUnread(ctx context.Context, userID, tenantID string) (int, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return 0, err
	}

	qb := query.NewQueryBuilder().Where("user_id").Eq(userID).Where("read").Eq(false)
	if tenantID != "" {
		qb = qb.Where("tenant_id").Eq(tenantID)
	}

	q := qb.Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return 0, err
	}
	unexpired := m.filterExpired(result.Data)
	return len(unexpired), nil
}

func (m *NotificationModel) MarkRead(ctx context.Context, notificationID string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}

	q := query.NewQueryBuilder().Where("_id").Eq(notificationID).Build()
	doc := data.Patch(map[string]any{
		"read":       true,
		"expires_at": time.Now().Add(readTTL).UnixMilli(),
	}).Document(ctx)
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set:    doc,
		Filter: q.Filters,
	})
	return err
}

func (m *NotificationModel) MarkAllRead(ctx context.Context, userID, tenantID string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}

	qb := query.NewQueryBuilder().Where("user_id").Eq(userID).Where("read").Eq(false)
	if tenantID != "" {
		qb = qb.Where("tenant_id").Eq(tenantID)
	}

	doc := data.Patch(map[string]any{
		"read":       true,
		"expires_at": time.Now().Add(readTTL).UnixMilli(),
	}).Document(ctx)
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set:    doc,
		Filter: qb.Build().Filters,
	})
	return err
}

func (m *NotificationModel) DeleteExpired(ctx context.Context) (int, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return 0, err
	}
	q := query.NewQueryBuilder().
		Where("expires_at").Lt(time.Now().UnixMilli()).
		Build()
	count, err := col.Delete(ctx, q.Filters, false)
	if err != nil {
		return 0, err
	}
	return count, nil
}
