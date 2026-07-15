package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/schema"
	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
)

func IsSystemCollection(name string) bool {
	return len(name) > 2 && name[0] == '_' && name[len(name)-1] == '_'
}

type CollectionMeta struct {
	Name    string `json:"name"`
	Schema  *schema.Schema `json:"schema,omitempty"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

type QueryCommand struct {
	ctx        context.Context
	Collection string
	QDSL       *query.Query
}

func NewQueryCommand(ctx context.Context, collection string, q *query.Query) QueryCommand {
	return QueryCommand{ctx: ctx, Collection: collection, QDSL: q}
}

func (q QueryCommand) QueryName() string       { return "collections:document:query" }
func (q QueryCommand) Context() context.Context { return q.ctx }
func (q QueryCommand) ResourceContext() any     { return map[string]any{"collection": q.Collection} }

func NewCollectionListHandler(persist persistence.Persistence) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		names, err := persist.ListCollections(ctx)
		if err != nil {
			return nil, fmt.Errorf("list collections: %w", err)
		}

		metas := make([]CollectionMeta, 0, len(names))
		now := time.Now().UTC().Format(time.RFC3339)
		for _, name := range names {
			if IsSystemCollection(name) {
				continue
			}
			s, err := persist.Schema(ctx, name)
			if err != nil {
				continue
			}
			metas = append(metas, CollectionMeta{
				Name:    name,
				Schema:  s,
				Created: now,
				Updated: now,
			})
		}

		docs := make([]*data.Document, 0, len(metas))
		for _, meta := range metas {
			docs = append(docs, data.MustNewDocument(map[string]any{
				"name":    meta.Name,
				"schema":  meta.Schema,
				"created": meta.Created,
				"updated": meta.Updated,
			}, ctx)) // is this really neccessary ?
		}

		return &registration.Result{
			Page: &registration.Page{Documents: docs},
		}, nil
	}
}

func NewCollectionGetHandler(persist persistence.Persistence) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		s, err := persist.Schema(ctx, name)
		if err != nil {
			return &registration.Result{}, nil
		}
		if s == nil {
			return &registration.Result{}, nil
		}

		raw, _ := json.Marshal(s)
		now := time.Now().UTC().Format(time.RFC3339)
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"name":    name,
				"schema":  raw,
				"created": now,
				"updated": now,
			}, ctx),
		}, nil
	}
}

func NewNamedCollectionQueryHandler(collectionName string, persist persistence.Persistence) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		doc.Set("arguments.name", collectionName)
		return NewCollectionQueryHandler(persist)(ctx, msg)
	}
}

func NewCollectionQueryHandler(persist persistence.Persistence) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		var q *query.Query
		if raw := doc.GetOr("payload", nil); raw != nil {
			body, err := json.Marshal(raw)
			if err == nil && len(body) > 0 {
				parsed, err := query.FromBytes(body)
				if err != nil {
					return nil, fmt.Errorf("parse query: %w", err)
				}
				q = parsed
			}
		}
		if q == nil {
			built := query.NewQueryBuilder().Build()
			q = &built
		}
		if q.Pagination == nil {
			includeTotal := true
			q.Pagination = &query.PaginationOptions{
				Type:         query.PaginationTypeOffset,
				Limit:        100,
				IncludeTotal: &includeTotal,
			}
		} else if q.Pagination.IncludeTotal == nil || !*q.Pagination.IncludeTotal {
			includeTotal := true
			q.Pagination.IncludeTotal = &includeTotal
		}

		col, err := persist.Collection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("access collection %q: %w", name, err)
		}

		rctx := common.ContextWithCollectionName(ctx, name)
		result, err := col.Read(rctx, q)
		if err != nil {
			return nil, fmt.Errorf("query collection %q: %w", name, err)
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

func NewReadCollectionHandler(persist persistence.Persistence) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("name", "").(string)

		var q *query.Query
		if raw := doc.GetOr("query", nil); raw != nil {
			body, _ := json.Marshal(raw)
			if len(body) > 0 {
				parsed, err := query.FromBytes(body)
				if err != nil {
					return nil, fmt.Errorf("parse query: %w", err)
				}
				q = parsed
			}
		}
		if q == nil {
			built := query.NewQueryBuilder().Build()
			q = &built
		}
		if q.Pagination == nil {
			includeTotal := true
			q.Pagination = &query.PaginationOptions{
				Type:         query.PaginationTypeOffset,
				Limit:        100,
				IncludeTotal: &includeTotal,
			}
		} else if q.Pagination.IncludeTotal == nil || !*q.Pagination.IncludeTotal {
			includeTotal := true
			q.Pagination.IncludeTotal = &includeTotal
		}

		col, err := persist.Collection(ctx, name)
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

func NewDocumentGetHandler(persist persistence.Persistence) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		documentID, _ := doc.GetOr("arguments.doc_id", "").(string)

		col, err := persist.Collection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("access collection %q: %w", name, err)
		}

		built := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(documentID).Build()
		result, err := col.Read(ctx, &built)
		if err != nil {
			return nil, fmt.Errorf("get document %q from %q: %w", documentID, name, err)
		}

		if result.Count == 0 {
			return &registration.Result{}, nil
		}

		return &registration.Result{Document: result.Data[0]}, nil
	}
}
