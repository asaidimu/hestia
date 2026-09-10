package collections

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/stretchr/testify/require"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	usersmodel "github.com/asaidimu/hestia/core/system/users/model"
)

type scratchMsg struct {
	name  string
	ctx   context.Context
	input data.Documenter
}

func (m scratchMsg) ID() string                               { return "" }
func (m scratchMsg) Name() string                             { return m.name }
func (m scratchMsg) Context() context.Context                 { return m.ctx }
func (m scratchMsg) Input() data.Documenter                   { return m.input }
func (m scratchMsg) InputChannel() <-chan abstract.StreamItem { return nil }
func (m scratchMsg) BlobInputChannel() <-chan abstract.Blob   { return nil }
func (m scratchMsg) TenantID() string                         { return "" }
func (m scratchMsg) TraceID() string                          { return "" }
func (m scratchMsg) RequestID() string                        { return "" }
func (m scratchMsg) SessionID() string                        { return "" }
func (m scratchMsg) SourceIP() string                         { return "" }
func (m scratchMsg) UserAgent() string                        { return "" }
func (m scratchMsg) ResourceID() string                       { return "" }

var _ abstract.Message = scratchMsg{}

// Scratch debug: full user:query path with a QDSL body through the real
// registration input schema.
func TestScratchUserQueryFilter(t *testing.T) {
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

	schema := dispatch.SchemaFromTypeWithTag[usersmodel.UserQueryInput]("input", true)
	t.Logf("schema payload field present")
	_ = schema

	doc := testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usersmodel.UserQueryInput]("input", true), `{
		"payload": {
			"filter": {"field": "email", "operator": "eq", "value": "bbb@example.co"},
			"pagination": {"type": "offset", "offset": 0, "limit": 1}
		}
	}`)
	m := doc.ToMap()
	t.Logf("input doc payload: %v", m["payload"])

	// Show exactly what the handler's parse step produces.
	rawPayload, _ := m["payload"].(map[string]any)
	payloadBytes, _ := json.Marshal(rawPayload)
	parsed, err := query.FromBytes(payloadBytes)
	require.NoError(t, err)
	parsedBytes, _ := json.Marshal(parsed)
	t.Logf("parsed query: %s", string(parsedBytes))
	directProbe := query.NewQueryBuilder().From("_user_").Where("email").Eq("bbb@example.co").Build()
	directBytes, _ := json.Marshal(directProbe)
	t.Logf("direct query: %s", string(directBytes))

	handler := NewNamedCollectionQueryHandler("_user_", p)
	res, err := handler(ctx, scratchMsg{name: "system:collections:user:query", ctx: ctx, input: doc})
	require.NoError(t, err)
	require.NotNil(t, res.Page)
	t.Logf("rows: %d", len(res.Page.Documents))
	for _, d := range res.Page.Documents {
		t.Logf("row: %v", d.ToMap()["email"])
	}
	require.Equal(t, 1, len(res.Page.Documents))

	// Control: the identical filter built with Go structs, no JSON involved.
	direct := query.NewQueryBuilder().From("_user_").Where("email").Eq("bbb@example.co").Build()
	dres, err := users.Read(ctx, &direct)
	require.NoError(t, err)
	for _, d := range dres.Data {
		dd, ok := d.(interface{ ToMap() map[string]any })
		require.True(t, ok)
		t.Logf("direct row: %v", dd.ToMap()["email"])
	}
}
