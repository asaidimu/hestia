package testutil

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	pevents "github.com/asaidimu/go-anansi/v8/core/persistence/events"
	"github.com/asaidimu/go-anansi/v8/core/persistence/persistence"
	"github.com/asaidimu/go-anansi/v8/core/query/native"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	sqliteExecutor "github.com/asaidimu/go-anansi/v8/sqlite/executor"
	sqliteQuery "github.com/asaidimu/go-anansi/v8/sqlite/query"
	"github.com/asaidimu/go-anansi/v8/tests/testutils"
	"github.com/asaidimu/go-anansi/v8/utils"
	_ "github.com/mattn/go-sqlite3"
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
	return NewPersistenceTB(t)
}

// NewPersistenceTB is NewPersistence for any testing context (tests and
// benchmarks). It builds a unique in-memory SQLite database named after the
// test, creates the schema-locked collections, and registers cleanup.
func NewPersistenceTB(t testing.TB) base.Persistence {
	t.Helper()
	ctx := context.Background()
	logger := zap.NewNop()

	testutils.ConfigureDocumentFactory()

	// Unique in-memory SQLite database per test (mirrors
	// testutils.SetupTestDB but accepts testing.TB).
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var version string
	require.NoError(t, db.QueryRow("SELECT sqlite_version()").Scan(&version))

	devLogger, _ := zap.NewDevelopment()
	executor, err := sqliteExecutor.NewSQLiteExecutor(db, devLogger)
	require.NoError(t, err)
	queryFactory := sqliteQuery.NewSQLiteFactory(nil)
	interactor, err := native.NewNativeInteractor(executor, queryFactory, devLogger)
	require.NoError(t, err)

	bus, err := utils.NewInMemoryGoEventsBus("test")
	require.NoError(t, err)

	pbus := pevents.NewGoEventsBusAdapter[base.PersistenceEvent](bus)
	p, err := persistence.NewPersistence(interactor, pbus, logger, nil)
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
