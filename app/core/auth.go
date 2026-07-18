package core

import (
	"context"
	"strings"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/abstract"
)

// SystemScopePrefix is the prefix for all system-level permission scopes.
// Override at build time: go build -ldflags '-X github.com/asaidimu/hestia/app/core.SystemScopePrefix=hestia'
var SystemScopePrefix = "system"

func IsSystemIdentity(ctx context.Context) bool {
	identity, ok := iam.GetIdentity(ctx)
	if !ok {
		return false
	}
	for _, p := range identity.Permissions {
		if strings.HasPrefix(p, SystemScopePrefix+":") {
			return true
		}
	}
	return false
}

type Claims = abstract.Claims
