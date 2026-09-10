//go:build sqlite_fts5

package sqlite

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/tests/testutils"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// NOTE: this file requires -tags sqlite_fts5 (mattn/go-sqlite3 only
// compiles the FTS5 module with that tag). Without it the fulltext index
// creation fails with "no such module: fts5".

func ftsArticlesSchema() *definition.Schema {
	return &definition.Schema{
		BaseSchema: definition.BaseSchema{
			Name: "FTSArticles",
			Fields: map[definition.FieldId]definition.Field{
				"019fb22d-a9a1-7e60-8924-8f09eb81a0b1": {Name: "id", Required: true, FieldProperties: definition.FieldProperties{Type: definition.FieldTypeString}},
				"019fb22d-a9a1-7e61-8924-8f09eb81a0b2": {Name: "title", Required: true, FieldProperties: definition.FieldProperties{Type: definition.FieldTypeString}},
				"019fb22d-a9a1-7e62-8924-8f09eb81a0b3": {Name: "body", FieldProperties: definition.FieldProperties{Type: definition.FieldTypeString}},
			},
			Indexes: map[definition.IndexID]definition.Index{
				"019fb22d-a9a1-7e63-8924-8f09eb81a0c1": {
					Name:   "fts_articles_fts",
					Type:   definition.IndexTypeFullText,
					Fields: []definition.FieldName{"title", "body"},
				},
			},
		},
		Version: common.MustNewVersion("1.0.0"),
	}
}

func seedFTSArticles(t *testing.T, ctx context.Context, coll base.Collection) {
	t.Helper()
	docs := []map[string]any{
		{"id": "a1", "title": "Introduction to Go", "body": "Go is a statically typed, compiled programming language."},
		{"id": "a2", "title": "Database design fundamentals", "body": "Relational databases organize data into tables."},
		{"id": "a3", "title": "Full-text search with SQLite FTS5", "body": "SQLite FTS5 provides fast tokenized search via MATCH queries."},
	}
	for _, d := range docs {
		_, err := coll.CreateOne(ctx, data.MustNewDocument(d))
		require.NoError(t, err)
	}
}

func TestTextSearch_ContainsExactPhrase(t *testing.T) {
	ctx := context.Background()
	// The factory leaves document-factory configuration to core; direct
	// users configure it themselves (as core/internal/boot does).
	testutils.ConfigureDocumentFactory()

	p, cleanup, err := Memory("test-fts-basic")(abstract.PersistenceDeps{
		Logger:  zap.NewNop(),
		DataDir: t.TempDir(),
	})
	require.NoError(t, err)
	defer cleanup()

	_, err = p.CreateCollection(ctx, ftsArticlesSchema())
	require.NoError(t, err)
	coll, err := p.Collection(ctx, "FTSArticles")
	require.NoError(t, err)
	seedFTSArticles(t, ctx, coll)

	// Contains: prefix match on "SQLite" hits a3's title.
	q := query.NewQueryBuilder().From("FTSArticles").TextSearch("title").Contains("SQLite").Build()
	res, err := coll.Read(ctx, &q)
	require.NoError(t, err)
	require.Equal(t, 1, res.Count)

	// Field scoping: "tokenized" lives only in a3's body.
	inTitle := query.NewQueryBuilder().From("FTSArticles").TextSearch("title").Contains("tokenized").Build()
	res, err = coll.Read(ctx, &inTitle)
	require.NoError(t, err)
	require.Equal(t, 0, res.Count)

	inBody := query.NewQueryBuilder().From("FTSArticles").TextSearch("body").Contains("tokenized").Build()
	res, err = coll.Read(ctx, &inBody)
	require.NoError(t, err)
	require.Equal(t, 1, res.Count)

	// Phrase: contiguous match only.
	phrase := query.NewQueryBuilder().From("FTSArticles").TextSearch("title").Phrase("Full-text search").Build()
	res, err = coll.Read(ctx, &phrase)
	require.NoError(t, err)
	require.Equal(t, 1, res.Count)

	// Exact: single-token match.
	exact := query.NewQueryBuilder().From("FTSArticles").TextSearch("title").Exact("Go").Build()
	res, err = coll.Read(ctx, &exact)
	require.NoError(t, err)
	require.Equal(t, 1, res.Count)

	// FTS composes with regular filters (AND).
	both := query.NewQueryBuilder().From("FTSArticles").
		TextSearch("title").Contains("SQLite").
		Where("id").Eq("a3").Build()
	res, err = coll.Read(ctx, &both)
	require.NoError(t, err)
	require.Equal(t, 1, res.Count)

	mismatch := query.NewQueryBuilder().From("FTSArticles").
		TextSearch("title").Contains("SQLite").
		Where("id").Eq("a1").Build()
	res, err = coll.Read(ctx, &mismatch)
	require.NoError(t, err)
	require.Equal(t, 0, res.Count)
}
