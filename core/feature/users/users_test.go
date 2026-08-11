package users_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/users"
	usermodel "github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"go.uber.org/zap"
)

func testModel(t *testing.T) *usermodel.SystemUsers {
	t.Helper()
	usermodel.DangerouslyResetSystemUsersModel()
	model, err := usermodel.InitSystemUsersModel(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("NewSystemUsers: %v", err)
	}
	return model
}

func testMsg(name string, input data.Documenter) abstract.Message {
	return dispatch.NewMessage(name, context.Background(), input)
}

func TestCreateUserHandler(t *testing.T) {
	ctx := context.Background()
	model := testModel(t)

	handler := users.NewCreateUserHandler(model)
	msg := testMsg("system:users:user:create", testutil.InputDoc(t, usermodel.UserRegisterInputSchema(), `{
		"payload": {
			"email":    "new@test.com",
			"password": "secret123",
			"name":     "New User"
		}
	}`))

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("CreateUserHandler: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("expected non-nil result document")
	}

	email, err := result.Document.GetString("email")
	if err != nil {
		t.Fatalf("result document missing email field: %v", err)
	}
	if email != "new@test.com" {
		t.Errorf("email = %q, want %q", email, "new@test.com")
	}
}

func TestGetUserHandler(t *testing.T) {
	ctx := context.Background()
	model := testModel(t)

	user, err := model.Register(ctx, "get@test.com", "password123", "Get User", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := user.ID

	handler := users.NewGetUserHandler(model)
	msg := testMsg("system:users:user:get", testutil.InputDoc(t, usermodel.UserGetInputSchema(), fmt.Sprintf(
		`{"arguments":{"user_id":%s}}`, strconv.Quote(userID),
	)))

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
	model := testModel(t)

	user, err := model.Register(ctx, "update@test.com", "password123", "Original Name", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := user.ID

	handler := users.NewUpdateUserHandler(model)
	msg := testMsg("system:users:user:update", testutil.InputDoc(t, usermodel.UserUpdateInputSchema(), fmt.Sprintf(
		`{"arguments":{"user_id":%s},"payload":{"name":"Updated Name"}}`, strconv.Quote(userID),
	)))

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
	model := testModel(t)

	user, err := model.Register(ctx, "changepw@test.com", "oldPassword", "PW User", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := user.ID

	handler := users.NewChangePasswordHandler(model)
	msg := testMsg("system:users:password:change", testutil.InputDoc(t, usermodel.UserChangePasswordInputSchema(), fmt.Sprintf(
		`{"arguments":{"user_id":%s},"payload":{"current":"oldPassword","new":"newPassword"}}`, strconv.Quote(userID),
	)))

	_, err = handler(ctx, msg)
	if err != nil {
		t.Fatalf("ChangePasswordHandler: %v", err)
	}

	storedUser, err := model.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !runtime.CheckPassword("newPassword", storedUser.Password) {
		t.Error("new password should match stored hash")
	}
	if runtime.CheckPassword("oldPassword", storedUser.Password) {
		t.Error("old password should not match stored hash")
	}
}

func TestDeleteUserHandler(t *testing.T) {
	ctx := context.Background()
	model := testModel(t)

	user, err := model.Register(ctx, "delete@test.com", "password123", "Delete User", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID := user.ID

	handler := users.NewDeleteUserHandler(model)
	msg := testMsg("system:users:user:delete", testutil.InputDoc(t, usermodel.UserDeleteInputSchema(), fmt.Sprintf(
		`{"arguments":{"user_id":%s}}`, strconv.Quote(userID),
	)))

	_, err = handler(ctx, msg)
	if err != nil {
		t.Fatalf("DeleteUserHandler: %v", err)
	}

	_, err = model.GetByID(ctx, userID)
	if err == nil {
		t.Error("expected GetByID to return error after delete")
	}
}
