package apikeys_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/internal/abstract"
	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/identity"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

func testMsg(name string, input *data.Document) abstract.Message {
	return abstract.NewMessage(name, context.Background(), input)
}

func TestListAPIKeysHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := apikeys.NewAPIKeyModel(p)

	gen, err := model.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, err = model.Create(ctx, gen, "test-user", &apikeys.CreateKeyRequest{Name: "my-key"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	handler := apikeys.NewListAPIKeysHandler(model)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"user_id": "test-user"},
	}, ctx)
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
	p := persistest.NewPersistence(t)
	model := apikeys.NewAPIKeyModel(p)

	gen, err := model.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	doc, err := model.Create(ctx, gen, "test-user", &apikeys.CreateKeyRequest{Name: "my-key"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	handler := apikeys.NewGetAPIKeyHandler(model)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"user_id": "test-user",
			"key_id":  doc.ID(),
		},
	}, ctx)
	result, err := handler(ctx, testMsg("get", input))
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
	ctx := identity.ContextWithClaims(context.Background(), &core.Claims{UserID: "test-user"})
	p := persistest.NewPersistence(t)
	model := apikeys.NewAPIKeyModel(p)

	handler := apikeys.NewCreateAPIKeyHandler(model)
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{"name": "new-key"},
	}, ctx)
	msg := abstract.NewMessage("create", ctx, input)
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
	p := persistest.NewPersistence(t)
	model := apikeys.NewAPIKeyModel(p)

	gen, err := model.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	doc, err := model.Create(ctx, gen, "test-user", &apikeys.CreateKeyRequest{Name: "to-delete"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	claimsCtx := identity.ContextWithClaims(context.Background(), &core.Claims{UserID: "test-user"})
	handler := apikeys.NewDeleteAPIKeyHandler(model)
	input := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"key_id": doc.ID()},
	}, claimsCtx)
	msg := abstract.NewMessage("delete", claimsCtx, input)
	result, err := handler(claimsCtx, msg)
	if err != nil {
		t.Fatalf("delete handler: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	_, err = model.Get(ctx, doc.ID(), "test-user")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}
