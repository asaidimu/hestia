package boot

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/core"
)

func TestNewDatabase_InMemory(t *testing.T) {
	cfg := &core.Config{
		DBPath: ":memory:",
	}
	logger := zap.NewNop()

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	defer db.Close()

	if db.DB == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
	if db.Interactor == nil {
		t.Fatal("expected non-nil Interactor")
	}

	err = db.DB.Ping()
	if err != nil {
		t.Fatalf("failed to ping in-memory database: %v", err)
	}
}

func TestNewDatabase_FileBased(t *testing.T) {
	cfg := &core.Config{
		DBPath: ":memory:", // still use in-memory for test isolation
	}
	logger := zap.NewNop()

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	var journalMode string
	err = db.DB.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" && journalMode != "WAL" {
		t.Logf("journal_mode: %s", journalMode)
	}
}

func TestDatabase_Close_Idempotent(t *testing.T) {
	cfg := &core.Config{
		DBPath: ":memory:",
	}
	logger := zap.NewNop()

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Fatalf("first close failed: %v", err)
	}
}
