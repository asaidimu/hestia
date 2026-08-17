package users

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"
)

// UserClaims is a cached subset of a _user_ document used for identity
// resolution during authentication. It is compiled from the full document
// once and served from a LiveCollection for subsequent lookups.
type UserClaims struct {
	UserID       string
	Email        string
	TenantID     string
	Permissions  []string
	TokenVersion int
	Disabled     int
}

// UserClaimsDocProcessor compiles _user_ documents into UserClaims values.
type UserClaimsDocProcessor struct{}

func (p *UserClaimsDocProcessor) Compile(ctx context.Context, doc data.Documenter) (*UserClaims, error) {
	email, _ := doc.GetString("email")
	tenantID, _ := doc.GetString("tenant_id")
	perms, _ := doc.GetStringArray("permissions")
	tv, _ := doc.GetInt("token_version")
	disabled, _ := doc.GetInt("disabled")

	return &UserClaims{
		UserID:       doc.ID(),
		Email:        email,
		TenantID:     tenantID,
		Permissions:  perms,
		TokenVersion: tv,
		Disabled:     disabled,
	}, nil
}

func (p *UserClaimsDocProcessor) Create(ctx context.Context, doc data.Documenter) (*UserClaims, error) {
	return p.Compile(ctx, doc)
}

func (p *UserClaimsDocProcessor) Destroy(ctx context.Context, c *UserClaims) error {
	return nil
}

func (p *UserClaimsDocProcessor) CloneState(c *UserClaims) (*UserClaims, error) {
	if c == nil {
		return nil, nil
	}
	cp := *c
	cp.Permissions = append([]string{}, c.Permissions...)
	return &cp, nil
}
