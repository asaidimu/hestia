package apikeys_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/apikeys"
	apikeysmodel "github.com/asaidimu/hestia/core/feature/apikeys/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func testModel(t *testing.T) *apikeysmodel.SystemAPIKeys {
	t.Helper()
	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	model, err := apikeysmodel.InitSystemAPIKeysModel(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}
	return model
}

func testMsg(name string, input data.Documenter) abstract.Message {
	return dispatch.NewMessage(name, context.Background(), input)
}

func TestListAPIKeysHandler(t *testing.T) {
	ctx := context.Background()
	model := testModel(t)

	gen, err := model.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, err = model.CreateKey(ctx, gen, "test-user", &apikeysmodel.APIKeyCreate{Name: "my-key"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	handler := apikeys.NewListAPIKeysHandler(model)
	input := testutil.InputDoc(t, apikeysmodel.APIKeyListInputSchema(), `{
		"arguments": { "user_id": "test-user" }
	}`)
	result, err := handler(ctx, testMsg("list", input))
	if err != nil {
		t.Fatalf("list handler: %v", err)
	}
	if len(result.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(result.Documents))
	}
	name, _ := result.Documents[0].GetString("name")
	if name != "my-key" {
		t.Errorf("name = %q, want %q", name, "my-key")
	}
}

func TestGetAPIKeyHandler(t *testing.T) {
	ctx := context.Background()
	model := testModel(t)

	gen, err := model.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	created, err := model.CreateKey(ctx, gen, "test-user", &apikeysmodel.APIKeyCreate{Name: "my-key"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	handler := apikeys.NewGetAPIKeyHandler(model)
	claimsCtx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	input := testutil.InputDoc(t, apikeysmodel.APIKeyGetInputSchema(), `{
		"arguments": { "key_id": "`+created.ID+`" }
	}`)
	result, err := handler(claimsCtx, testMsg("get", input))
	if err != nil {
		t.Fatalf("get handler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil document")
	}
	name, _ := result.Document.GetString("name")
	if name != "my-key" {
		t.Errorf("name = %q, want %q", name, "my-key")
	}
}

func TestCreateAPIKeyHandler(t *testing.T) {
	ctx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	model := testModel(t)

	handler := apikeys.NewCreateAPIKeyHandler(model)
	input := testutil.InputDoc(t, apikeysmodel.APIKeyCreateInputSchema(), `{
		"payload": { "name": "new-key" }
	}`)
	msg := dispatch.NewMessage("create", ctx, input)
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil document")
	}
	name, _ := result.Document.GetString("name")
	if name != "new-key" {
		t.Errorf("name = %q, want %q", name, "new-key")
	}
	key, _ := result.Document.GetString("key")
	if key == "" {
		t.Error("expected non-empty key in result")
	}
}

func TestDeleteAPIKeyHandler(t *testing.T) {
	ctx := context.Background()
	model := testModel(t)

	gen, err := model.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	created, err := model.CreateKey(ctx, gen, "test-user", &apikeysmodel.APIKeyCreate{Name: "to-delete"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	claimsCtx := runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: "test-user"})
	handler := apikeys.NewDeleteAPIKeyHandler(model)
	input := testutil.InputDoc(t, apikeysmodel.APIKeyDeleteInputSchema(), `{
		"arguments": { "key_id": "`+created.ID+`" }
	}`)
	msg := dispatch.NewMessage("delete", claimsCtx, input)
	result, err := handler(claimsCtx, msg)
	if err != nil {
		t.Fatalf("delete handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	_, err = model.Get(ctx, created.ID, "test-user")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}
