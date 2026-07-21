package runtime

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/common"
)

type CapabilityMetadata struct {
	Name        string     `json:"name"`
	Type        IntentType `json:"type"`
	Scope       string     `json:"scope"`
	Description string     `json:"description"`
}

type PermissionManager interface {
	Resolve(msg Message) (ruleKey string, enabled bool, err error)
	ListCapabilities() []CapabilityMetadata
}

type ReloadablePermissionManager interface {
	PermissionManager
	Reload(ctx context.Context) error
}

type MapPermissionManager struct {
	scopes map[string]string
	capabs map[string]CapabilityMetadata
}

func NewMapPermissionManager() *MapPermissionManager {
	return &MapPermissionManager{
		scopes: make(map[string]string),
		capabs: make(map[string]CapabilityMetadata),
	}
}

func (m *MapPermissionManager) RegisterScope(name, scope, description string) {
	m.scopes[name] = scope
	m.capabs[name] = CapabilityMetadata{
		Name:        name,
		Scope:       scope,
		Description: description,
	}
}

func (m *MapPermissionManager) Resolve(msg Message) (string, bool, error) {
	scope, ok := m.scopes[msg.Name()]
	if !ok {
		return "", false, ErrPermissionNotRegistered.WithOperation(msg.Name()).WithIssue(common.Issue{
			Path: msg.Name(),
		})
	}
	return scope, true, nil
}

func (m *MapPermissionManager) ListCapabilities() []CapabilityMetadata {
	result := make([]CapabilityMetadata, 0, len(m.capabs))
	for _, c := range m.capabs {
		result = append(result, c)
	}
	return result
}

var _ PermissionManager = (*MapPermissionManager)(nil)
