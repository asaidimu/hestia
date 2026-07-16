package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"

	corepkg "github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
)

func logQueryHandler(persist base.Persistence, defaultLimit int) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		doc := msg.Input()
		var body []byte
		if raw := doc.GetOr("payload.query", nil); raw != nil {
			body, _ = json.Marshal(raw)
		}

		var q *query.Query
		if len(body) == 0 {
			built := query.NewQueryBuilder().Build()
			q = &built
		} else {
			parsed, err := query.FromBytes(body)
			if err != nil {
				return nil, err
			}
			q = parsed
		}
		if q.Pagination == nil {
			includeTotal := true
			q.Pagination = &query.PaginationOptions{
				Type:         query.PaginationTypeOffset,
				Limit:        defaultLimit,
				IncludeTotal: &includeTotal,
				Order: []query.SortConfiguration{
					{Field: "occurred_at", Direction: query.SortDirectionDesc},
				},
			}
		} else {
			if q.Pagination.IncludeTotal == nil || !*q.Pagination.IncludeTotal {
				includeTotal := true
				q.Pagination.IncludeTotal = &includeTotal
			}
			if len(q.Pagination.Order) == 0 {
				q.Pagination.Order = []query.SortConfiguration{
					{Field: "occurred_at", Direction: query.SortDirectionDesc},
				}
			}
		}

		col, err := persist.Collection(ctx, auditCollectionName)
		if err != nil {
			return nil, err
		}
		result, err := col.Read(ctx, q)
		if err != nil {
			return nil, err
		}
		docs := make([]*data.Document, 0, len(result.Data))
		for _, d := range result.Data {
			docs = append(docs, d)
		}
		return &registration.Result{
			Page: &registration.Page{
				Documents:  docs,
				Pagination: result.PaginationInfo,
			},
		}, nil
	}
}

func logStreamHandler(persist base.Persistence) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		docCh := make(chan *data.Document, 64)

		col, err := persist.Collection(ctx, auditCollectionName)
		if err != nil {
			return nil, fmt.Errorf("get audit_log collection: %w", err)
		}

		go func() {
			select {
			case <-msg.InputChannel():
			case <-ctx.Done():
				close(docCh)
				return
			}

			var mu sync.Mutex
			closed := false

			subID := col.Subscribe(ctx, base.SubscriptionOptions{
				Event: base.DocumentCreateSuccess,
				Callback: func(_ context.Context, event base.PersistenceEvent) error {
					outMap, ok := event.Output.(map[string]any)
					if !ok {
						return nil
					}
					dataRaw, ok := outMap["data"]
					if !ok {
						return nil
					}
					dataMap, ok := dataRaw.(map[string]any)
					if !ok || dataMap == nil {
						return nil
					}
					mu.Lock()
					defer mu.Unlock()
					if closed {
						return nil
					}
					doc := data.MustNewDocument(dataMap, context.Background())
					select {
					case docCh <- doc:
					default:
					}
					return nil
				},
			})

			<-ctx.Done()

			mu.Lock()
			closed = true
			close(docCh)
			mu.Unlock()
			col.Unsubscribe(ctx, subID)
		}()

		return &registration.Result{DocumentChannel: docCh}, nil
	}
}
