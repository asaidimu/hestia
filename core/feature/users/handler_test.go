package users_test

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/core/feature/users"
	usermodel "github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
)

// TestUpdateUserHandlerPreservesUnsetValues guards the end-to-end profile
// update path: a partial payload (name only) must not clobber the password or
// any other field the caller didn't include.
func TestUpdateUserHandlerPreservesUnsetValues(t *testing.T) {
	ctx := context.Background()
	m := testModel(t)

	created, err := m.Register(ctx, "handler-update@example.com", "s3cret-pass", "Original", "public", map[string]any{"plan": "pro"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := created.ID

	before, err := m.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	hashBefore := before.Password

	handler := users.NewUpdateUserHandler(m)
	input := testutil.InputDoc(t, usermodel.UserUpdateInputSchema(), `{
		"arguments": { "user_id": "`+id+`" },
		"payload": { "name": "Renamed" }
	}`)
	if _, err := handler(ctx, testMsg("update", input)); err != nil {
		t.Fatalf("update handler: %v", err)
	}

	updated, err := m.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
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
