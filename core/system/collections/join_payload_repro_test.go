package collections_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/stretchr/testify/require"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/collections"
)

type authorRow struct {
	ID   string `anansi:"id,required=true"`
	Name string `anansi:"name,required=true"`
}

type bookRow struct {
	ID       string  `anansi:"id,required=true"`
	Title    string  `anansi:"title,required=true"`
	AuthorID string  `anansi:"author_id,required=true"`
	Price    float64 `anansi:"price"`
}

func schemaFrom[T any](t *testing.T, name string) *definition.Schema {
	t.Helper()
	raw, err := data.SchemaFrom[T]()
	require.NoError(t, err)
	s, err := definition.FromJSON(raw)
	require.NoError(t, err)
	s.Name = name
	return s
}

// denormalize takes flat dotted-key maps from anansi joins and nests them
// under the table alias prefix. e.g. {"author.name": "Alice"} →
// {"author": {"name": "Alice"}}
func denormalize(flat map[string]any) map[string]any {
	nested := make(map[string]any)
	for k, v := range flat {
		parts := strings.SplitN(k, ".", 2)
		if len(parts) == 2 {
			table := parts[0]
			field := parts[1]
			if _, ok := nested[table]; !ok {
				nested[table] = make(map[string]any)
			}
			nested[table].(map[string]any)[field] = v
		} else {
			nested[k] = v
		}
	}
	return nested
}

func TestJoinPayloadRepro(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	queryHandler := collections.NewCollectionQueryHandler(p)

	_, err := p.CreateCollection(ctx, schemaFrom[authorRow](t, "Author"))
	require.NoError(t, err)
	_, err = p.CreateCollection(ctx, schemaFrom[bookRow](t, "Book"))
	require.NoError(t, err)

	authors, err := p.Collection(ctx, "Author")
	require.NoError(t, err)
	for _, a := range []map[string]any{
		{"id": "a1", "name": "Alice"},
		{"id": "a2", "name": "Bob"},
	} {
		_, err = authors.CreateOne(ctx, data.MustNewDocument(a))
		require.NoError(t, err)
	}
	books, err := p.Collection(ctx, "Book")
	require.NoError(t, err)
	for _, b := range []map[string]any{
		{"id": "b1", "title": "Go in Action", "author_id": "a1", "price": 39.99},
		{"id": "b2", "title": "Advanced Go", "author_id": "a1", "price": 49.99},
		{"id": "b3", "title": "Rust Basics", "author_id": "a2", "price": 29.99},
	} {
		_, err = books.CreateOne(ctx, data.MustNewDocument(b))
		require.NoError(t, err)
	}

	// ── 1. No alias: keys use full table name prefix ──────────────
	fmt.Println("=== 1. No alias (default) ===")
	noAliasQ := query.NewQueryBuilder().
		From("Book").
		LeftJoin("Author").On(query.QueryFilter{
		Condition: &query.FilterCondition{
			Field:    "Book.author_id",
			Operator: query.ComparisonOperatorEq,
			Value: query.FilterValue{
				FieldRefVal: &query.FieldReference{Type: "field", Field: "Author.id"},
			},
		},
	}).End().
		ThenSortBy("Book._id_", query.SortDirectionAsc).
		Build()

	inputNoAlias := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"name": "Book"},
		"payload":   noAliasQ,
	})
	resultNoAlias, err := queryHandler(ctx, dispatch.NewMessage("system:collections:document:query", ctx, inputNoAlias))
	require.NoError(t, err)
	for i, doc := range resultNoAlias.Page.Documents {
		raw, _ := json.MarshalIndent(doc.Data(), "  ", "  ")
		fmt.Printf("  [%d]: %s\n", i, raw)
	}

	// ── 2. Table alias: keys use alias prefix ─────────────────────
	fmt.Println("\n=== 2. Table alias (Alias: author) ===")
	tableAliasQ := query.NewQueryBuilder().
		From("Book").
		LeftJoin("Author").Alias("author").On(query.QueryFilter{
		Condition: &query.FilterCondition{
			Field:    "Book.author_id",
			Operator: query.ComparisonOperatorEq,
			Value: query.FilterValue{
				FieldRefVal: &query.FieldReference{Type: "field", Field: "Author.id"},
			},
		},
	}).End().
		ThenSortBy("Book._id_", query.SortDirectionAsc).
		Build()

	inputTableAlias := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"name": "Book"},
		"payload":   tableAliasQ,
	})
	resultTableAlias, err := queryHandler(ctx, dispatch.NewMessage("system:collections:document:query", ctx, inputTableAlias))
	require.NoError(t, err)

	fmt.Println("  Flat keys from anansi:")
	for i, doc := range resultTableAlias.Page.Documents {
		raw, _ := json.MarshalIndent(doc.Data(), "  ", "  ")
		fmt.Printf("  [%d]: %s\n", i, raw)
	}

	// ── 3. Denormalization attempt: flat → nested ──────────────────
	fmt.Println("\n=== 3. Client-side denormalization (flat → nested) ===")
	for i, doc := range resultTableAlias.Page.Documents {
		nested := denormalize(doc.Data())
		raw, _ := json.MarshalIndent(nested, "  ", "  ")
		fmt.Printf("  [%d]: %s\n", i, raw)
	}

	// Verify that anansi returns flat dotted keys (the core finding)
	firstDoc := resultTableAlias.Page.Documents[0].Data()
	_, hasAuthorName := firstDoc["author.name"]
	_, hasFlatAuthorID := firstDoc["author.id"]
	require.True(t, hasAuthorName, "expected flat key 'author.name' from join")
	require.True(t, hasFlatAuthorID, "expected flat key 'author.id' from join")

	// Verify denormalization produces nested structure
	nested := denormalize(firstDoc)
	authorSection, ok := nested["author"].(map[string]any)
	require.True(t, ok, "expected nested 'author' object")
	require.Equal(t, "Alice", authorSection["name"])
	require.Equal(t, "a1", authorSection["id"])
}
