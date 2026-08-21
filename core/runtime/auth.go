// @note #arch-20260821-002 issue status=open priority=P1 tags=#arch,#duplication : Duplicate SystemScopePrefix variable
//
// SystemScopePrefix is defined in both:
// - runtime/auth.go:14 (with ldflags override comment)
// - runtime/context/identity.go:45
//
// Both are mutable `var` and can diverge independently. The one in auth.go has
// an ldflags override comment. The one in context/identity.go is used to construct
// systemIdentity (lines 47-50). These must be kept in sync manually.
//
// Resolution: Move to a single canonical location (probably the config package
// or abstract package) and reference it from both locations.
package runtime

import (
	"context"
	"strings"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
)

// SystemScopePrefix is the prefix for all system-level permission scopes.
// Override at build time: go build -ldflags '-X github.com/asaidimu/hestia/core/runtime.SystemScopePrefix=hestia'
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
