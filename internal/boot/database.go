package boot

import (
	"database/sql"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/query/native"
	sqliteExecutor "github.com/asaidimu/go-anansi/v8/sqlite/executor"
	sqliteQuery "github.com/asaidimu/go-anansi/v8/sqlite/query"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/core"
)

type Database struct {
	DB         *sql.DB
	Interactor query.DatabaseInteractor
}

func NewDatabase(cfg *core.Config, logger *zap.Logger) (*Database, error) {
	var dsn string
	switch cfg.DBPath {
	case ":memory:":
		dsn = ":memory:?cache=shared"
	default:
		dsn = fmt.Sprintf("file:%s?cache=shared&_fk=1", cfg.DBPath)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	executor, err := sqliteExecutor.NewSQLiteExecutor(db, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create SQLite executor: %w", err)
	}

	queryFactory := sqliteQuery.NewSQLiteFactory(logger)
	interactor, err := native.NewNativeInteractor(executor, queryFactory, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create native interactor: %w", err)
	}

	return &Database{
		DB:         db,
		Interactor: interactor,
	}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}
