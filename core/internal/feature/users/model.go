package users

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/utils"

	"github.com/asaidimu/hestia/core/runtime"
)

// IdentityDocProcessor returns the document unchanged — no compilation step.
// Useful when the LiveCollection stores raw documents rather than derived artifacts.
type IdentityDocProcessor struct{}

func (p *IdentityDocProcessor) Compile(ctx context.Context, doc *data.Document) (*data.Document, error) {
	return doc, nil
}
func (p *IdentityDocProcessor) Create(ctx context.Context, doc *data.Document) (*data.Document, error) {
	return doc, nil
}
func (p *IdentityDocProcessor) Destroy(ctx context.Context, doc *data.Document) error {
	return nil
}
func (p *IdentityDocProcessor) CloneState(doc *data.Document) (*data.Document, error) {
	return doc, nil
}

type UserModel struct {
	persistence base.Persistence
	docCache    collection.LiveCollection[*data.Document]
}

func NewUserModel(persistence base.Persistence) *UserModel {
	return &UserModel{persistence: persistence}
}

// UseDocCache sets the document cache for read-through lookups and write interception.
func (m *UserModel) UseDocCache(c collection.LiveCollection[*data.Document]) {
	m.docCache = c
}

func (m *UserModel) collection(ctx context.Context) (base.Collection, error) {
	if m.docCache != nil {
		return m.docCache, nil
	}
	return m.persistence.Collection(ctx, "_user_")
}

func (m *UserModel) Register(ctx context.Context, email, password, name, tenantID string, userData map[string]any, permissions ...string) (*data.Document, error) {

	col, err := m.collection(ctx)
	if err != nil {
		return nil, fmt.Errorf("access user collection: %w", err)
	}

	// TODO email string should be in a constant
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
		"email":         email,
		"password":      hashed,
		"name":          name,
		"verified":      false,
		"permissions":   permissions,
		"token_version": 0,
	})

	if tenantID != "" {
		doc.Set("tenant_id", tenantID)
	}
	if userData != nil {
		doc.Set("data", userData)
	}

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
	doc := result.Data[0]
	disabled, _ := doc.GetInt("disabled")
	if disabled != 0 {
		return nil, fmt.Errorf("user not found")
	}

	id := doc.ID()
	if m.docCache != nil {
		m.docCache.Set(id, doc)
	}

	return doc, nil
}

func (m *UserModel) GetByID(ctx context.Context, id string) (*data.Document, error) {
	if m.docCache != nil {
		doc, ok := m.docCache.Get(id)
		if !ok {
			return nil, fmt.Errorf("user not found")
		}
		return doc, nil
	}

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
		Build()

	result, err := col.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query active user: %w", err)
	}
	if result.Count == 0 {
		return nil, fmt.Errorf("user not found")
	}
	doc := result.Data[0]
	disabled, _ := doc.GetInt("disabled")
	if disabled != 0 {
		return nil, fmt.Errorf("user not found")
	}
	return doc, nil
}

func (m *UserModel) IncrementTokenVersion(ctx context.Context, id string) error {
	user, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}
	v, _ := user.GetInt("token_version")
	return m.Update(ctx, id, map[string]any{"token_version": v + 1})
}

func (m *UserModel) GetSettings(ctx context.Context, id string) (map[string]any, error) {
	user, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	raw, err := user.Get("settings")
	if err != nil || raw == nil {
		return make(map[string]any), nil
	}
	s, ok := raw.(map[string]any)
	if !ok {
		return make(map[string]any), nil
	}
	return s, nil
}

func (m *UserModel) UpdateSettings(ctx context.Context, id string, settings map[string]any) error {
	user, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}
	raw, _ := user.Get("settings")
	current, _ := raw.(map[string]any)
	if current == nil {
		current = make(map[string]any)
	}
	for k, v := range settings {
		current[k] = v
	}
	return m.Update(ctx, id, map[string]any{"settings": current})
}

func (m *UserModel) Delete(ctx context.Context, id string) error {
	col, err := m.collection(ctx)
	if err != nil {
		return fmt.Errorf("access user collection: %w", err)
	}

	filter := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(id).
		Build().Filters

	deleted, err := col.Delete(ctx, filter, false)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
