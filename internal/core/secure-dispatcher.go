package core

import (
	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/internal/core/registration"
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
		ruleKey, err := d.permMgr.Resolve(msg)
		if err != nil {
			return nil, err
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
