package collections_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/stretchr/testify/require"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/collections"
)

func updateInputDoc(t *testing.T, name, docID string, payload map[string]any) data.Documenter {
	t.Helper()
	args := map[string]any{"name": name}
	if docID != "" {
		args["doc_id"] = docID
	}
	return data.MustNewDocument(map[string]any{
		"arguments": args,
		"payload":   payload,
	})
}

// TestDocumentUpdateByIDAndFilter covers both update addressing modes of
// system:collections:document:update: legacy doc_id with a raw partial body,
// and the { set, filter } envelope without doc_id.
func TestDocumentUpdateByIDAndFilter(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)

	_, err := p.CreateCollection(ctx, viewTestSchema(t))
	require.NoError(t, err)

	orders, err := p.Collection(ctx, "ViewOrders")
	require.NoError(t, err)
	for _, d := range []map[string]any{
		{"id": "o1", "status": "paid", "total": 10.0},
		{"id": "o2", "status": "paid", "total": 20.0},
		{"id": "o3", "status": "pending", "total": 30.0},
	} {
		_, err := orders.CreateOne(ctx, data.MustNewDocument(d))
		require.NoError(t, err)
	}

	handler := collections.NewDocumentUpdateHandler(p)

	// Map the custom id field to the system _id_ (by-id updates address _id_).
	empty := query.NewQueryBuilder().Build()
	seeded, err := orders.Read(ctx, &empty)
	require.NoError(t, err)
	systemID := map[string]string{}
	for _, d := range seeded.Data {
		doc, ok := d.(*document.Document)
		require.True(t, ok)
		m := doc.ToMap()
		id, _ := m["id"].(string)
		sysID, _ := m["_id_"].(string)
		systemID[id] = sysID
	}
	require.NotEmpty(t, systemID["o3"])

	// 1. Legacy by-id path with a raw partial body.
	res, err := handler(ctx, testMessage{
		name:  "system:collections:document:update",
		ctx:   ctx,
		input: updateInputDoc(t, "ViewOrders", systemID["o3"], map[string]any{"status": "paid"}),
	})
	require.NoError(t, err)
	require.NotNil(t, res.Document)

	// 2. Filter path via update_many: { set, filter } envelope, no doc_id.
	manyHandler := collections.NewDocumentUpdateManyHandler(p)
	res, err = manyHandler(ctx, testMessage{
		name: "system:collections:document:update_many",
		ctx:  ctx,
		input: updateInputDoc(t, "ViewOrders", "", map[string]any{
			"set": map[string]any{"status": "archived"},
			"filter": map[string]any{
				"condition": map[string]any{"field": "status", "operator": "eq", "value": "paid"},
			},
		}),
	})
	require.NoError(t, err)
	require.NotNil(t, res.Document)

	empty = query.NewQueryBuilder().Build()
	got, err := orders.Read(ctx, &empty)
	require.NoError(t, err)
	statuses := map[string]int{}
	for _, d := range got.Data {
		doc, ok := d.(*document.Document)
		require.True(t, ok)
		m := doc.ToMap()
		status, _ := m["status"].(string)
		statuses[status]++
	}
	// o1+o2 matched the filter (o3 was already flipped to paid by step 1,
	// so all three paid rows archived).
	require.Equal(t, 3, statuses["archived"])
	require.Equal(t, 0, statuses["paid"])

	// 3. Neither doc_id nor envelope → actionable error.
	_, err = handler(ctx, testMessage{
		name:  "system:collections:document:update",
		ctx:   ctx,
		input: updateInputDoc(t, "ViewOrders", "", map[string]any{"status": "x"}),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "UPDATE_TARGET_REQUIRED")
}
