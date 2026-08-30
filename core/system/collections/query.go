package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema"

	"github.com/asaidimu/hestia/core/abstract"
)

func IsSystemCollection(name string) bool {
	return len(name) > 2 && name[0] == '_' && name[len(name)-1] == '_'
}

type CollectionMeta struct {
	Name    string         `json:"name"`
	Schema  *schema.Schema `json:"schema,omitempty"`
	Created string         `json:"created"`
	Updated string         `json:"updated"`
}

type QueryCommand struct {
	ctx        context.Context
	Collection string
	QDSL       *query.Query
}

func NewQueryCommand(ctx context.Context, collection string, q *query.Query) QueryCommand {
	return QueryCommand{ctx: ctx, Collection: collection, QDSL: q}
}

func (q QueryCommand) QueryName() string        { return "collections:document:query" }
func (q QueryCommand) Context() context.Context { return q.ctx }
func (q QueryCommand) ResourceContext() any     { return map[string]any{"collection": q.Collection} }

func NewCollectionListHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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

		docs := make([]*document.Document, 0, len(metas))
		for _, meta := range metas {
			doc, err := document.New(&CollectionMetaView{
				Name:    meta.Name,
				Schema:  meta.Schema.AsMap(),
				Created: meta.Created,
				Updated: meta.Updated,
			}).Document()
			if err != nil {
				return nil, err
			}
			docs = append(docs, doc)
		}

		return &abstract.Result{
			Page: &abstract.Page{Documents: docs},
		}, nil
	}
}

func NewCollectionGetHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		s, err := persist.Schema(ctx, name)
		if err != nil {
			return &abstract.Result{}, nil
		}
		if s == nil {
			return &abstract.Result{}, nil
		}

		now := time.Now().UTC().Format(time.RFC3339)
		resultDoc, err := document.New(&CollectionMetaView{
			Name:    name,
			Schema:  s.AsMap(),
			Created: now,
			Updated: now,
		}).Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: resultDoc}, nil
	}
}

func NewNamedCollectionQueryHandler(collectionName string, persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return runCollectionQuery(ctx, msg, collectionName, persist)
	}
}

func NewCollectionQueryHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		return runCollectionQuery(ctx, msg, name, persist)
	}
}

func runCollectionQuery(ctx context.Context, msg abstract.Message, name string, persist persistence.Persistence) (*abstract.Result, error) {
	doc := msg.Input()

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
	// S-17: a client-supplied pagination.limit used to pass through
	// untouched — limit: 1000000000 materializes the whole collection.
	// Clamp to a server-side maximum; clients page instead.
	const maxQueryLimit = 1000
	if q.Pagination.Limit > maxQueryLimit {
		q.Pagination.Limit = maxQueryLimit
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

	docs := make([]*document.Document, 0, len(result.Data))
	for _, d := range result.Data {
		doc, ok := d.(*document.Document)
		if !ok {
			return nil, fmt.Errorf("persistence returned %T, want *document.Document", d)
		}
		docs = append(docs, doc)
	}
	return &abstract.Result{
		Page: &abstract.Page{
			Documents:  docs,
			Pagination: result.PaginationInfo,
		},
	}, nil
}

func NewReadCollectionHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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
		docs := make([]*document.Document, 0, len(result.Data))
		for _, d := range result.Data {
			doc, ok := d.(*document.Document)
			if !ok {
				return nil, fmt.Errorf("persistence returned %T, want *document.Document", d)
			}
			docs = append(docs, doc)
		}
		return &abstract.Result{
			Page: &abstract.Page{
				Documents:  docs,
				Pagination: result.PaginationInfo,
			},
		}, nil
	}
}

func NewDocumentGetHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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
			return &abstract.Result{}, nil
		}

		r, ok := result.Data[0].(*document.Document)
		if !ok {
			return nil, fmt.Errorf("persistence returned %T, want *document.Document", result.Data[0])
		}
		return &abstract.Result{Document: r}, nil
	}
}
