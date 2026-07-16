package identity

import (
	"context"

	"github.com/asaidimu/go-iam/v2/iam"
)

// SystemScopePrefix is the prefix for all system-level permission scopes.
// Override at build time: go build -ldflags '-X github.com/asaidimu/hestia/app/core/identity.SystemScopePrefix=hestia'
var SystemScopePrefix = "system"

var systemIdentity = iam.Identity{
	Permissions: []string{SystemScopePrefix + ":http"},
	Properties:  map[string]any{"system": "http"},
}

func SystemContext(ctx context.Context) context.Context {
	return iam.WithIdentity(ctx, systemIdentity)
}
