package collections_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/stretchr/testify/require"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/collections"
)

type filterMsg struct {
	name  string
	ctx   context.Context
	input data.Documenter
}

func (m filterMsg) ID() string                               { return "" }
func (m filterMsg) Name() string                             { return m.name }
func (m filterMsg) Context() context.Context                 { return m.ctx }
func (m filterMsg) Input() data.Documenter                   { return m.input }
func (m filterMsg) InputChannel() <-chan abstract.StreamItem { return nil }
func (m filterMsg) BlobInputChannel() <-chan abstract.Blob   { return nil }
func (m filterMsg) TenantID() string                         { return "" }
func (m filterMsg) TraceID() string                          { return "" }
func (m filterMsg) RequestID() string                        { return "" }
func (m filterMsg) SessionID() string                        { return "" }
func (m filterMsg) SourceIP() string                         { return "" }
func (m filterMsg) UserAgent() string                        { return "" }
func (m filterMsg) ResourceID() string                       { return "" }

var _ abstract.Message = filterMsg{}

// TestNamedCollectionQueryAppliesFilter guards the user:query regression
// where a singular "filter" payload key was silently dropped by the query
// parser (canonical key is "filters"), returning every row unfiltered.
// Both spellings must return exactly the matching document.
func TestNamedCollectionQueryAppliesFilter(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)

	users, err := p.Collection(ctx, "_user_")
	require.NoError(t, err)
	for _, d := range []map[string]any{
		{"email": "aaa@example.co", "password": "Passw0rd1", "name": "A"},
		{"email": "bbb@example.co", "password": "Passw0rd1", "name": "B"},
	} {
		_, err := users.CreateOne(ctx, data.MustNewDocument(d))
		require.NoError(t, err)
	}

	handler := collections.NewNamedCollectionQueryHandler("_user_", p)

	for _, payload := range []map[string]any{
		{"filter": map[string]any{
			"condition": map[string]any{"field": "email", "operator": "eq", "value": "bbb@example.co"},
		}},
		{"filters": map[string]any{
			"condition": map[string]any{"field": "email", "operator": "eq", "value": "bbb@example.co"},
		}},
	} {
		res, err := handler(ctx, filterMsg{
			name:  "system:collections:user:query",
			ctx:   ctx,
			input: data.MustNewDocument(map[string]any{"payload": payload}),
		})
		require.NoError(t, err)
		require.NotNil(t, res.Page)
		require.Equal(t, 1, len(res.Page.Documents), "payload %v", payload)
		require.Equal(t, "bbb@example.co", res.Page.Documents[0].ToMap()["email"])
	}
}
