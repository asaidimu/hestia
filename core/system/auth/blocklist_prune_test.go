package auth_test

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/core/internal/testutil"
	authmodel "github.com/asaidimu/hestia/core/system/auth/model"
)

func TestPruneBlocklist(t *testing.T) {
	// Reset the blocklist model singleton
	authmodel.DangerouslyResetSystemTokenBlocklistsModel()

	p := testutil.NewPersistence(t)
	svc := newTestAuthService(t, p)

	ctx := context.Background()
	msg := testMessage{name: "prune-blocklist", ctx: ctx}

	// Pruning an empty blocklist should succeed
	err := svc.PruneBlocklist(ctx, msg)
	if err != nil {
		t.Fatalf("PruneBlocklist on empty blocklist failed: %v", err)
	}
}
