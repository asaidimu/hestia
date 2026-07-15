package persistest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	pevents "github.com/asaidimu/go-anansi/v8/core/persistence/events"
	"github.com/asaidimu/go-anansi/v8/core/persistence/persistence"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/tests/testutils"
	"github.com/asaidimu/go-events"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	t.Helper()
	ctx := context.Background()
	logger := zap.NewNop()

	interactor, cleanup := testutils.CreateNativeInteractor(t)
	t.Cleanup(cleanup)

	bus, err := events.NewTypedEventBus[base.PersistenceEvent](events.DefaultConfig())
	require.NoError(t, err)

	p, err := persistence.NewPersistence(interactor, pevents.NewGoEventsBusAdapter(bus), logger, nil)
	require.NoError(t, err)

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
