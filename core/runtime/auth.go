package runtime

import (
	"context"
	"strings"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
)

func IsSystemIdentity(ctx context.Context) bool {
	identity, ok := iam.GetIdentity(ctx)
	if !ok {
		return false
	}
	for _, p := range identity.Permissions {
		if strings.HasPrefix(p, abstract.SystemScopePrefix+":") {
			return true
		}
	}
	return false
}

type Claims = abstract.Claims
