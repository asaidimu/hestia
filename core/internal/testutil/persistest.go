package testutil

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/tests/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	hestiasqlite "github.com/asaidimu/hestia/core/persistence/sqlite"
)

func init() {
	os.Setenv("ANANSI_ENV", "development")
}

func projectRoot() string {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return ""
		}
		wd = parent
	}
}

func NewPersistence(t *testing.T) base.Persistence {
	return NewPersistenceTB(t)
}

// NewPersistenceTB is NewPersistence for any testing context (tests and
// benchmarks). It builds a unique in-memory SQLite database named after the
// test via the default sqlite backend, creates the schema-locked
// collections, and registers cleanup.
func NewPersistenceTB(t testing.TB) base.Persistence {
	t.Helper()
	ctx := context.Background()

	testutils.ConfigureDocumentFactory()

	p, cleanup, err := hestiasqlite.Memory(t.Name())(abstract.PersistenceDeps{
		Logger:  zap.NewNop(),
		DataDir: t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	root := projectRoot()
	if root == "" {
		t.Fatal("could not find project root")
	}

	lockData, err := os.ReadFile(filepath.Join(root, "schemas.lock.json"))
	require.NoError(t, err)

	var lockfile struct {
		Schemas map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"schemas"`
	}
	require.NoError(t, json.Unmarshal(lockData, &lockfile))

	for name, entry := range lockfile.Schemas {
		schemaData, err := json.Marshal(entry.Schema)
		require.NoError(t, err)
		schema, err := definition.FromJSON(schemaData)
		require.NoError(t, err)
		_, err = p.CreateCollection(ctx, schema)
		require.NoError(t, err)
		_ = name
	}

	return p
}
