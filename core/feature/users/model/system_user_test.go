package model_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"go.uber.org/zap"
)

func newTestModel(t *testing.T) *model.SystemUsers {
	t.Helper()
	model.DangerouslyResetSystemUsersModel()
	m, err := model.InitSystemUsersModel(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemUsersModel: %v", err)
	}
	return m
}

func TestUserModelRegisterAndGet(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	user, err := m.Register(ctx, "alice@example.com", "p4ssw0rd", "Alice", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	byEmail, err := m.GetByEmail(ctx, "alice@example.com")
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

	byID, err := m.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.ID != id {
		t.Errorf("GetByID returned doc %s, want %s", byID.ID, id)
	}
}

func TestUserModelUpdate(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	user, err := m.Register(ctx, "bob@example.com", "p4ssw0rd", "Bob", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	update := data.New(&model.SystemUser{Name: "Robert"})
	if _, err := m.Update(ctx, id, update); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := m.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Name != "Robert" {
		t.Errorf("name = %q, want %q", updated.Name, "Robert")
	}
}

func TestUserModelPassword(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	user, err := m.Register(ctx, "carol@example.com", "first-pass", "Carol", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	hash, err := m.GetPasswordHash(ctx, id)
	if err != nil {
		t.Fatalf("GetPasswordHash: %v", err)
	}
	if hash == "" {
		t.Fatal("GetPasswordHash returned empty string")
	}
	if hash == "first-pass" {
		t.Error("hash should not equal plaintext password")
	}

	err = m.ChangePassword(ctx, id, "first-pass", "new-pass")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	newHash, err := m.GetPasswordHash(ctx, id)
	if err != nil {
		t.Fatalf("GetPasswordHash after change: %v", err)
	}
	if newHash == hash {
		t.Error("password hash should have changed")
	}
}

func TestUserModelDisable(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	user, err := m.Register(ctx, "dave@example.com", "p4ssw0rd", "Dave", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	disabled := int64(1)
	update := data.New(&model.SystemUser{Disabled: &disabled})
	if _, err := m.Update(ctx, id, update); err != nil {
		t.Fatalf("Update disabled: %v", err)
	}

	// GetByEmail should reject disabled user
	_, err = m.GetByEmail(ctx, "dave@example.com")
	if err == nil {
		t.Error("expected GetByEmail to return error for disabled user")
	}

	// GetActiveByID should reject disabled user
	_, err = m.GetActiveByID(ctx, id)
	if err == nil {
		t.Error("expected GetActiveByID to return error for disabled user")
	}

	// GetByID should still return the doc regardless
	_, err = m.GetByID(ctx, id)
	if err != nil {
		t.Errorf("GetByID should still return disabled user: %v", err)
	}
}

func TestUserModelDelete(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	user, err := m.Register(ctx, "eve@example.com", "p4ssw0rd", "Eve", "public", nil)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	id := user.ID

	err = m.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = m.GetByID(ctx, id)
	if err == nil {
		t.Error("expected GetByID to return error after Delete")
	}
}

func TestUserModelList(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	user1, _ := m.Register(ctx, "fay@example.com", "p4ss", "Fay", "public", nil)
	user2, _ := m.Register(ctx, "gia@example.com", "p4ss", "Gia", "public", nil)

	q := query.NewQueryBuilder().Limit(10).Build()
	docs, err := m.Read(ctx, &q)
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