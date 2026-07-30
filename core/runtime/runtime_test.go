package runtime

import (
	"os"
	"testing"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/data"
)

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	os.Exit(m.Run())
}
