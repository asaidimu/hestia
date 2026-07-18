package policies

import (
	"context"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"

	"github.com/asaidimu/hestia/app/core"
)

// DBPermissionManager resolves operation names to rule keys using an in-memory
// map populated at startup via RegisterScope.  Used as a fallback when the
// LiveCollection-backed permission manager cannot be initialised.
type DBPermissionManager struct {
	mu     sync.RWMutex
	scopes map[string]string
	capabs map[string]core.CapabilityMetadata

	policyModel *PolicyModel
}

func NewDBPermissionManager(policyModel *PolicyModel) *DBPermissionManager {
	return &DBPermissionManager{
		scopes:      make(map[string]string),
		capabs:      make(map[string]core.CapabilityMetadata),
		policyModel: policyModel,
	}
}

func (m *DBPermissionManager) Resolve(msg core.Message) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	scope, ok := m.scopes[msg.Name()]
	if !ok {
		return "", core.ErrOperationLacksPolicy.WithOperation(msg.Name())
	}
	return scope, nil
}

func (m *DBPermissionManager) ListCapabilities() []core.CapabilityMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]core.CapabilityMetadata, 0, len(m.capabs))
	for _, c := range m.capabs {
		result = append(result, c)
	}
	return result
}

func (m *DBPermissionManager) RegisterScope(name, scope, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scopes[name] = scope
	m.capabs[name] = core.CapabilityMetadata{
		Name:        name,
		Scope:       scope,
		Description: description,
	}
}

func (m *DBPermissionManager) Reload(ctx context.Context) error {
	policies, err := m.policyModel.ListPolicies(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	clear(m.scopes)
	clear(m.capabs)

	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		m.scopes[p.OperationName] = p.RuleName
		m.capabs[p.OperationName] = core.CapabilityMetadata{
			Name:        p.OperationName,
			Scope:       p.RuleName,
			Description: p.OperationName,
		}
	}

	return nil
}

var _ core.ReloadablePermissionManager = (*DBPermissionManager)(nil)

// LivePermissionManager implements PermissionManager backed by a
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

func (m *LivePermissionManager) Resolve(msg core.Message) (string, error) {
	policy, ok := m.livePolicies.Get(msg.Name())
	if ok && policy != nil {
		if !policy.Enabled {
			return "", core.ErrOperationLacksPolicy.WithOperation(msg.Name())
		}
		return policy.RuleName, nil
	}
	for _, d := range m.onEmpty {
		if d.OperationName == msg.Name() {
			if !d.Enabled {
				return "", core.ErrOperationLacksPolicy.WithOperation(msg.Name())
			}
			return d.RuleName, nil
		}
	}
	return "", core.ErrOperationLacksPolicy.WithOperation(msg.Name())
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
