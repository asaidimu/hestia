package users

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/utils"

	"github.com/asaidimu/hestia/core/runtime"
)

const userCollectionName = "_user_"

type UserModel struct {
	persistence base.Persistence
}

func NewUserModel(persistence base.Persistence) *UserModel {
	return &UserModel{persistence: persistence}
}

func (m *UserModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persistence.Collection(ctx, userCollectionName)
}

func (m *UserModel) Register(ctx context.Context, email, password, name string, permissions ...string) (*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access user collection: %w", err)
	}

	existingQ := query.NewQueryBuilder().Where("email").Eq(email).Build()
	existing, err := col.Read(ctx, &existingQ)
	if err != nil {
		return nil, fmt.Errorf("query existing user: %w", err)
	}
	if existing.Count > 0 {
		return nil, fmt.Errorf("email already exists")
	}

	hashed, err := runtime.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	if len(permissions) == 0 {
		permissions = []string{"read:*"}
	}

	doc := data.MustNewDocument(map[string]any{
		"email":       email,
		"password":    hashed,
		"name":        name,
		"verified":    false,
		"permissions": permissions,
	})

	result, err := col.CreateOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return result.Data, nil
}

func (m *UserModel) GetByEmail(ctx context.Context, email string) (*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access user collection: %w", err)
	}

	q := query.NewQueryBuilder().Where("email").Eq(email).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return result.Data[0], nil
}

func (m *UserModel) GetByID(ctx context.Context, id string) (*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access user collection: %w", err)
	}

	q := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build()
	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return result.Data[0], nil
}

func (m *UserModel) Update(ctx context.Context, id string, fields map[string]any) error {
	col, err := m.collection(ctx)
	if err != nil {
		return fmt.Errorf("access user collection: %w", err)
	}

	setDoc := data.Patch(fields).Document(ctx)
	_, err = col.Update(ctx, &base.CollectionUpdate{
		Set:    setDoc,
		Filter: query.NewQueryBuilder().Where(data.DocumentIDField).Eq(id).Build().Filters,
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (m *UserModel) GetPasswordHash(ctx context.Context, id string) (string, error) {
	doc, err := m.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return doc.GetString("password")
}

func (m *UserModel) ChangePassword(ctx context.Context, id, newPassword string) error {
	hashed, err := runtime.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return m.Update(ctx, id, map[string]any{"password": hashed})
}

func (m *UserModel) List(ctx context.Context, offset, limit int) ([]*data.Document, int, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("access user collection: %w", err)
	}

	q := query.NewQueryBuilder().Build()
	q.Pagination = &query.PaginationOptions{
		Type:         query.PaginationTypeOffset,
		Offset:       &offset,
		Limit:        limit,
		IncludeTotal: utils.PrimitivePtr(true),
	}

	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return result.Data, result.Count, nil
}

func (m *UserModel) GetActiveByID(ctx context.Context, id string) (*data.Document, error) {
	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access user collection: %w", err)
	}

	q := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(id).
		Where("deleted").NotExists().
		Build()

	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query active user: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return result.Data[0], nil
}

func (m *UserModel) IsDeleted(doc *data.Document) bool {
	deleted, err := doc.GetString("deleted")
	return err == nil && deleted != ""
}

func (m *UserModel) SoftDelete(ctx context.Context, id string) error {
	return m.Update(ctx, id, map[string]any{
		"deleted": time.Now().Format(time.RFC3339),
	})
}

func (m *UserModel) HardDelete(ctx context.Context, id string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return fmt.Errorf("access user collection: %w", err)
	}

	filter := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(id).
		Build().Filters

	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("hard delete user: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
