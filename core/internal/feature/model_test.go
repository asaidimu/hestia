package feature_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/audit"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/operations"
	"github.com/asaidimu/hestia/core/internal/feature/users/schema"
	"github.com/asaidimu/hestia/core/internal/testutil"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
	"go.uber.org/zap"
)

func newUserModel(t *testing.T) *schema.SystemUsers {
	t.Helper()
	model, err := schema.NewSystemUsers(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("NewSystemUsers: %v", err)
	}
	return model
}

func newUserModelOn(t *testing.T, p base.Persistence) *schema.SystemUsers {
	t.Helper()
	model, err := schema.NewSystemUsers(p, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSystemUsers: %v", err)
	}
	return model
}

func newUserAndKeyModels(t *testing.T) (*schema.SystemUsers, *apikeys.APIKeyModel) {
	t.Helper()
	p := testutil.NewPersistence(t)
	userModel, err := schema.NewSystemUsers(p, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSystemUsers: %v", err)
	}
	return userModel, apikeys.NewAPIKeyModel(p)
}

func TestUserModelRegisterAndGet(t *testing.T) {
	ctx := context.Background()
	model := newUserModel(t)

	user, err := model.Register(ctx, "alice@example.com", "p4ssw0rd", "Alice", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	byEmail, err := model.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if byEmail.ID != id {
		t.Errorf("GetByEmail returned doc %s, want %s", byEmail.ID, id)
	}
	if byEmail.Email != "alice@example.com" {
		t.Errorf("email = %q, want %q", byEmail.Email, "alice@example.com")
	}
	if byEmail.Name != "Alice" {
		t.Errorf("name = %q, want %q", byEmail.Name, "Alice")
	}

	byID, err := model.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.ID != id {
		t.Errorf("GetByID returned doc %s, want %s", byID.ID, id)
	}
}

func TestUserModelUpdate(t *testing.T) {
	ctx := context.Background()
	model := newUserModel(t)

	user, err := model.Register(ctx, "bob@example.com", "p4ssw0rd", "Bob", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	update := data.New(&schema.SystemUser{Name: "Robert"})
	if _, err := model.Update(ctx, id, update); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := model.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Name != "Robert" {
		t.Errorf("name = %q, want %q", updated.Name, "Robert")
	}
}

func TestUserModelPassword(t *testing.T) {
	ctx := context.Background()
	model := newUserModel(t)

	user, err := model.Register(ctx, "carol@example.com", "first-pass", "Carol", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

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

func TestUserModelDisable(t *testing.T) {
	ctx := context.Background()
	model := newUserModel(t)

	user, err := model.Register(ctx, "dave@example.com", "p4ssw0rd", "Dave", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	disabled := int64(1)
	update := data.New(&schema.SystemUser{Disabled: &disabled})
	if _, err := model.Update(ctx, id, update); err != nil {
		t.Fatalf("Update disabled: %v", err)
	}

	// GetByEmail should reject disabled user
	_, err = model.GetByEmail(ctx, "dave@example.com")
	if err == nil {
		t.Error("expected GetByEmail to return error for disabled user")
	}

	// GetActiveByID should reject disabled user
	_, err = model.GetActiveByID(ctx, id)
	if err == nil {
		t.Error("expected GetActiveByID to return error for disabled user")
	}

	// GetByID should still return the doc regardless
	_, err = model.GetByID(ctx, id)
	if err != nil {
		t.Errorf("GetByID should still return disabled user: %v", err)
	}
}

func TestUserModelDelete(t *testing.T) {
	ctx := context.Background()
	model := newUserModel(t)

	user, err := model.Register(ctx, "eve@example.com", "p4ssw0rd", "Eve", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	err = model.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = model.GetByID(ctx, id)
	if err == nil {
		t.Error("expected GetByID to return error after Delete")
	}
}

func TestUserModelList(t *testing.T) {
	ctx := context.Background()
	model := newUserModel(t)

	user1, _ := model.Register(ctx, "fay@example.com", "p4ss", "Fay", "public", nil)
	user2, _ := model.Register(ctx, "gia@example.com", "p4ss", "Gia", "public", nil)

	q := query.NewQueryBuilder().Limit(10).Build()
	docs, err := model.Read(ctx, &q)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(docs) < 2 {
		t.Errorf("got %d docs, want >= 2", len(docs))
	}
	ids := map[string]bool{user1.ID: true, user2.ID: true}
	for _, d := range docs {
		delete(ids, d.ID)
	}
	if len(ids) != 0 {
		t.Error("Read did not return both registered users")
	}
}

func TestAPIKeyModelGenerateAndCreate(t *testing.T) {
	ctx := context.Background()
	userModel, keyModel := newUserAndKeyModels(t)

	userDoc, err := userModel.Register(ctx, "hank@example.com", "p4ss", "Hank", "public", nil)
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
	doc, err := keyModel.Create(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	name, _ := doc.GetString("name")
	if name != "test-key" {
		t.Errorf("name = %q, want %q", name, "test-key")
	}

	keys, err := keyModel.List(ctx, userDoc.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}

func TestAPIKeyModelDelete(t *testing.T) {
	ctx := context.Background()
	userModel, keyModel := newUserAndKeyModels(t)

	userDoc, err := userModel.Register(ctx, "iris@example.com", "p4ss", "Iris", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeys.CreateKeyRequest{Name: "delete-me"}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = keyModel.Delete(ctx, doc.ID(), userDoc.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = keyModel.Get(ctx, doc.ID(), userDoc.ID)
	if err == nil {
		t.Error("expected Get to return error after Delete")
	}
}

func TestSeedModelSetAndGet(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
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
	p := testutil.NewPersistence(t)
	model := audit.NewAuditModel(p)

	entry := auditdomain.AuditEntry{
		EventName:    "test.message",
		ActorID:      "user-1",
		ActorType:    auditdomain.ActorTypeUser,
		AuthMethod:   auditdomain.AuthMethodPassword,
		Operation:    auditdomain.OperationExecute,
		ResourceType: "test",
		Status:       auditdomain.AuditStatusSuccess,
		LatencyMs:    42,
		ServiceName:  "hestia",
	}

	err := model.Insert(ctx, entry)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
}

func TestSeedAdmin(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	userModel := newUserModelOn(t, p)
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
	p := testutil.NewPersistence(t)
	userModel := newUserModelOn(t, p)
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
	if doc.Email != "custom@admin.com" {
		t.Errorf("email = %q, want %q", doc.Email, "custom@admin.com")
	}
}

func TestAPIKeyModelRotate(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	userModel := newUserModelOn(t, p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "jake@example.com", "p4ss", "Jake", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeys.CreateKeyRequest{Name: "rotatable"}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rotatedKey, rotatedDoc, err := keyModel.Rotate(ctx, doc.ID(), userDoc.ID)
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
	p := testutil.NewPersistence(t)
	userModel := newUserModelOn(t, p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "kay@example.com", "p4ss", "Kay", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeys.CreateKeyRequest{Name: "validatable", Operations: []string{"read:*"}}
	_, err = keyModel.Create(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	claims, err := keyModel.ValidateKey(ctx, gk.FullKey)
	if err != nil {
		t.Fatalf("ValidateKey: %v", err)
	}
	if claims.UserID != userDoc.ID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, userDoc.ID)
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
	p := testutil.NewPersistence(t)
	userModel := newUserModelOn(t, p)
	keyModel := apikeys.NewAPIKeyModel(p)

	userDoc, err := userModel.Register(ctx, "lia@example.com", "p4ss", "Lia", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, _ := keyModel.Generate()
	req := &apikeys.CreateKeyRequest{Name: "updatable", Operations: []string{"read:*"}}
	doc, err := keyModel.Create(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "updated-name"
	updated, err := keyModel.Update(ctx, doc.ID(), userDoc.ID, &apikeys.UpdateKeyRequest{
		Name:       &newName,
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
