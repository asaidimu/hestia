package model

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/testutil"
)

func newBenchKeyModel(b *testing.B) (*SystemAPIKeys, *GeneratedKey) {
	b.Helper()
	DangerouslyResetSystemAPIKeysModel()
	m, err := InitSystemAPIKeysModel(testutil.NewPersistenceTB(b), zap.NewNop())
	if err != nil {
		b.Fatalf("InitSystemAPIKeysModel: %v", err)
	}
	gen, err := m.Generate()
	if err != nil {
		b.Fatalf("Generate: %v", err)
	}
	return m, gen
}

// BenchmarkAPIKey_Generate measures bcrypt hashing cost for key issuance
// (create/rotate). This is intentionally slow — it is the anti-bruteforce
// control — and the number must be surfaced rather than hidden.
func BenchmarkAPIKey_Generate(b *testing.B) {
	m, _ := newBenchKeyModel(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.Generate(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAPIKey_ValidateKey measures key authentication on the hot path,
// including the best-effort usage counter update.
func BenchmarkAPIKey_ValidateKey(b *testing.B) {
	m, gen := newBenchKeyModel(b)
	created, err := m.CreateKey(context.Background(), gen, "bench-user", &APIKeyCreate{Name: "bench"})
	if err != nil {
		b.Fatalf("CreateKey: %v", err)
	}

	// Give the persisted key a cache warm-up so reads hit memory.
	if _, err := m.Get(context.Background(), created.ID, "bench-user"); err != nil {
		b.Fatalf("Get: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.ValidateKey(context.Background(), gen.FullKey); err != nil {
			b.Fatal(err)
		}
	}
}