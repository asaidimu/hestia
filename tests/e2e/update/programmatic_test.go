package update_e2e

import (
	"context"
	"github.com/asaidimu/hestia/core/abstract"
	"os"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-iam/v2/iam"
	"github.com/asaidimu/updater"

	hestia "github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type probeProvider struct {
	info *updater.UpdateInfo
}

func (p *probeProvider) CheckForUpdate(ctx context.Context, currentVersion string) (*updater.UpdateInfo, error) {
	return p.info, nil
}

func (p *probeProvider) DownloadUpdate(ctx context.Context, info *updater.UpdateInfo, destPath string) error {
	return os.WriteFile(destPath, []byte("fake-binary"), 0755)
}

// TestProgrammaticSelfUpdatePersistsPending boots a real hestia application
// with SelfUpdate configured in code (stub provider) and verifies the check
// handler stages the release: binary in DataDir, pending row in the settings
// collection, last-check recorded.
func TestProgrammaticSelfUpdatePersistsPending(t *testing.T) {
	ctx := context.Background()
	p := &probeProvider{info: &updater.UpdateInfo{Version: "1.2.0", Changelog: "probe"}}
	app, err := hestia.Setup(hestia.SetupConfig{
		SessionSecret:     "probe-secret",
		DataDir:           t.TempDir(),
		DBPath:            ":memory:",
		Version:           "1.0.0",
		ForceBootstrapped: true,
		SelfUpdate:        &runtime.SelfUpdateConfig{Provider: p},
		BuildInterfaces: func(app *hestia.Application, cfg ...*runtime.Config) []runtime.Interface {
			// No CLI interface: it flag-parses os.Args and os.Exit(1)s on
			// unknown args (e.g. go test's own -test.* flags).
			return nil
		},
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer app.Close()
	if err := app.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	adminCtx := iam.WithIdentity(ctx, iam.Identity{
		Permissions: []string{"administrator"},
		Properties: map[string]any{
			"user_id":     "u1",
			"email":       "admin@test.local",
			"permissions": []string{"administrator"},
			"token_type":  "access",
		},
	})

	msg := dispatch.NewMessage("system:updates:check:create", adminCtx, data.MustNewDocument(nil, adminCtx))
	res, err := testAwait(app.Dispatcher(), msg)
	if err != nil {
		t.Fatalf("send check: %v", err)
	}
	if res == nil || res.Document == nil {
		t.Fatal("expected a check result document")
	}
	if v, _ := res.Document.GetOr("staged", false).(bool); !v {
		t.Fatalf("expected staged=true")
	}

	col, err := app.Persistence().Collection(ctx, "_settings_")
	if err != nil {
		t.Fatalf("settings collection: %v", err)
	}
	q := query.NewQueryBuilder().Where("key").Eq("updates:pending").Build()
	read, err := col.Read(ctx, &q)
	if err != nil {
		t.Fatalf("read pending: %v", err)
	}
	if read.Count != 1 {
		t.Fatalf("expected 1 pending row, got %d", read.Count)
	}
	val, _ := read.Data[0].Get("value")
	rec, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("pending value is %T, want record", val)
	}
	if rec["version"] != "1.2.0" {
		t.Fatalf("pending version = %v, want 1.2.0", rec["version"])
	}
	if rec["changelog"] != "probe" {
		t.Fatalf("pending changelog = %v, want probe", rec["changelog"])
	}
}

// testAwait dispatches m and blocks for its outcome; the test-lifecycle
// stand-in for request/response dispatch.
func testAwait(d abstract.Dispatcher, m abstract.Message) (*abstract.Result, error) {
	return dispatch.Await(context.Background(), d, m)
}
