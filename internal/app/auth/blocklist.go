package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

const blocklistCollectionName = "_token_blocklist_"

type TokenBlocklistService struct {
	persistence base.Persistence
}

func NewTokenBlocklistService(persistence base.Persistence) *TokenBlocklistService {
	return &TokenBlocklistService{persistence: persistence}
}

func (s *TokenBlocklistService) Collection(ctx context.Context) (base.Collection, error) {
	return s.persistence.Collection(ctx, blocklistCollectionName)
}

func (s *TokenBlocklistService) Blocklist(ctx context.Context, jti string, exp int64, userID string) error {
	col, err := s.Collection(ctx)
	if err != nil {
		return fmt.Errorf("access collection: %w", err)
	}

	doc := data.MustNewDocument(map[string]any{
		"jti":     jti,
		"exp":     exp,
		"user_id": userID,
	})

	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create blocklist entry: %w", err)
	}
	return nil
}

func (s *TokenBlocklistService) IsBlocklisted(ctx context.Context, jti string) (bool, error) {
	col, err := s.Collection(ctx)
	if err != nil {
		return false, fmt.Errorf("access collection: %w", err)
	}

	now := time.Now().Unix()
	q := query.NewQueryBuilder().Where("jti").Eq(jti).Where("exp").Gte(now).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return false, fmt.Errorf("query blocklist: %w", err)
	}

	return result.Count > 0, nil
}

func (s *TokenBlocklistService) PurgeExpired(ctx context.Context) error {
	col, err := s.Collection(ctx)
	if err != nil {
		return fmt.Errorf("access collection: %w", err)
	}
	now := time.Now().Unix()
	filter := query.NewQueryBuilder().Where("exp").Lt(now).Build().Filters
	_, err = col.Delete(ctx, filter, false)
	return err
}
