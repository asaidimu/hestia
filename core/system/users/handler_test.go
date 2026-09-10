package users_test

import (
	"testing"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/users/model"
	usermodel "github.com/asaidimu/hestia/core/system/users/model"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// TestUserQueryInputPreservesQDSL guards the user:query filter regression:
// the input document pool is built from the registration schema (exactly
// what the HTTP layer does), and a QDSL body must survive it verbatim. The
// previous typed DTO (username/limit/cursor) silently stripped filter and
// pagination, so every user:query returned all users unfiltered.
func TestUserQueryInputPreservesQDSL(t *testing.T) {
	doc := testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserQueryInput]("input", true), `{
		"payload": {
			"filter": {"field": "email", "operator": "eq", "value": "someone@example.com"},
			"pagination": {"type": "offset", "offset": 0, "limit": 1}
		}
	}`)

	m := doc.ToMap()
	payload, _ := m["payload"].(map[string]any)
	if payload == nil {
		t.Fatalf("payload missing from the query input: %v", m)
	}
	rawFilter, _ := payload["filter"].(map[string]any)
	if rawFilter == nil {
		t.Fatalf("payload.filter was stripped from the query input: %v", payload)
	}
	if rawFilter["field"] != "email" || rawFilter["operator"] != "eq" || rawFilter["value"] != "someone@example.com" {
		t.Errorf("payload.filter = %v, want email eq someone@example.com", rawFilter)
	}
	if _, ok := payload["pagination"].(map[string]any); !ok {
		t.Errorf("payload.pagination was stripped from the query input: %v", payload)
	}

	var in model.UserQueryInput
	if err := doc.BindToTag(&in, "input"); err != nil {
		t.Fatalf("bind query input: %v", err)
	}
	if len(in.Payload) == 0 {
		t.Error("bound UserQueryInput.Payload is empty, want the QDSL body")
	}
}

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

	// Verify password is still valid by attempting to change it with the original password
	changeMsg := testMsg("system:users:password:change", testutil.InputDoc(t, dispatch.SchemaFromTypeWithTag[usermodel.UserChangePasswordInput]("input"), `{
		"arguments": { "user_id": "`+id+`" },
		"payload": { "current": "s3cret-pass", "new": "new-pass" }
	}`))
	if _, err := fx.handlers["system:users:password:change"](ctx, changeMsg); err != nil {
		t.Errorf("password change with original password should succeed: %v", err)
	}
}
