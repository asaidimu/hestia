package model

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

// Register creates a new enabled user. The persisted document always carries
// the domain defaults: disabled=-1, token_version=0, verified=false, and the
// granted permissions.
func (m *SystemUsers) Register(ctx context.Context, email, password, name, tenantID string, userData map[string]any, permissions ...string) (*SystemUser, error) {
	hashed, err := runtime.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	disabled := int64(-1)
	tokenVersion := int64(0)
	verified := false

	user := document.New(&SystemUser{
		Email:        email,
		Password:     hashed,
		Name:         name,
		Permissions:  permissions,
		Disabled:     &disabled,
		TokenVersion: &tokenVersion,
		Verified:     &verified,
	})

	if tenantID != "" {
		user.TenantID = &tenantID
	}

	if userData != nil {
		user.Data = userData
	}

	created, err := m.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

// GetByEmail returns the active user with the given email.
func (m *SystemUsers) GetByEmail(ctx context.Context, email string) (*SystemUser, error) {
	q := query.NewQueryBuilder().
		Where("email").Eq(email).
		Where("disabled").Eq(-1).
		Limit(1).
		Build()

	users, err := m.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return users[0], nil
}

// GetByID returns the user with the given ID regardless of disabled state.
func (m *SystemUsers) GetByID(ctx context.Context, id string) (*SystemUser, error) {
	user, err := m.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetActiveByID returns the enabled (disabled=-1) user with the given ID as a
// UserIdentity, satisfying abstract.UserResolver for the HTTP middleware.
func (m *SystemUsers) GetActiveByID(ctx context.Context, id string) (abstract.UserIdentity, error) {
	q := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(id).
		Where("disabled").Eq(-1).
		Limit(1).
		Build()

	users, err := m.Read(ctx, &q)
	if err != nil {
		return nil, fmt.Errorf("query active user: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return users[0], nil
}

// GetPasswordHash returns the stored bcrypt hash for the given user.
func (m *SystemUsers) GetPasswordHash(ctx context.Context, id string) (string, error) {
	user, err := m.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return user.Password, nil
}

func (m *SystemUsers) UpdateUserProfile(ctx context.Context, id string, patch *UserUpdate) (*SystemUser, error) {
	option := base.CollectionUpdate{}
	if patch.Disabled != nil {
		option.WithComputedField("token_version", query.NewQueryBuilder().Increment("token_version", 1).End().Build())
	}
	opts := []base.CollectionUpdate{ option }
	return m.UpdateFrom[*UserUpdate, *SystemUser](ctx, id, patch, opts...)
}

// ChangePassword validates user credentials and status before safely updated password,
// invalidating existing active sessions upon success.
func (m *SystemUsers) ChangePassword(ctx context.Context, id, currentPassword, newPassword string) error {
	// 1. Fetch target user
	user, err := m.GetByID(ctx, id)
	if err != nil {
		return runtime.ErrNotFound.WithOperation("change_password")
	}

	// 2. Validate state policies
	if user.Disabled != nil && *user.Disabled > 0 {
		return runtime.ErrUserDisabled.WithOperation("change_password")
	}

	// 3. Authenticate current credentials
	if !runtime.CheckPassword(currentPassword, user.Password) {
		return runtime.ErrInvalidCredentials.WithOperation("change_password")
	}

	// 4. Hash new credentials
	hashed, err := runtime.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 5. Update password
	if _, err := m.Update(ctx, id, &SystemUser{Password: hashed}); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	// 6. Invalidate active sessions upon password reset
	if _, err := m.IncrementTokenVersion(ctx, id); err != nil {
		return fmt.Errorf("invalidate active sessions: %w", err)
	}

	return nil
}

// UpdateEmail changes the user's email address via the typed update path.
func (m *SystemUsers) UpdateEmail(ctx context.Context, id, email string) error {
	if _, err := m.Update(ctx, id, &SystemUser{Email: email}); err != nil {
		return err
	}
	return nil
}

// IncrementTokenVersion bumps the user's token_version, invalidating any
// previously issued sessions.
func (m *SystemUsers) IncrementTokenVersion(ctx context.Context, id string) (*SystemUser, error) {
	user, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	next := int64(1)
	if user.TokenVersion != nil {
		next = *user.TokenVersion + 1
	}
	update := &SystemUser{TokenVersion: &next}
	user, err = m.Update(ctx, id, update)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Delete removes the given user.
func (m *SystemUsers) Delete(ctx context.Context, id string) error {
	return m.DeleteByID(ctx, id)
}

// GetEmail returns the user's email.
func (u *SystemUser) GetEmail() string { return u.Email }

// GetTenantID returns the user's tenant ID, or "" when unset.
func (u *SystemUser) GetTenantID() string {
	if u.TenantID == nil {
		return ""
	}
	return *u.TenantID
}

// GetPermissions returns the user's granted permissions.
func (u *SystemUser) GetPermissions() []string { return u.Permissions }

// GetTokenVersion returns the user's token version.
func (u *SystemUser) GetTokenVersion() int {
	if u.TokenVersion == nil {
		return 0
	}
	return int(*u.TokenVersion)
}
