package policies

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"

	"github.com/asaidimu/hestia/app/core"
)

// LivePermissionManager resolves operation names to rule keys backed by a
// LiveCollection of Policy documents.  Policies are loaded on demand via
// read-through cache; writes update the LiveCollection which writes through
// to the database and refreshes the cache atomically.
type LivePermissionManager struct {
	livePolicies collection.LiveCollection[*Policy]
	onEmpty      []Policy // fallback if livePolicies is empty
}

func NewLivePermissionManager(livePolicies collection.LiveCollection[*Policy], onEmpty []Policy) *LivePermissionManager {
	return &LivePermissionManager{livePolicies: livePolicies, onEmpty: onEmpty}
}

func (m *LivePermissionManager) Resolve(msg core.Message) (string, bool, error) {
	policy, ok := m.livePolicies.Get(msg.Name())
	if ok && policy != nil {
		return policy.RuleName, policy.Enabled, nil
	}
	for _, d := range m.onEmpty {
		if d.OperationName == msg.Name() {
			return d.RuleName, d.Enabled, nil
		}
	}
	return "", false, core.ErrOperationLacksPolicy.WithIssue(common.Issue{Message: fmt.Sprintf("command %s has no policy",msg.Name())}).WithOperation(msg.Name())
}

func (m *LivePermissionManager) ListCapabilities() []core.CapabilityMetadata {
	seen := make(map[string]bool, len(m.onEmpty))
	for _, policy := range m.onEmpty {
		seen[policy.OperationName] = true
	}
	result := make([]core.CapabilityMetadata, 0, len(m.onEmpty))
	for _, policy := range m.onEmpty {
		result = append(result, core.CapabilityMetadata{
			Name:        policy.OperationName,
			Scope:       policy.RuleName,
			Description: policy.OperationName,
		})
	}
	for _, k := range m.livePolicies.Keys() {
		if seen[k] {
			continue
		}
		policy, ok := m.livePolicies.Get(k)
		if !ok || policy == nil {
			continue
		}
		result = append(result, core.CapabilityMetadata{
			Name:        policy.OperationName,
			Scope:       policy.RuleName,
			Description: policy.OperationName,
		})
	}
	return result
}

func (m *LivePermissionManager) Reload(ctx context.Context) error {
	return nil
}

var _ core.ReloadablePermissionManager = (*LivePermissionManager)(nil)
