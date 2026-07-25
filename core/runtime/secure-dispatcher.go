package runtime

import (
	"context"
	"slices"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/registration"
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
	next    Dispatcher
	permMgr PermissionManager
	ac      iam.AccessController
}

func NewSecureDispatcher(next Dispatcher, permMgr PermissionManager, ac iam.AccessController) *SecureDispatcher {
	return &SecureDispatcher{next: next, permMgr: permMgr, ac: ac}
}

func (d *SecureDispatcher) Wrap(next Dispatcher) Dispatcher {
	return &SecureDispatcher{next: next, permMgr: d.permMgr, ac: d.ac}
}

func (d *SecureDispatcher) Send(msg Message) (*registration.Result, error) {
	if !IsSystemIdentity(msg.Context()) {
		// API key operation-name gate: if the IAM identity carries an
		// "operations" property (set by ContextWithClaims for API key
		// auth), only named operations are permitted. A nil/missing
		// property means "allow all" (backward-compatible with keys
		// created before this feature).
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
		if ex, ok := msg.(ResourceContextExtractor); ok {
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

// isAnonymous reports whether the context carries an anonymous identity
// (user has no UserID set).
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

var _ Dispatcher = (*SecureDispatcher)(nil)
