package feature_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	apikeysmodel "github.com/asaidimu/hestia/core/feature/apikeys/model"
	"github.com/asaidimu/hestia/core/feature/audit"
	"github.com/asaidimu/hestia/core/feature/auth"
	"github.com/asaidimu/hestia/core/feature/operations"
	usermodel "github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
	"go.uber.org/zap"
)

func newUserModel(t *testing.T) *usermodel.SystemUsers {
	t.Helper()
	usermodel.DangerouslyResetSystemUsersModel()
	m, err := usermodel.InitSystemUsersModel(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemUsersModel: %v", err)
	}
	return m
}

func newUserModelOn(t *testing.T, p base.Persistence) *usermodel.SystemUsers {
	t.Helper()
	usermodel.DangerouslyResetSystemUsersModel()
	m, err := usermodel.InitSystemUsersModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemUsersModel: %v", err)
	}
	return m
}

func newUserAndKeyModels(t *testing.T) (*usermodel.SystemUsers, *apikeysmodel.SystemAPIKeys) {
	t.Helper()
	usermodel.DangerouslyResetSystemUsersModel()
	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	p := testutil.NewPersistence(t)
	userModel, err := usermodel.InitSystemUsersModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemUsersModel: %v", err)
	}
	keyModel, err := apikeysmodel.InitSystemAPIKeysModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}
	return userModel, keyModel
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

	req := &apikeysmodel.APIKeyCreate{Name: "test-key", Operations: []string{"read:*"}}
	doc, err := keyModel.CreateKey(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if doc.Name != "test-key" {
		t.Errorf("name = %q, want %q", doc.Name, "test-key")
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
	req := &apikeysmodel.APIKeyCreate{Name: "delete-me"}
	doc, err := keyModel.CreateKey(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	err = keyModel.Delete(ctx, doc.ID, userDoc.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = keyModel.Get(ctx, doc.ID, userDoc.ID)
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
	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	keyModel, err := apikeysmodel.InitSystemAPIKeysModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}

	userDoc, err := userModel.Register(ctx, "jake@example.com", "p4ss", "Jake", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeysmodel.APIKeyCreate{Name: "rotatable"}
	doc, err := keyModel.CreateKey(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	rotatedKey, rotatedDoc, err := keyModel.Rotate(ctx, doc.ID, userDoc.ID)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if rotatedKey.FullKey == "" {
		t.Fatal("expected non-empty rotated key")
	}
	if rotatedKey.FullKey == gk.FullKey {
		t.Error("rotated key should differ from original")
	}
	if rotatedDoc.ID != doc.ID {
		t.Error("rotated doc should have same ID as original")
	}
}

func TestAPIKeyModelValidateKey(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	userModel := newUserModelOn(t, p)
	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	keyModel, err := apikeysmodel.InitSystemAPIKeysModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}

	userDoc, err := userModel.Register(ctx, "kay@example.com", "p4ss", "Kay", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, err := keyModel.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req := &apikeysmodel.APIKeyCreate{Name: "validatable", Operations: []string{"read:*"}}
	_, err = keyModel.CreateKey(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
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
	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	keyModel, err := apikeysmodel.InitSystemAPIKeysModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}

	userDoc, err := userModel.Register(ctx, "lia@example.com", "p4ss", "Lia", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	gk, _ := keyModel.Generate()
	req := &apikeysmodel.APIKeyCreate{Name: "updatable", Operations: []string{"read:*"}}
	doc, err := keyModel.CreateKey(ctx, gk, userDoc.ID, req)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	newName := "updated-name"
	updated, err := keyModel.UpdateKey(ctx, doc.ID, userDoc.ID, &apikeysmodel.APIKeyUpdate{
		Name:       &newName,
		Operations: []string{"read:*", "write:*"},
	})
	if err != nil {
		t.Fatalf("UpdateKey: %v", err)
	}
	if updated.Name != "updated-name" {
		t.Errorf("name = %q, want %q", updated.Name, "updated-name")
	}
}
