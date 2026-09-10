package sqlite

import (
	"context"
	"testing"

	anansisqlite "github.com/asaidimu/go-anansi/v8/sqlite"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testDeps(t *testing.T) abstract.PersistenceDeps {
	return abstract.PersistenceDeps{Logger: zap.NewNop(), DataDir: t.TempDir()}
}

func TestMemory_IsolatedPerName(t *testing.T) {
	p1, cleanup1, err := Memory("test-mem-iso-a")(testDeps(t))
	require.NoError(t, err)
	defer cleanup1()

	p2, cleanup2, err := Memory("test-mem-iso-b")(testDeps(t))
	require.NoError(t, err)
	defer cleanup2()

	ctx := context.Background()
	n1, err := p1.ListCollections(ctx)
	require.NoError(t, err)
	n2, err := p2.ListCollections(ctx)
	require.NoError(t, err)
	require.Equal(t, n1, n2)
	require.NotNil(t, p1)
	require.NotNil(t, p2)
}

func TestDefault_MemoryPath(t *testing.T) {
	for _, path := range []string{"", ":memory:"} {
		p, cleanup, err := Default(path)(testDeps(t))
		require.NoError(t, err, "path %q", path)
		require.NotNil(t, p)
		cleanup()
	}
}

func TestDefault_FilePath(t *testing.T) {
	path := t.TempDir() + "/test.db"
	deps := abstract.PersistenceDeps{Logger: zap.NewNop(), DataDir: t.TempDir()}

	p, cleanup, err := Default(path)(deps)
	require.NoError(t, err)
	require.NotNil(t, p)
	cleanup()

	// Reopening the same file must work (durable state, WAL defaults apply).
	// A fresh DataDir is used: the durable event bus holds a process-wide
	// lock per directory, and factory cleanup only releases the DB handle.
	deps2 := abstract.PersistenceDeps{Logger: zap.NewNop(), DataDir: t.TempDir()}
	p2, cleanup2, err := Default(path)(deps2)
	require.NoError(t, err)
	require.NotNil(t, p2)
	cleanup2()
}

func TestDefault_EmptyPathUsesDepsDBPath(t *testing.T) {
	deps := abstract.PersistenceDeps{
		Logger:  zap.NewNop(),
		DataDir: t.TempDir(),
		DBPath:  t.TempDir() + "/resolved.db",
	}
	p, cleanup, err := Default("")(deps)
	require.NoError(t, err)
	require.NotNil(t, p)
	cleanup()
}

func TestDefault_ConfigOverride(t *testing.T) {
	deps := testDeps(t)
	p, cleanup, err := Default(":memory:", func(c *anansisqlite.Config) {
		c.MaxOpenConns = 2
	})(deps)
	require.NoError(t, err)
	require.NotNil(t, p)
	cleanup()
}
