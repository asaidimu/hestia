package module

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/hestia/core/abstract"
)

type testModule struct{ BaseModule }

func (m *testModule) Name() string                                      { return "test" }
func (m *testModule) Setup(_ context.Context, _ base.Persistence) error { return nil }
func (m *testModule) Capabilities() []abstract.Capability               { return nil }

func TestModuleInterface(t *testing.T) {
	var _ abstract.Module = (*testModule)(nil)
}
