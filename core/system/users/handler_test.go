package users_test

import (
	"testing"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/users/model"
	usermodel "github.com/asaidimu/hestia/core/system/users/model"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// TestUpdateUserHandlerPreservesUnsetValues guards the end-to-end profile
// update path: a partial payload (name only) must not clobber the password or
// any other field the caller didn't include.
func TestUpdateUserHandlerPreservesUnsetValues(t *testing.T) {
	fx := newFixture(t)
	handler := fx.handlers["system:users:user:update"]

	created, err := fx.svc.CreateUser(ctx, msg, &model.UserRegisterInput{
		Email:    "handler-update@example.com",
		Password: "s3cret-pass",
		Name:     "Original",
		Data:     map[string]any{"plan": "pro"},
		TenantID: ptr("public"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	id := created.ID

	before, err := fx.svc.GetUser(ctx, msg, &model.UserGetInput{UserID: id})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	hashBefore := before.Password

	doc := testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserUpdateInput]("input", true), `{
		"arguments": { "user_id": "`+id+`" },
		"payload": { "name": "Renamed" }
	}`)
	if _, err := handler(ctx, testMsg("update", doc)); err != nil {
		t.Fatalf("update handler: %v", err)
	}

	updated, err := fx.svc.GetUser(ctx, msg, &model.UserGetInput{UserID: id})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Errorf("name = %q, want Renamed", updated.Name)
	}
	if updated.Password != hashBefore {
		t.Errorf("password hash was overwritten: got %q, want %q", updated.Password, hashBefore)
	}
	if updated.Email != "handler-update@example.com" {
		t.Errorf("email = %q, want unchanged", updated.Email)
	}
	if len(updated.Permissions) != 0 {
		t.Errorf("permissions = %v, want empty (unchanged)", updated.Permissions)
	}
	if updated.Data == nil || updated.Data["plan"] != "pro" {
		t.Errorf("data = %v, want preserved", updated.Data)
	}
	if updated.Disabled == nil || *updated.Disabled != -1 {
		t.Errorf("disabled = %v, want preserved -1", updated.Disabled)
	}
	if updated.Verified == nil || *updated.Verified {
		t.Errorf("verified = %v, want preserved false", updated.Verified)
	}
	if updated.TokenVersion == nil || *updated.TokenVersion != 0 {
		t.Errorf("token_version = %v, want preserved 0", updated.TokenVersion)
	}
	if updated.TenantID == nil || *updated.TenantID != "public" {
		t.Errorf("tenant_id = %v, want preserved public", updated.TenantID)
	}
}
