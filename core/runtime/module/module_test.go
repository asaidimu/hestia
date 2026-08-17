package module

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

type testModule struct{ BaseModule }

func (m *testModule) Name() string { return "test" }
func (m *testModule) Setup(_ context.Context, _ abstract.Container) error {
	return nil
}
func (m *testModule) Capabilities(_ abstract.Container) ([]abstract.Capability, error) {
	return nil, nil
}

func TestModuleInterface(t *testing.T) {
	var _ abstract.Module = (*testModule)(nil)
}
