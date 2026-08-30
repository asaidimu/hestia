package auth

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/system/auth/model"
)

// TokenBlocklist persists revoked session identifiers and consumed reset
// tokens in the `_token_blocklist_` collection. That collection existed as
// schema-and-migration only before; every revocation decision and reset
// consumption now flows through here.
//
// Rows carry the token's own expiry (exp is indexed), so the table is
// bounded: a blocked identifier only needs to outlive its token, after
// which the stateless HMAC fails expiry validation regardless.
//
// Key namespace in `jti`:
//   - session logout:    the raw session ID
//   - reset consumption: "reset:" + the token payload nonce
type TokenBlocklist struct {
	model  *model.SystemTokenBlocklists
	logger *zap.Logger
}

// NewTokenBlocklist initializes the blocklist model over the shared
// persistence layer. The model singleton is process-wide, so multiple
// wrappers (provider + service) observe the same rows.
func NewTokenBlocklist(persist persistence.Persistence, logger *zap.Logger) (*TokenBlocklist, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	m, err := model.InitSystemTokenBlocklistsModel(persist, logger)
	if err != nil {
		return nil, common.SystemErrorFrom(err).WithOperation("InitTokenBlocklist").WithMessage("init token blocklist model")
	}
	return &TokenBlocklist{model: m, logger: logger}, nil
}

// Revoke records jti as unusable until exp. Idempotent: revoking an
// already-recorded jti is a no-op, so concurrent logouts racing the unique
// index are not errors.
func (b *TokenBlocklist) Revoke(ctx context.Context, jti, userID string, exp int64) error {
	if b == nil || jti == "" {
		return nil
	}
	row := model.SystemTokenBlocklist{Jti: jti, UserID: userID, Exp: exp}
	if _, err := b.model.Create(ctx, model.NewSystemTokenBlocklist(row)); err != nil {
		if revoked, rerr := b.Revoked(ctx, jti); rerr == nil && revoked {
			return nil
		}
		return common.SystemErrorFrom(err).WithOperation("TokenBlocklist.Revoke").WithMessage("persist revoked token")
	}
	return nil
}

// Revoked reports whether jti has been revoked or consumed.
func (b *TokenBlocklist) Revoked(ctx context.Context, jti string) (bool, error) {
	if b == nil || jti == "" {
		return false, nil
	}
	q := query.NewQueryBuilder().Where("jti").Eq(jti).Limit(1).Build()
	rows, err := b.model.Read(ctx, &q)
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

// RevokedSafe is the fail-open variant used on the hot validation path: a
// blocklist infrastructure error must not lock out every session, but it
// must be loud. (Reset-token consumption is NOT subject to this — the
// unique-index insert there fails closed by construction.)
func (b *TokenBlocklist) RevokedSafe(ctx context.Context, jti string) bool {
	revoked, err := b.Revoked(ctx, jti)
	if err != nil {
		b.logger.Error("token blocklist check failed; failing open",
			zap.String("jti", jti), zap.Error(err))
		return false
	}
	return revoked
}

// Prune removes up to 100 expired rows. Called opportunistically from the
// logout and reset paths so the table self-cleans without a dedicated
// background worker.
func (b *TokenBlocklist) Prune(ctx context.Context) (int, error) {
	if b == nil {
		return 0, nil
	}
	q := query.NewQueryBuilder().Where("exp").Lt(time.Now().Unix()).Limit(100).Build()
	rows, err := b.model.Read(ctx, &q)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, row := range rows {
		if err := b.model.DeleteByID(ctx, row.ID); err != nil {
			b.logger.Warn("token blocklist prune: delete failed",
				zap.String("jti", row.Jti), zap.Error(err))
			continue
		}
		removed++
	}
	return removed, nil
}
