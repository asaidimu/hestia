package policies

import (
	"context"
	"fmt"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/hestia/app/core"

	"github.com/asaidimu/go-anansi/v8/core/common"
)

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
		return "", core.ErrPermissionNotRegistered.WithOperation(msg.Name()).WithIssue(common.Issue{
			Path: msg.Name(),
		})
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
	ops, err := m.policyModel.ListOperations(ctx)
	if err != nil {
		return fmt.Errorf("reload policies: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	clear(m.scopes)
	clear(m.capabs)

	for _, op := range ops {
		m.scopes[op.Name] = op.RuleKey
		m.capabs[op.Name] = core.CapabilityMetadata{
			Name:        op.Name,
			Scope:       op.RuleKey,
			Description: op.Description,
		}
	}

	return nil
}

var _ core.PermissionManager = (*DBPermissionManager)(nil)

// LivePermissionManager implements PermissionManager backed by a
// LiveCollection of OperationPolicy documents.  Operations are loaded
// on demand via read-through cache; writes (seeding, EnsureOperation,
// DeleteOperation) update the LiveCollection which writes through to
// the database and refreshes the cache atomically.
type LivePermissionManager struct {
	liveOps collection.LiveCollection[*OperationPolicy]
	onEmpty []OperationPolicy // fallback if liveOps is empty
}

func NewLivePermissionManager(liveOps collection.LiveCollection[*OperationPolicy], onEmpty []OperationPolicy) *LivePermissionManager {
	return &LivePermissionManager{liveOps: liveOps, onEmpty: onEmpty}
}

func (m *LivePermissionManager) Resolve(msg core.Message) (string, error) {
	// LiveCollection does a read-through on cache miss, so if the
	// operation exists in the database it is served from the cache.
	op, ok := m.liveOps.Get(msg.Name())
	if ok && op != nil {
		return op.RuleKey, nil
	}
	// Fall back to default operations that may not have been seeded yet.
	for _, d := range m.onEmpty {
		if d.Name == msg.Name() {
			return d.RuleKey, nil
		}
	}
	return "", core.ErrPermissionNotRegistered.WithOperation(msg.Name())
}

func (m *LivePermissionManager) ListCapabilities() []core.CapabilityMetadata {
	seen := make(map[string]bool, len(m.onEmpty))
	for _, op := range m.onEmpty {
		seen[op.Name] = true
	}
	result := make([]core.CapabilityMetadata, 0, len(m.onEmpty))
	for _, op := range m.onEmpty {
		result = append(result, core.CapabilityMetadata{
			Name:        op.Name,
			Scope:       op.RuleKey,
			Description: op.Description,
		})
	}
	// Merge in any cached operations not already covered by defaults.
	for _, k := range m.liveOps.Keys() {
		if seen[k] {
			continue
		}
		op, ok := m.liveOps.Get(k)
		if !ok || op == nil {
			continue
		}
		result = append(result, core.CapabilityMetadata{
			Name:        op.Name,
			Scope:       op.RuleKey,
			Description: op.Description,
		})
	}
	return result
}

func (m *LivePermissionManager) Reload(ctx context.Context) error {
	return nil
}

var _ core.ReloadablePermissionManager = (*LivePermissionManager)(nil)
