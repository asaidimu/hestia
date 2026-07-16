package users_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

func testMsg(name string, input *data.Document) abstract.Message {
	return abstract.NewMessage(name, context.Background(), input)
}

func TestGetUserHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "get@test.com", "password123", "Get User")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := doc.ID()

	handler := users.NewGetUserHandler(model)
	msg := testMsg("system:users:user:get", data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"user_id": userID,
		},
	}))

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("GetUserHandler: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected non-nil Document")
	}

	name, _ := result.Document.GetString("name")
	email, _ := result.Document.GetString("email")
	if name != "Get User" {
		t.Errorf("name = %q, want %q", name, "Get User")
	}
	if email != "get@test.com" {
		t.Errorf("email = %q, want %q", email, "get@test.com")
	}
}

func TestUpdateUserHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "update@test.com", "password123", "Original Name")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := doc.ID()

	handler := users.NewUpdateUserHandler(model)
	msg := testMsg("system:users:user:update", data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"user_id": userID,
		},
		"payload": map[string]any{
			"name": "Updated Name",
		},
	}))

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("UpdateUserHandler: %v", err)
	}

	name, _ := result.Document.GetString("name")
	if name != "Updated Name" {
		t.Errorf("name = %q, want %q", name, "Updated Name")
	}
}

func TestChangePasswordHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "changepw@test.com", "oldPassword", "PW User")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := doc.ID()

	handler := users.NewChangePasswordHandler(model)
	msg := testMsg("system:users:password:change", data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"user_id": userID,
		},
		"payload": map[string]any{
			"current": "oldPassword",
			"new":     "newPassword",
		},
	}))

	_, err = handler(ctx, msg)
	if err != nil {
		t.Fatalf("ChangePasswordHandler: %v", err)
	}

	storedDoc, err := model.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	storedPassword, err := storedDoc.GetString("password")
	if err != nil {
		t.Fatalf("GetString(password): %v", err)
	}
	if !core.CheckPassword("newPassword", storedPassword) {
		t.Error("new password should match stored hash")
	}
	if core.CheckPassword("oldPassword", storedPassword) {
		t.Error("old password should not match stored hash")
	}
}

func TestDeleteUserHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	model := users.NewUserModel(p)

	doc, err := model.Register(ctx, "delete@test.com", "password123", "Delete User")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := doc.ID()

	handler := users.NewDeleteUserHandler(model)
	msg := testMsg("system:users:user:delete", data.MustNewDocument(map[string]any{
		"arguments": map[string]any{
			"user_id": userID,
		},
	}))

	_, err = handler(ctx, msg)
	if err != nil {
		t.Fatalf("DeleteUserHandler: %v", err)
	}

	doc, err = model.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByID after soft delete: %v", err)
	}
	if !model.IsDeleted(doc) {
		t.Error("expected user to be marked as deleted")
	}
}
