package policies

import (
	"context"
	"fmt"
	"sync"

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
