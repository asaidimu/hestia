package users_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/users"
	usermodel "github.com/asaidimu/hestia/core/system/users/model"
	"go.uber.org/zap"
)

// fixture holds a DI-built users service plus the generated registrations. The
// service exposes the domain methods used for setup/verification; the
// registrations expose the same methods bound through dispatch adapters — the
// exact path the boot wiring uses. This exercises annotation → registration →
// input binding → domain method → result document end to end.
type fixture struct {
	svc      *users.UsersService
	handlers map[string]abstract.MessageHandler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	usermodel.DangerouslyResetSystemUsersModel()

	rt := runtime.NewRuntime()
	if err := rt.RegisterInstance[persistence.Persistence](testutil.NewPersistence(t)); err != nil {
		t.Fatalf("RegisterInstance persistence: %v", err)
	}
	if err := rt.RegisterInstance[*zap.Logger](zap.NewNop()); err != nil {
		t.Fatalf("RegisterInstance logger: %v", err)
	}
	if err := users.RegisterService(rt); err != nil {
		t.Fatalf("RegisterService: %v", err)
	}
	if err := rt.Rebuild(); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	svc, err := users.NewUsersService(rt)
	if err != nil {
		t.Fatalf("NewUsersService: %v", err)
	}

	regs, err := users.Registrations(rt)
	if err != nil {
		t.Fatalf("Registrations: %v", err)
	}
	handlers := make(map[string]abstract.MessageHandler, len(regs))
	for _, r := range regs {
		handlers[r.Name] = r.Handler
	}
	return &fixture{svc: svc, handlers: handlers}
}

// testMsg wraps an input document into a dispatch message.
func testMsg(name string, input data.Documenter) abstract.Message {
	return dispatch.NewMessage(name, context.Background(), input)
}

func TestCreateUserHandler(t *testing.T) {
	fx := newFixture(t)
	handler := fx.handlers["system:users:user:create"]

	msg := testMsg("system:users:user:create", testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserRegisterInput]("input", true), `{
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
	fx := newFixture(t)
	handler := fx.handlers["system:users:user:get"]

	user, err := fx.svc.CreateUser(ctx, msg, &usermodel.UserRegisterInput{Email: "get@test.com", Password: "password123", Name: "Get User"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	userID := user.ID

	msg := testMsg("system:users:user:get", testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserGetInput]("input"), fmt.Sprintf(
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
	fx := newFixture(t)
	handler := fx.handlers["system:users:user:update"]

	user, err := fx.svc.CreateUser(ctx, msg, &usermodel.UserRegisterInput{Email: "update@test.com", Password: "password123", Name: "Original Name"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	userID := user.ID

	msg := testMsg("system:users:user:update", testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserUpdateInput]("input", true), fmt.Sprintf(
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
	fx := newFixture(t)
	handler := fx.handlers["system:users:password:change"]

	user, err := fx.svc.CreateUser(ctx, msg, &usermodel.UserRegisterInput{Email: "changepw@test.com", Password: "oldPassword", Name: "PW User"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	userID := user.ID

	msg := testMsg("system:users:password:change", testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserChangePasswordInput]("input"), fmt.Sprintf(
		`{"arguments":{"user_id":%s},"payload":{"current":"oldPassword","new":"newPassword"}}`, strconv.Quote(userID),
	)))

	_, err = handler(ctx, msg)
	if err != nil {
		t.Fatalf("ChangePasswordHandler: %v", err)
	}

	storedUser, err := fx.svc.GetUser(ctx, msg, &usermodel.UserGetInput{UserID: userID})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	ok, err := runtime.CheckPassword("newPassword", storedUser.Password)
	if err != nil {
		t.Fatalf("CheckPassword: %v", err)
	}
	if !ok {
		t.Error("new password should match stored hash")
	}
	ok, err = runtime.CheckPassword("oldPassword", storedUser.Password)
	if err != nil {
		t.Fatalf("CheckPassword: %v", err)
	}
	if ok {
		t.Error("old password should not match stored hash")
	}
}

func TestDeleteUserHandler(t *testing.T) {
	fx := newFixture(t)
	handler := fx.handlers["system:users:user:delete"]

	user, err := fx.svc.CreateUser(ctx, msg, &usermodel.UserRegisterInput{Email: "delete@test.com", Password: "password123", Name: "Delete User"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	userID := user.ID

	msg := testMsg("system:users:user:delete", testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserDeleteInput]("input"), fmt.Sprintf(
		`{"arguments":{"user_id":%s}}`, strconv.Quote(userID),
	)))

	_, err = handler(ctx, msg)
	if err != nil {
		t.Fatalf("DeleteUserHandler: %v", err)
	}

	_, err = fx.svc.GetUser(ctx, msg, &usermodel.UserGetInput{UserID: userID})
	if err == nil {
		t.Error("expected GetUser to return error after delete")
	}
}

var ctx = context.Background()
var msg = abstract.Message(nil)

func ptr[T any](v T) *T { return &v }
