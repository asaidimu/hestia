package model

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/query"
)

// readTTL is how long a notification stays visible after it is read; expired
// notifications are filtered from list/count results and removed by the hourly
// cleanup job.
const readTTL = 30 * 24 * time.Hour

// List returns the user's notifications, newest first, filtering out expired
// ones that the cleanup job has not deleted yet.
func (m *SystemNotificationss) List(ctx context.Context, userID, tenantID string, limit, offset int) ([]*SystemNotifications, error) {
	qb := query.NewQueryBuilder().
		Where("user_id").Eq(userID).
		OrderByDesc("created_at")
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
	docs, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	return filterExpired(docs), nil
}

// CountUnread returns the number of unread notifications for the user.
func (m *SystemNotificationss) CountUnread(ctx context.Context, userID, tenantID string) (int, error) {
	qb := query.NewQueryBuilder().
		Where("user_id").Eq(userID).
		Where("read").Eq(false)
	if tenantID != "" {
		qb = qb.Where("tenant_id").Eq(tenantID)
	}
	q := qb.Build()
	docs, err := m.Read(ctx, &q)
	if err != nil {
		return 0, err
	}
	return len(filterExpired(docs)), nil
}

// MarkRead marks a single notification as read and stamps its expiry.
func (m *SystemNotificationss) MarkRead(ctx context.Context, notificationID string) error {
	read := true
	expiresAt := time.Now().Add(readTTL).UnixMilli()
	_, err := m.Update(ctx, notificationID, &SystemNotifications{
		Read:      &read,
		ExpiresAt: &expiresAt,
	})
	return err
}

// MarkAllRead marks every unread notification of the user as read.
func (m *SystemNotificationss) MarkAllRead(ctx context.Context, userID, tenantID string) error {
	qb := query.NewQueryBuilder().
		Where("user_id").Eq(userID).
		Where("read").Eq(false)
	if tenantID != "" {
		qb = qb.Where("tenant_id").Eq(tenantID)
	}
	read := true
	expiresAt := time.Now().Add(readTTL).UnixMilli()
	_, err := m.UpdateMany(ctx, qb.Build().Filters, &SystemNotifications{
		Read:      &read,
		ExpiresAt: &expiresAt,
	})
	return err
}

// DeleteExpired removes notifications whose expiry has passed.
func (m *SystemNotificationss) DeleteExpired(ctx context.Context) (int, error) {
	q := query.NewQueryBuilder().
		Where("expires_at").Lt(time.Now().UnixMilli()).
		Build()
	return m.DeleteMany(ctx, q.Filters, false)
}

// filterExpired drops notifications past their expiry (readTTL after read).
func filterExpired(docs []*SystemNotifications) []*SystemNotifications {
	now := time.Now().UnixMilli()
	out := make([]*SystemNotifications, 0, len(docs))
	for _, d := range docs {
		if d.ExpiresAt == nil || *d.ExpiresAt > now {
			out = append(out, d)
		}
	}
	return out
}
