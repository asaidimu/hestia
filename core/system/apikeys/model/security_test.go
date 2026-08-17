package model

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/testutil"
)

func newTestKeyModel(t *testing.T) *SystemAPIKeys {
	t.Helper()
	DangerouslyResetSystemAPIKeysModel()
	m, err := InitSystemAPIKeysModel(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}
	return m
}

func mustCreateKey(t *testing.T, m *SystemAPIKeys, userID string, mutate func(*APIKeyCreate)) (*SystemAPIKeys, *GeneratedKey, *SystemAPIKey) {
	t.Helper()
	gen, err := m.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &APIKeyCreate{Name: "ci-key"}
	if mutate != nil {
		mutate(req)
	}
	created, err := m.CreateKey(context.Background(), gen, userID, req)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	return m, gen, created
}

// TestAPIKey_StoreHoldsHashNotRawKey pins the core invariant: the raw API key
// material is NEVER persisted. Only a bcrypt hash is stored, and the stored
// value is not equal to the raw key.
func TestAPIKey_StoreHoldsHashNotRawKey(t *testing.T) {
	m, gen, created := mustCreateKey(t, newTestKeyModel(t), "owner", nil)

	stored, err := m.Get(context.Background(), created.ID, "owner")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Hash == "" {
		t.Fatal("stored key must have a hash")
	}
	if !strings.HasPrefix(stored.Hash, "$2") {
		t.Fatalf("stored hash must be bcrypt (prefix $2), got %q", stored.Hash[:min(4, len(stored.Hash))])
	}
	if stored.Hash == gen.FullKey {
		t.Fatal("stored hash must never equal the raw key")
	}
	if strings.Contains(stored.Hash, gen.FullKey) {
		t.Fatal("raw key material leaked into the stored hash")
	}

	// The raw key is gone after create — the only way to validate is by
	// presenting the raw key again; it is not recoverable from storage.
	if err := bcrypt.CompareHashAndPassword([]byte(stored.Hash), []byte(gen.FullKey)); err != nil {
		t.Fatalf("hash must verify against the raw key: %v", err)
	}
}

// TestAPIKey_ValidateRequiresFullKey verifies that only the full raw key
// validates — neither the prefix nor a wrong key can authenticate.
func TestAPIKey_ValidateRequiresFullKey(t *testing.T) {
	m, gen, _ := mustCreateKey(t, newTestKeyModel(t), "owner", nil)

	if _, err := m.ValidateKey(context.Background(), gen.FullKey); err != nil {
		t.Fatalf("full key must validate: %v", err)
	}
	if _, err := m.ValidateKey(context.Background(), gen.Prefix); err == nil {
		t.Fatal("prefix alone must not validate, got nil")
	}
	if _, err := m.ValidateKey(context.Background(), gen.FullKey[:len(gen.FullKey)-1]+"X"); err == nil {
		t.Fatal("wrong key must not validate, got nil")
	}
}

// TestAPIKey_OwnerIsolation verifies that Get/Delete/Rotate are scoped to the
// owning user — a different user's key is unreachable ("not found"), and the
// malicious user cannot delete or rotate another user's key.
func TestAPIKey_OwnerIsolation(t *testing.T) {
	m, _, created := mustCreateKey(t, newTestKeyModel(t), "owner", nil)
	ctx := context.Background()

	if _, err := m.Get(ctx, created.ID, "attacker"); err == nil {
		t.Fatal("attacker must not be able to read another user's key, got nil")
	}
	if err := m.Delete(ctx, created.ID, "attacker"); err == nil {
		t.Fatal("attacker must not be able to delete another user's key, got nil")
	}
	if _, _, err := m.Rotate(ctx, created.ID, "attacker"); err == nil {
		t.Fatal("attacker must not be able to rotate another user's key, got nil")
	}

	// Sanity: the owner can still see it after the attacks.
	if _, err := m.Get(ctx, created.ID, "owner"); err != nil {
		t.Fatalf("owner must still access their key: %v", err)
	}
}

// TestAPIKey_RevokedRejected verifies a revoked key no longer authenticates.
func TestAPIKey_RevokedRejected(t *testing.T) {
	m, gen, created := mustCreateKey(t, newTestKeyModel(t), "owner", nil)
	ctx := context.Background()

	revoked := "revoked"
	if _, err := m.UpdateKey(ctx, created.ID, "owner", &APIKeyUpdate{Status: &revoked}); err != nil {
		t.Fatalf("UpdateKey(revoked): %v", err)
	}
	_, err := m.ValidateKey(ctx, gen.FullKey)
	if err == nil {
		t.Fatal("revoked key must not validate, got nil")
	}
	if !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("expected revoked error, got: %v", err)
	}
}

// TestAPIKey_ExpiredRejected verifies an expired key no longer authenticates.
func TestAPIKey_ExpiredRejected(t *testing.T) {
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	m, gen, _ := mustCreateKey(t, newTestKeyModel(t), "owner", func(req *APIKeyCreate) {
		req.Expiry = &past
	})

	_, err := m.ValidateKey(context.Background(), gen.FullKey)
	if err == nil {
		t.Fatal("expired key must not validate, got nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got: %v", err)
	}
}

// TestAPIKey_RotateInvalidatesOldKey verifies rotation replaces the stored
// hash so the previous raw key stops working immediately.
func TestAPIKey_RotateInvalidatesOldKey(t *testing.T) {
	m, gen, created := mustCreateKey(t, newTestKeyModel(t), "owner", nil)
	ctx := context.Background()

	newKey, updated, err := m.Rotate(ctx, created.ID, "owner")
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if updated.Hash == gen.Hash {
		t.Fatal("rotation must change the stored hash")
	}
	if newKey.FullKey == gen.FullKey {
		t.Fatal("rotation must issue new raw key material")
	}

	if _, err := m.ValidateKey(ctx, gen.FullKey); err == nil {
		t.Fatal("old key must be invalidated after rotation, got nil")
	}
	if _, err := m.ValidateKey(ctx, newKey.FullKey); err != nil {
		t.Fatalf("new key must validate after rotation: %v", err)
	}
}

// TestAPIKey_PublicProjectionExcludesHash verifies the public DTO never
// exposes the hash (and therefore can never leak key material to list/get).
func TestAPIKey_PublicProjectionExcludesHash(t *testing.T) {
	typ := reflect.TypeOf(APIKeyPublic{})
	if f, ok := typ.FieldByName("Hash"); ok {
		t.Fatalf("APIKeyPublic must not expose a Hash field, got %v", f.Name)
	}

	// The output schema (serialized JSON contract) must not contain hash either.
	schema := APIKeyOutputSchema()
	for _, f := range schema.Fields {
		if strings.EqualFold(string(f.Name), "hash") {
			t.Fatalf("output schema must not expose a hash field, got %q", f.Name)
		}
	}
	if _, ok := schema.Fields["key"]; ok {
		t.Fatal("APIKeyOutput (single get/list projection) must not expose the raw key")
	}
}

// TestAPIKey_CreatedOutputRevealsKeyOnceOnly documents the one-shot contract:
// the raw key is present ONLY in APIKeyCreatedOutput (create/rotate), which is
// the single time a client can read it.
func TestAPIKey_CreatedOutputRevealsKeyOnceOnly(t *testing.T) {
	out := &APIKeyCreatedOutput{}
	if _, ok := reflect.TypeOf(out).Elem().FieldByName("Key"); !ok {
		t.Fatal("APIKeyCreatedOutput must carry the one-shot raw key")
	}
}
