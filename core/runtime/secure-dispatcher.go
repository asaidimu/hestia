// @note #arch-20260821-003 issue status=open priority=P1 tags=#arch,#duplication : Duplicated identity property extraction pattern
//
// The pattern `ident.Properties.(map[string]any)` followed by key extraction
// appears in 8+ places across 5 files:
//
// 1. secure-dispatcher.go:66,127
// 2. rate-limit.go:132,145
// 3. throttle.go:46
// 4. access-log-dispatcher.go:89
// 5. core/interface/http/middleware.go:144
//
// This is a mechanical pattern that should be a shared helper. The existing
// devnotes #review-20260821-006 and #review-20260821-007 call for a helper
// like `GetIdentityProperty[T](ctx, key)` or a typed `IdentityProperties` struct.
//
// Resolution: Create a `runtime/identity` helper package with generic functions
// like `GetProperty[T](ctx, key) (T, bool)` to eliminate all copies.
package runtime

import (
	"context"
	"slices"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
)

func stringSlice(v any) ([]string, bool) {
	switch s := v.(type) {
	case []string:
		return s, true
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out, true
	}
	return nil, false
}

type SecureDispatcher struct {
	next    abstract.Dispatcher
	permMgr PermissionManager
	ac      iam.AccessController
}

func NewSecureDispatcher(next abstract.Dispatcher, permMgr PermissionManager, ac iam.AccessController) *SecureDispatcher {
	return &SecureDispatcher{next: next, permMgr: permMgr, ac: ac}
}

func (d *SecureDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &SecureDispatcher{next: next, permMgr: d.permMgr, ac: d.ac}
}

// @note #review-20260821-006 issue status=open priority=P2 tags=#review,#security : Deeply nested identity property access
// The code accesses iam.Identity.Properties through multiple nested type assertions
// (ident.Properties.(map[string]any) then props["operations"]). This pattern is
// repeated in several places (rate-limit.go, throttle.go, access-log-dispatcher.go).
//
// Consider extracting a helper function like `GetIdentityProperty[T](ctx, key)` or
// defining a typed IdentityProperties struct to reduce duplication and improve type safety.
func (d *SecureDispatcher) Send(msg abstract.Message) (*abstract.Result, error) {
	if !IsSystemIdentity(msg.Context()) {
		// API key operation-name gate: if the IAM identity carries an
		// "operations" property (set by ContextWithClaims for API key
		// auth), only named operations are permitted. A nil/missing
		// property means "allow all" (backward-compatible with keys
		// created before this feature).

		// @note #83l69u todo : Investigate other methods of identification
		//
		// We need to investiagate other auth methods such as a list of trusted domains so that external events such as
		// Webhooks can be processed without requiring API Keys
		if ident, ok := iam.GetIdentity(msg.Context()); ok {
			if props, ok := ident.Properties.(map[string]any); ok {
				if raw, ok := props["operations"]; ok {
					if ops, ok := stringSlice(raw); ok && !slices.Contains(ops, msg.Name()) {
						return nil, ErrAccessDenied.WithIssues(common.Issues{
							common.Issue{
								Message: "operation not in API key allowlist",
								Path:    msg.Name(),
							},
						})
					}
				}
			}
		}

		ruleKey, enabled, err := d.permMgr.Resolve(msg)
		if err != nil {
			return nil, err
		}

		if !enabled {
			if isAnonymous(msg.Context()) {
				return nil, ErrAuthRequired
			}
			return nil, ErrAccessDenied.WithIssues(common.Issues{
				common.Issue{
					Message: "policy disabled",
					Path:    msg.Name(),
				},
			})
		}
		var resource any
		if ex, ok := msg.(abstract.ResourceContextExtractor); ok {
			resource = ex.ResourceContext()
		}

		can := d.ac.Can(msg.Context(), ruleKey, resource, nil)
		if !can {
			if isAnonymous(msg.Context()) {
				return nil, ErrAuthRequired
			}
			return nil, ErrAccessDenied.WithIssues(common.Issues{
				common.Issue{
					Message: ruleKey,
					Path:    msg.Name(),
				},
			})
		}
	}
	return d.next.Send(msg)
}

// @note #review-20260821-007 issue status=open priority=P2 tags=#review,#consolidation : Duplicate identity property extraction
// isAnonymous extracts user_id from iam.Identity.Properties using the same
// nested type assertion pattern found in extractUserID (rate-limit.go),
// extractAPIKeyID (rate-limit.go), and deriveActorType (access-log-dispatcher.go).
//
// Consider consolidating these into a single helper function in a shared package
// (e.g., runtime/identity) to reduce duplication and ensure consistent behavior.
func isAnonymous(ctx context.Context) bool {
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		return true
	}
	props, ok := ident.Properties.(map[string]any)
	if !ok {
		return true
	}
	uid, _ := props["user_id"].(string)
	return uid == ""
}

var _ abstract.Dispatcher = (*SecureDispatcher)(nil)
