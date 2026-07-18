package app_test

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/app/audit"
	"github.com/asaidimu/hestia/internal/app/auth"
	"github.com/asaidimu/hestia/internal/app/operations"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/utility/persistest"
	"go.uber.org/zap"
)

func TestUserModelRegisterAndGet(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "alice@example.com", "p4ssw0rd", "Alice")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := doc.ID()

	byEmail, err := model.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if byEmail.ID() != id {
		t.Errorf("GetByEmail returned doc %s, want %s", byEmail.ID(), id)
	}
	email, _ := byEmail.GetString("email")
	if email != "alice@example.com" {
		t.Errorf("email = %q, want %q", email, "alice@example.com")
	}
	name, _ := byEmail.GetString("name")
	if name != "Alice" {
		t.Errorf("name = %q, want %q", name, "Alice")
	}

	byID, err := model.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.ID() != id {
		t.Errorf("GetByID returned doc %s, want %s", byID.ID(), id)
	}
}

func TestUserModelUpdate(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "bob@example.com", "p4ssw0rd", "Bob")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := doc.ID()

	err = model.Update(ctx, id, map[string]any{"name": "Robert"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := model.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	name, _ := updated.GetString("name")
	if name != "Robert" {
		t.Errorf("name = %q, want %q", name, "Robert")
	}
}

func TestUserModelPassword(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "carol@example.com", "first-pass", "Carol")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := doc.ID()

	hash, err := model.GetPasswordHash(ctx, id)
	if err != nil {
		t.Fatalf("GetPasswordHash: %v", err)
	}
	if hash == "" {
		t.Fatal("GetPasswordHash returned empty string")
	}
	if hash == "first-pass" {
		t.Error("hash should not equal plaintext password")
	}

	err = model.ChangePassword(ctx, id, "new-pass")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	newHash, err := model.GetPasswordHash(ctx, id)
	if err != nil {
		t.Fatalf("GetPasswordHash after change: %v", err)
	}
	if newHash == hash {
		t.Error("password hash should have changed")
	}
}

func TestUserModelSoftDelete(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "dave@example.com", "p4ssw0rd", "Dave")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := doc.ID()

	err = model.SoftDelete(ctx, id)
	if err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	softDeleted, err := model.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID after soft delete: %v", err)
	}
	if !model.IsDeleted(softDeleted) {
		t.Error("expected IsDeleted to be true after SoftDelete")
	}
}

func TestUserModelHardDelete(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "eve@example.com", "p4ssw0rd", "Eve")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := doc.ID()

	err = model.HardDelete(ctx, id)
	if err != nil {
		t.Fatalf("HardDelete: %v", err)
	}

	_, err = model.GetByID(ctx, id)
	if err == nil {
		t.Error("expected GetByID to return error after HardDelete")
	}
}

func TestUserModelList(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc1, _ := model.Register(ctx, "fay@example.com", "p4ss", "Fay")
	doc2, _ := model.Register(ctx, "gia@example.com", "p4ss", "Gia")

	docs, total, err := model.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 2 {
		t.Errorf("total = %d, want >= 2", total)
	}
	ids := map[string]bool{doc1.ID(): true, doc2.ID(): true}
	for _, d := range docs {
		delete(ids, d.ID())
	}
	if len(ids) != 0 {
		t.Error("List did not return both registered users")
	}
}

func TestAPIKeyModelGenerateAndCreate(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "hank@example.com", "p4ss", "Hank")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if gk.FullKey == "" {
		t.Fatal("GeneratedKey.FullKey is empty")
	}
	if gk.Hash == "" {
		t.Fatal("GeneratedKey.Hash is empty")
	}

	req := &apikeys.CreateKeyRequest{Name: "test-key", Operations: []string{"read:*"}}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	name, _ := doc.GetString("name")
	if name != "test-key" {
		t.Errorf("name = %q, want %q", name, "test-key")
	}

	keys, err := keyModel.List(ctx, userDoc.ID())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}

func TestAPIKeyModelDelete(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "iris@example.com", "p4ss", "Iris")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeys.CreateKeyRequest{Name: "delete-me"}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = keyModel.Delete(ctx, doc.ID(), userDoc.ID())
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = keyModel.Get(ctx, doc.ID(), userDoc.ID())
	if err == nil {
		t.Error("expected Get to return error after Delete")
	}
}

func TestSeedModelSetAndGet(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := operations.NewSeedModel(p)

	err := model.Set(ctx, "my_key", "my_value")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := model.Get(ctx, "my_key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "my_value" {
		t.Errorf("got %q, want %q", val, "my_value")
	}

	err = model.Set(ctx, "my_key", "updated_value")
	if err != nil {
		t.Fatalf("Set (update): %v", err)
	}

	val, err = model.Get(ctx, "my_key")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if val != "updated_value" {
		t.Errorf("got %q, want %q", val, "updated_value")
	}

	missing, err := model.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if missing != "" {
		t.Errorf("expected empty string for missing key, got %q", missing)
	}
}

