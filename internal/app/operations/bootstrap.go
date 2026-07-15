package operations

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const seedCollName = "_seed_"

type SeedModel struct {
	persistence base.Persistence
}

func NewSeedModel(persistence base.Persistence) *SeedModel {
	return &SeedModel{persistence: persistence}
}

func (m *SeedModel) Get(ctx context.Context, key string) (string, error) {
	col, err := m.persistence.Collection(ctx, seedCollName)
	if err != nil {
		return "", fmt.Errorf("access _seed collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("key").Eq(key).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return "", fmt.Errorf("query seed: %w", err)
	}
	if result.Count == 0 {
		return "", nil
	}
	return result.Data[0].GetString("value")
}

func (m *SeedModel) Set(ctx context.Context, key, value string) error {
	col, err := m.persistence.Collection(ctx, seedCollName)
	if err != nil {
		return fmt.Errorf("access _seed collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("key").Eq(key).Build()
	existing, err := col.Read(ctx, &q)
	if err != nil {
		return fmt.Errorf("query seed: %w", err)
	}

	fields := map[string]any{"key": key, "value": value}

	if existing.Count > 0 {
		docID := existing.Data[0].ID()
		setDoc := data.Patch(fields).Document(ctx)
		_, err = col.Update(ctx, &base.CollectionUpdate{
			Set:    setDoc,
			Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(docID).Build().Filters,
		})
		if err != nil {
			return fmt.Errorf("update seed: %w", err)
		}
		return nil
	}

	doc := data.MustNewDocument(fields)
	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create seed: %w", err)
	}
	return nil
}
