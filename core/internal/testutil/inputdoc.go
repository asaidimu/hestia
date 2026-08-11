package testutil

import (
	"sync"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
)

// inputPoolCache caches a schema-bound DocumentPool per schema name so tests
// pay the enrich/compile/link cost once per schema.
var inputPoolCache sync.Map // schema name -> *document.DocumentPool

// InputDoc reads a JSON byte slice into a schema-bound, pooled document built
// against the given input schema — the same schema the dispatcher validates
// handler inputs against. No map[string]any intermediate is involved.
func InputDoc(t *testing.T, s *definition.Schema, j string) *document.Document {
	t.Helper()
	pool := InputDocPool(t, s)
	return pool.MustFromJSON([]byte(j))
}

// InputDocPool returns the cached schema-bound DocumentPool for s, building
// and caching it on first use.
func InputDocPool(t *testing.T, s *definition.Schema) *document.DocumentPool {
	t.Helper()
	if s != nil && s.Name != "" {
		if v, ok := inputPoolCache.Load(s.Name); ok {
			return v.(*document.DocumentPool)
		}
		pool, err := document.NewDocumentPool(s)
		if err != nil {
			t.Fatalf("NewDocumentPool(%s): %v", s.Name, err)
		}
		actual, _ := inputPoolCache.LoadOrStore(s.Name, pool)
		return actual.(*document.DocumentPool)
	}
	pool, err := document.NewDocumentPool(s)
	if err != nil {
		t.Fatalf("NewDocumentPool: %v", err)
	}
	return pool
}