func TestAuditModelInsert(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := audit.NewAuditModel(p)

	entry := core.AuditEntry{
		EventName:    "test.message",
		ActorID:      "user-1",
		ActorType:    core.ActorTypeUser,
		AuthMethod:   core.AuthMethodPassword,
		Operation:    core.OperationExecute,
		ResourceType: "test",
		Status:       core.AuditStatusSuccess,
		LatencyMs:   42,
		ServiceName:  "hestia",
	}

	err := model.Insert(ctx, entry)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
}

func TestSeedAdmin(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	seedModel := operations.NewSeedModel(p)
	logger := zap.NewNop()

	adminID, adminEmail, bootstrapped, err := auth.SeedAdmin(ctx, userModel, seedModel, logger)
	if err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	if adminID == "" {
		t.Fatal("expected non-empty admin ID")
	}
	if adminEmail == "" {
		t.Fatal("expected non-empty admin email")
	}
	if bootstrapped {
		t.Error("expected bootstrapped=false on first run")
	}

	adminID2, adminEmail2, bootstrapped2, err := auth.SeedAdmin(ctx, userModel, seedModel, logger)
	if err != nil {
		t.Fatalf("SeedAdmin second call: %v", err)
	}
	if adminID2 != adminID {
		t.Errorf("expected same admin ID on second call, got %s vs %s", adminID2, adminID)
	}
	if adminEmail2 != adminEmail {
		t.Errorf("expected same admin email on second call, got %s vs %s", adminEmail2, adminEmail)
	}
	if bootstrapped2 {
		t.Error("expected bootstrapped=false on second call (hash unchanged)")
	}
}

func TestSeedAdminWithOptions(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	seedModel := operations.NewSeedModel(p)
	logger := zap.NewNop()

	adminID, adminEmail, _, err := auth.SeedAdmin(ctx, userModel, seedModel, logger, auth.SeedAdminOptions{
		Email:    "custom@admin.com",
		Password: "custom-pass-123",
	})
	if err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	if adminEmail != "custom@admin.com" {
		t.Errorf("adminEmail = %q, want %q", adminEmail, "custom@admin.com")
	}

	doc, err := userModel.GetByID(ctx, adminID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	email, _ := doc.GetString("email")
	if email != "custom@admin.com" {
		t.Errorf("email = %q, want %q", email, "custom@admin.com")
	}
}

func TestAPIKeyModelRotate(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "jake@example.com", "p4ss", "Jake")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeys.CreateKeyRequest{Name: "rotatable"}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rotatedKey, rotatedDoc, err := keyModel.Rotate(ctx, doc.ID(), userDoc.ID())
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if rotatedKey.FullKey == "" {
		t.Fatal("expected non-empty rotated key")
	}
	if rotatedKey.FullKey == gk.FullKey {
		t.Error("rotated key should differ from original")
	}
	if rotatedDoc.ID() != doc.ID() {
		t.Error("rotated doc should have same ID as original")
	}
}

func TestAPIKeyModelValidateKey(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "kay@example.com", "p4ss", "Kay")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeys.CreateKeyRequest{Name: "validatable", Operations: []string{"read:*"}}
	_, err = keyModel.Create(ctx, gk, userDoc.ID(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	claims, err := keyModel.ValidateKey(ctx, gk.FullKey)
	if err != nil {
		t.Fatalf("ValidateKey: %v", err)
	}
	if claims.UserID != userDoc.ID() {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, userDoc.ID())
	}
	if len(claims.Operations) == 0 || claims.Operations[0] != "read:*" {
		t.Errorf("claims.Operations = %v, want [read:*]", claims.Operations)
	}
	if claims.TokenType != "api_key" {
		t.Errorf("claims.TokenType = %q, want %q", claims.TokenType, "api_key")
	}
}

func TestAPIKeyModelUpdate(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "lia@example.com", "p4ss", "Lia")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, _ := keyModel.Generate()
	req := &apikeys.CreateKeyRequest{Name: "updatable", Operations: []string{"read:*"}}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "updated-name"
	updated, err := keyModel.Update(ctx, doc.ID(), userDoc.ID(), &apikeys.UpdateKeyRequest{
		Name:   &newName,
		Operations: []string{"read:*", "write:*"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	name, _ := updated.GetString("name")
	if name != "updated-name" {
		t.Errorf("name = %q, want %q", name, "updated-name")
	}
}
