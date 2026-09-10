package collections_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/stretchr/testify/require"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/collections"
)

// viewOrderRow is the scratch shape behind the ViewOrders collection below.
// The schema is derived from this struct (DTO → schema), so the test never
// hand-writes field IDs.
type viewOrderRow struct {
	ID     string  `anansi:"id,required=true"`
	Status string  `anansi:"status"`
	Total  float64 `anansi:"total"`
}

func viewTestSchema(t *testing.T) *definition.Schema {
	t.Helper()
	raw, err := data.SchemaFrom[viewOrderRow]()
	require.NoError(t, err)
	s, err := definition.FromJSON(raw)
	require.NoError(t, err)
	s.Name = "ViewOrders"
	return s
}

func viewInputDoc(t *testing.T, name string, payload map[string]any) data.Documenter {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"arguments": map[string]any{"name": name},
		"payload":   payload,
	})
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	return data.MustNewDocument(body)
}

func TestViewCreateQueryRefreshDelete(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)

	_, err := p.CreateCollection(ctx, viewTestSchema(t))
	require.NoError(t, err)

	orders, err := p.Collection(ctx, "ViewOrders")
	require.NoError(t, err)
	for _, d := range []map[string]any{
		{"id": "o1", "status": "paid", "total": 250.0},
		{"id": "o2", "status": "pending", "total": 20.0},
		{"id": "o3", "status": "paid", "total": 500.0},
	} {
		_, err := orders.CreateOne(ctx, data.MustNewDocument(d))
		require.NoError(t, err)
	}

	// Virtual view: paid orders.
	stored := query.NewQueryBuilder().
		From("ViewOrders").
		Where("status").Eq("paid").
		Build()
	storedBytes, err := json.Marshal(stored)
	require.NoError(t, err)
	var storedJSON map[string]any
	require.NoError(t, json.Unmarshal(storedBytes, &storedJSON))

	createHandler := collections.NewViewCreateHandler(p)
	res, err := createHandler(ctx, testMessage{
		name:  "system:collections:view:create",
		ctx:   ctx,
		input: viewInputDoc(t, "PaidOrders", map[string]any{"query": storedJSON}),
	})
	require.NoError(t, err)
	require.NotNil(t, res.Document)

	// The view is readable through the regular document:query path and
	// composes the stored filter (2 paid rows, not 3).
	empty := query.NewQueryBuilder().Build()
	view, err := p.Collection(ctx, "PaidOrders")
	require.NoError(t, err)
	got, err := view.Read(ctx, &empty)
	require.NoError(t, err)
	require.Equal(t, 2, got.Count)

	// Views are read-only.
	_, err = view.CreateOne(ctx, data.MustNewDocument(map[string]any{"id": "x"}))
	require.Error(t, err)

	// Refreshing a virtual view fails (only materialized views refresh).
	refreshHandler := collections.NewViewRefreshHandler(p)
	_, err = refreshHandler(ctx, testMessage{
		name:  "system:collections:view:refresh",
		ctx:   ctx,
		input: viewInputDoc(t, "PaidOrders", nil),
	})
	require.Error(t, err)

	// Materialized view: snapshot, stale until refreshed.
	storedAll := query.NewQueryBuilder().From("ViewOrders").Build()
	storedAllBytes, err := json.Marshal(storedAll)
	require.NoError(t, err)
	var storedAllJSON map[string]any
	require.NoError(t, json.Unmarshal(storedAllBytes, &storedAllJSON))

	res, err = createHandler(ctx, testMessage{
		name:  "system:collections:view:create",
		ctx:   ctx,
		input: viewInputDoc(t, "AllOrdersSnap", map[string]any{"query": storedAllJSON, "materialized": true}),
	})
	require.NoError(t, err)
	require.NotNil(t, res.Document)

	mv, err := p.Collection(ctx, "AllOrdersSnap")
	require.NoError(t, err)
	got, err = mv.Read(ctx, &empty)
	require.NoError(t, err)
	require.Equal(t, 3, got.Count)

	_, err = orders.CreateOne(ctx, data.MustNewDocument(map[string]any{"id": "o4", "status": "paid", "total": 999.0}))
	require.NoError(t, err)
	got, err = mv.Read(ctx, &empty)
	require.NoError(t, err)
	require.Equal(t, 3, got.Count, "materialized view must serve the stale snapshot")

	_, err = refreshHandler(ctx, testMessage{
		name:  "system:collections:view:refresh",
		ctx:   ctx,
		input: viewInputDoc(t, "AllOrdersSnap", nil),
	})
	require.NoError(t, err)
	got, err = mv.Read(ctx, &empty)
	require.NoError(t, err)
	require.Equal(t, 4, got.Count, "refresh must rebuild the snapshot")

	// Materialized snapshots are droppable through the collection delete
	// path (their physical table exists). Virtual views have no physical
	// table, so dropping them needs upstream DeleteView support
	// (Persistence.Delete forces DeletePhysicalData=true) — no view:delete
	// message is exposed until then.
	_, err = p.Delete(ctx, "AllOrdersSnap")
	require.NoError(t, err)
	exists, err := p.HasCollection(ctx, "AllOrdersSnap")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestViewCreate_Validation(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	createHandler := collections.NewViewCreateHandler(p)

	// Missing name.
	_, err := createHandler(ctx, testMessage{
		name:  "system:collections:view:create",
		ctx:   ctx,
		input: viewInputDoc(t, "", map[string]any{"query": map[string]any{}}),
	})
	require.Error(t, err)

	// Missing query.
	_, err = createHandler(ctx, testMessage{
		name:  "system:collections:view:create",
		ctx:   ctx,
		input: viewInputDoc(t, "NoQuery", map[string]any{}),
	})
	require.Error(t, err)

	// System-reserved name.
	_, err = createHandler(ctx, testMessage{
		name:  "system:collections:view:create",
		ctx:   ctx,
		input: viewInputDoc(t, "_secret_", map[string]any{"query": map[string]any{}}),
	})
	require.Error(t, err)
}
