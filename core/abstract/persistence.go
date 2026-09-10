package abstract

import (
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"
)

// PersistenceDeps is the input to a PersistenceFactory.
type PersistenceDeps struct {
	Logger *zap.Logger
	// DataDir is the application's data directory (resolved from
	// SetupConfig.DataDir or the environment). Factories that need
	// durable local state (e.g. the default event bus) anchor it here.
	DataDir string
	// DBPath is the resolved database path (SetupConfig.DBPath or the
	// DataDir-derived default; ":memory:" for ephemeral databases).
	// The default SQLite backend opens this path when constructed
	// with an empty path argument.
	DBPath string
}

// PersistenceFactory builds the storage backend for an application.
//
// hestia core deliberately ships NO default: importing core must not link
// a database driver. Applications opt into a backend explicitly — either
// the default SQLite backend
// (github.com/asaidimu/hestia/core/persistence/sqlite) or a custom
// implementation:
//
//	import hestiasqlite "github.com/asaidimu/hestia/core/persistence/sqlite"
//
//	app, err := hestia.Setup(hestia.SetupConfig{
//		PersistenceFactory: hestiasqlite.Default(":memory:"),
//		...
//	})
//
// The returned cleanup func releases backend resources (DB handles, buses)
// and is invoked on Application.Close. Document-factory configuration is
// owned by core and applied before the factory runs.
type PersistenceFactory func(deps PersistenceDeps) (base.Persistence, func(), error)
