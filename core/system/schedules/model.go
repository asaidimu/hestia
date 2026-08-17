package schedules

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
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

func (m *ScheduleModel) Create(ctx context.Context, doc data.Documenter) (*document.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

	doc.Set("created_at", time.Now().UnixMilli())

	result, err := col.CreateOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	return asDocument(result.Data)
}

func (m *ScheduleModel) Update(ctx context.Context, id string, fields map[string]any) error {
	col, err := m.collection(ctx)
	if err != nil {
		return err
	}

	q := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build()
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set:    data.Patch(fields).Document(ctx),
		Filter: q.Filters,
	})
	return err
}

func (m *ScheduleModel) Get(ctx context.Context, id string) (*document.Document, error) {
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
	return asDocument(result.Data[0])
}

func (m *ScheduleModel) List(ctx context.Context) ([]*document.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

	q := query.NewQueryBuilder().Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	return asDocuments(result.Data)
}

func (m *ScheduleModel) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*document.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, err
	}

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
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	return asDocuments(result.Data)
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

// asDocument narrows a single persistence-layer documenter to its concrete
// *document.Document, which is what anansi collections always return.
func asDocument(d data.Documenter) (*document.Document, error) {
	doc, ok := d.(*document.Document)
	if !ok {
		return nil, fmt.Errorf("persistence returned %T, want *document.Document", d)
	}
	return doc, nil
}

// asDocuments narrows a persistence-layer document set to its concrete
// *document.Document elements.
func asDocuments(docs data.DocumentSet) ([]*document.Document, error) {
	out := make([]*document.Document, len(docs))
	for i, d := range docs {
		doc, ok := d.(*document.Document)
		if !ok {
			return nil, fmt.Errorf("persistence returned %T, want *document.Document", d)
		}
		out[i] = doc
	}
	return out, nil
}
