package core

import (
	"slices"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core/registration"
)

type SecureDispatcher struct {
	next    Dispatcher
	permMgr PermissionManager
	ac      iam.AccessController
}

func NewSecureDispatcher(next Dispatcher, permMgr PermissionManager, ac iam.AccessController) *SecureDispatcher {
	return &SecureDispatcher{next: next, permMgr: permMgr, ac: ac}
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
				if ops, ok := props["operations"].([]string); ok && !slices.Contains(ops, msg.Name()) {
					return nil, ErrAccessDenied.WithIssues(common.Issues{
						common.Issue{
							Message: "operation not in API key allowlist",
							Path:    msg.Name(),
						},
					})
				}
			}
		}

		ruleKey, enabled, err := d.permMgr.Resolve(msg)
		if err != nil {
			return nil, err
		}
		if !enabled {
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

var _ Dispatcher = (*SecureDispatcher)(nil)
