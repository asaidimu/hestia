package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/utils"
)

const notificationCollectionName = "_notification_"

type NotificationModel struct {
	persistence base.Persistence
}

func NewNotificationModel(persistence base.Persistence) *NotificationModel {
	return &NotificationModel{persistence: persistence}
}

func (m *NotificationModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persistence.Collection(ctx, notificationCollectionName)
}

type Notification struct {
	ID         string
	TenantID   string
	UserID     string
	Type       string
	Title      string
	Body       string
	Link       string
	Read       bool
	ReadAt     int64
	Metadata   map[string]any
}

func docToNotification(doc *data.Document) *Notification {
	n := &Notification{
		ID:       doc.ID(),
		TenantID: mustString(doc, "tenant_id"),
		UserID:   mustString(doc, "user_id"),
		Type:     mustString(doc, "type"),
		Title:    mustString(doc, "title"),
		Body:     mustString(doc, "body"),
		Link:     mustString(doc, "link"),
	}
	read, _ := doc.GetBool("read")
	n.Read = read
	readAt, _ := doc.GetInt("read_at")
	n.ReadAt = int64(readAt)
	if m, err := doc.Get("metadata"); err == nil && m != nil {
		n.Metadata, _ = m.(map[string]any)
	}
	return n
}

func mustString(doc *data.Document, field string) string {
	s, _ := doc.GetString(field)
	return s
}

func (m *NotificationModel) Create(ctx context.Context, tenantID, userID, typ, title, body, link string, metadata map[string]any) (*Notification, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{
		"user_id":  userID,
		"type":     typ,
		"title":    title,
		"body":     body,
		"link":     link,
		"read":     false,
		"metadata": metadata,
	}
	if tenantID != "" {
		fields["tenant_id"] = tenantID
	}
	doc := data.MustNewDocument(fields)
	result, err := col.CreateOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}
	return docToNotification(result.Data), nil
}

func (m *NotificationModel) queryBuilder(tenantID string) *query.QueryBuilder {
	b := query.NewQueryBuilder()
	if tenantID != "" {
		b = b.Where("tenant_id").Eq(tenantID)
	}
	return b
}

func (m *NotificationModel) List(ctx context.Context, tenantID, userID string, offset, limit int) ([]*Notification, int, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := m.queryBuilder(tenantID).Where("user_id").Eq(userID).Build()
	q.Pagination = &query.PaginationOptions{
		Type:         query.PaginationTypeOffset,
		Offset:       &offset,
		Limit:        limit,
		IncludeTotal: utils.PrimitivePtr(true),
	}
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*Notification, result.Count)
	for i, doc := range result.Data {
		out[i] = docToNotification(doc)
	}
	total := 0
	if result.Total != nil {
		total = *result.Total
	}
	return out, total, nil
}

func (m *NotificationModel) UnreadCount(ctx context.Context, tenantID, userID string) (int, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return 0, err
	}
	q := m.queryBuilder(tenantID).Where("user_id").Eq(userID).Where("read").Eq(false).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return 0, err
	}
	return result.Count, nil
}

func (m *NotificationModel) MarkRead(ctx context.Context, id, tenantID, userID string) error {
	return m.markRead(ctx, id, tenantID, userID, true)
}

func (m *NotificationModel) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	doc := data.Patch(map[string]any{
		"read":    true,
		"read_at": now,
	}).Document(ctx)
	q := m.queryBuilder(tenantID).Where("user_id").Eq(userID).Where("read").Eq(false).Build()
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set: doc,
		Filter: q.Filters,
	})
	return err
}

func (m *NotificationModel) markRead(ctx context.Context, id, tenantID, userID string, read bool) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	doc := data.Patch(map[string]any{
		"read":    read,
		"read_at": now,
	}).Document(ctx)
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set: doc,
		Filter: m.queryBuilder(tenantID).Where(data.DocumentIDField).Eq(id).Where("user_id").Eq(userID).Build().Filters,
	})
	return err
}
