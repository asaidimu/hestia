package testutil

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paginationOptions(offset, limit int, includeTotal bool) *query.PaginationOptions {
	off := offset
	return &query.PaginationOptions{
		Type:         query.PaginationTypeOffset,
		Offset:       &off,
		Limit:        limit,
		IncludeTotal: &includeTotal,
	}
}

func TestPaginationReturnsTotal(t *testing.T) {
	ctx := context.Background()
	p := NewPersistence(t)

	schema, err := definition.FromJSON([]byte(`{
		"name": "_test_pagination_",
		"version": "1.0.0",
		"fields": {
			"value": { "name": "value", "type": "string" }
		}
	}`))
	require.NoError(t, err)

	col, err := p.CreateCollection(ctx, schema)
	require.NoError(t, err)

	for range 5 {
		doc := data.MustNewDocument(map[string]any{"value": "doc"})
		_, err := col.CreateOne(ctx, doc)
		require.NoError(t, err)
	}

	// Query with pagination and IncludeTotal=true
	q := query.Query{
		Pagination: paginationOptions(0, 2, true),
	}

	result, err := col.Read(ctx, &q)
	require.NoError(t, err)
	require.NotNil(t, result.PaginationInfo)

	t.Logf("PaginationInfo: number=%d size=%d count=%d total=%d pages=%d",
		result.PaginationInfo.Number,
		result.PaginationInfo.Size,
		result.PaginationInfo.Count,
		result.PaginationInfo.Total,
		result.PaginationInfo.Pages)

	assert.Equal(t, 5, result.PaginationInfo.Total)
	assert.Equal(t, 3, result.PaginationInfo.Pages)
	assert.Equal(t, 1, result.PaginationInfo.Number)
	assert.Equal(t, 2, result.PaginationInfo.Size)
	assert.Equal(t, 2, result.PaginationInfo.Count)
}

func TestPaginationIncludeTotalDefault(t *testing.T) {
	ctx := context.Background()
	p := NewPersistence(t)

	schema, err := definition.FromJSON([]byte(`{
		"name": "_test_pagination_default_",
		"version": "1.0.0",
		"fields": {
			"value": { "name": "value", "type": "string" }
		}
	}`))
	require.NoError(t, err)

	col, err := p.CreateCollection(ctx, schema)
	require.NoError(t, err)

	for range 3 {
		doc := data.MustNewDocument(map[string]any{"value": "doc"})
		_, err := col.CreateOne(ctx, doc)
		require.NoError(t, err)
	}

	off := 0
	q := query.Query{
		Pagination: &query.PaginationOptions{
			Type:   query.PaginationTypeOffset,
			Offset: &off,
			Limit:  10,
		},
	}

	result, err := col.Read(ctx, &q)
	require.NoError(t, err)
	require.NotNil(t, result.PaginationInfo)

	t.Logf("PaginationInfo (no IncludeTotal): number=%d size=%d count=%d total=%d pages=%d",
		result.PaginationInfo.Number,
		result.PaginationInfo.Size,
		result.PaginationInfo.Count,
		result.PaginationInfo.Total,
		result.PaginationInfo.Pages)

	assert.Equal(t, 3, result.PaginationInfo.Total,
		"Total should be the actual count even without IncludeTotal")
	assert.Equal(t, 1, result.PaginationInfo.Pages,
		"Pages should reflect the actual total")
}

func TestPaginationWithIncludeTotalFix(t *testing.T) {
	ctx := context.Background()
	p := NewPersistence(t)

	schema, err := definition.FromJSON([]byte(`{
		"name": "_test_pagination_fix_",
		"version": "1.0.0",
		"fields": {
			"value": { "name": "value", "type": "string" }
		}
	}`))
	require.NoError(t, err)

	col, err := p.CreateCollection(ctx, schema)
	require.NoError(t, err)

	for range 3 {
		doc := data.MustNewDocument(map[string]any{"value": "doc"})
		_, err := col.CreateOne(ctx, doc)
		require.NoError(t, err)
	}

	// Simulate what the server handler does when client sends pagination WITHOUT IncludeTotal
	clientQuery := `{"pagination":{"type":"offset","offset":0,"limit":2}}`

	parsed, err := query.FromBytes([]byte(clientQuery))
	require.NoError(t, err)

	// Apply the fix: ensure IncludeTotal=true
	if parsed.Pagination.IncludeTotal == nil || !*parsed.Pagination.IncludeTotal {
		includeTotal := true
		parsed.Pagination.IncludeTotal = &includeTotal
	}

	result, err := col.Read(ctx, parsed)
	require.NoError(t, err)
	require.NotNil(t, result.PaginationInfo)

	t.Logf("After fix: number=%d size=%d count=%d total=%d pages=%d",
		result.PaginationInfo.Number,
		result.PaginationInfo.Size,
		result.PaginationInfo.Count,
		result.PaginationInfo.Total,
		result.PaginationInfo.Pages)

	assert.Equal(t, 3, result.PaginationInfo.Total,
		"With IncludeTotal forced on, Total should be 3")
	assert.Equal(t, 2, result.PaginationInfo.Pages,
		"Pages should be ceil(3/2) = 2")
}
