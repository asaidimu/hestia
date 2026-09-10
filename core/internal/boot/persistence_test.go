package boot

import (
	"strings"
	"testing"

	"github.com/asaidimu/hestia/core/runtime"
	"go.uber.org/zap"
)

// Core ships no database driver: booting without an explicit
// PersistenceFactory must fail fast with an actionable error instead of
// silently constructing an implicit backend.
func TestNewPersistenceManager_RequiresFactory(t *testing.T) {
	cfg := runtime.DefaultConfig()
	cfg.PersistenceFactory = nil

	_, err := NewPersistenceManager(cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected missing-factory error, got nil")
	}
	if !strings.Contains(err.Error(), "ERR_PERSISTENCE_FACTORY_MISSING") {
		t.Fatalf("expected ERR_PERSISTENCE_FACTORY_MISSING, got: %v", err)
	}
}
