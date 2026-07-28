package schedules

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const collectionName = "_scheduled_messages_"

type ScheduleModel struct {
	persist base.Persistence
}

func NewScheduleModel(persist base.Persistence) *ScheduleModel {
	return &ScheduleModel{persist: persist}
}

func (m *ScheduleModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persist.Collection(ctx, collectionName)
}

func (m *ScheduleModel) Create(ctx context.Context, doc *data.Document) (string, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return "", err
	}

	doc.Set("sent", false)
	doc.Set("created_at", time.Now().UnixMilli())

	result, err := col.CreateOne(ctx, doc)
	if err != nil {
		return "", err
	}
	return result.Data.ID(), nil
}

func (m *ScheduleModel) List(ctx context.Context, tenantID string, limit, offset int) ([]*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

	qb := query.NewQueryBuilder().OrderByAsc("send_at")
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
	return result.Data, nil
}

func (m *ScheduleModel) Get(ctx context.Context, id string) (*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

	q := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, nil
	}
	return result.Data[0], nil
}

func (m *ScheduleModel) Delete(ctx context.Context, id string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}

	q := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build()
	_, err = col.Delete(ctx, q.Filters, false)
	return err
}

func (m *ScheduleModel) FindDue(ctx context.Context) ([]*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

	q := query.NewQueryBuilder().
		Where("send_at").Lte(time.Now().UnixMilli()).
		Where("sent").Eq(false).
		Build()

	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (m *ScheduleModel) MarkSent(ctx context.Context, id string, sendErr error) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}

	updates := map[string]any{
		"sent":    true,
		"sent_at": time.Now().UnixMilli(),
	}
	if sendErr != nil {
		updates["error_"] = sendErr.Error()
	}

	q := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build()
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set:    data.Patch(updates).Document(ctx),
		Filter: q.Filters,
	})
	return err
}
