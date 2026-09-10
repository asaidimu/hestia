// Package sqlite is hestia's default SQLite persistence backend.
//
// It is intentionally OPT-IN: importing hestia core must never link a
// database driver. Applications that want the embedded SQLite backend import
// this package and hand its factory to hestia.Setup:
//
//	import hestiasqlite "github.com/asaidimu/hestia/core/persistence/sqlite"
//
//	app, err := hestia.Setup(hestia.SetupConfig{
//		PersistenceFactory: hestiasqlite.Default(cfg.DBPath),
//		...
//	})
//
// Applications that do not import this package (custom PersistenceFactory,
// e.g. Postgres) never compile mattn/go-sqlite3 or cgo.
//
// Construction delegates to go-anansi's sqlite helpers
// (sqlite.Config/Handle, NewInteractor/NewMemoryInteractor), inheriting the
// recommended defaults: WAL journal mode, foreign-key enforcement, 5s busy
// timeout, and a bounded pool (1 writer + readers). Every knob remains
// overridable via functional options over sqlite.Config.
//
// Full-text search (IndexTypeFullText) requires building with
// `-tags sqlite_fts5`, otherwise the FTS5 virtual tables fail at runtime
// ("no such module: fts5").
package sqlite

import (
	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	pevents "github.com/asaidimu/go-anansi/v8/core/persistence/events"
	"github.com/asaidimu/go-anansi/v8/core/persistence/persistence"
	anansisqlite "github.com/asaidimu/go-anansi/v8/sqlite"
	events "github.com/asaidimu/go-events/v2"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
)

var (
	ErrSQLiteOpen = common.NewSystemError(
		"ERR_SQLITE_OPEN", "failed to open SQLite database")
	ErrEventBus = common.NewSystemError(
		"ERR_SQLITE_EVENT_BUS", "failed to create persistence event bus")
	ErrPersistenceSetup = common.NewSystemError(
		"ERR_SQLITE_PERSISTENCE_SETUP", "failed to setup persistence")
)

// Default returns a abstract.PersistenceFactory opening path with the
// recommended SQLite settings. An empty path defers to the boot-resolved
// database path (SetupConfig.DBPath or its DataDir-derived default);
// ":memory:" selects a named shared-memory database; any other value opens
// (or creates) a file. Optional overrides tune the underlying sqlite.Config
// before opening.
func Default(path string, overrides ...func(*anansisqlite.Config)) abstract.PersistenceFactory {
	return func(deps abstract.PersistenceDeps) (base.Persistence, func(), error) {
		p := path
		if p == "" {
			p = deps.DBPath
		}
		if p == "" || p == ":memory:" {
			return open(true, "", deps, overrides)
		}
		return open(false, p, deps, overrides)
	}
}

// Memory returns a abstract.PersistenceFactory opening a named shared-memory
// SQLite database. An empty name falls back to the anansi default. Use a
// unique name per test for isolation.
func Memory(name string, overrides ...func(*anansisqlite.Config)) abstract.PersistenceFactory {
	return func(deps abstract.PersistenceDeps) (base.Persistence, func(), error) {
		return open(true, name, deps, overrides)
	}
}

func open(memory bool, path string, deps abstract.PersistenceDeps, overrides []func(*anansisqlite.Config)) (base.Persistence, func(), error) {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	cfg := anansisqlite.Config{Path: path, Logger: logger}
	for _, override := range overrides {
		if override != nil {
			override(&cfg)
		}
	}

	var handle *anansisqlite.Handle
	var err error
	if memory {
		handle, err = anansisqlite.NewMemoryInteractor(cfg)
	} else {
		handle, err = anansisqlite.NewInteractor(cfg)
	}
	if err != nil {
		return nil, nil, ErrSQLiteOpen.
			WithOperation("sqlite.open").
			WithCause(err)
	}

	eventBus, err := events.NewEventBus(events.DefaultConfig(deps.DataDir, "persistence-events"))
	if err != nil {
		handle.Cleanup()
		return nil, nil, ErrEventBus.
			WithOperation("sqlite.open").
			WithCause(err)
	}
	bus := pevents.NewGoEventsBusAdapter[base.PersistenceEvent](eventBus)

	p, err := persistence.NewPersistence(handle.Interactor, bus, logger, nil)
	if err != nil {
		handle.Cleanup()
		return nil, nil, ErrPersistenceSetup.
			WithOperation("sqlite.open").
			WithCause(err)
	}

	logger.Info("Persistence layer initialized (sqlite) — waiting for module schemas.")
	return p, handle.Cleanup, nil
}
