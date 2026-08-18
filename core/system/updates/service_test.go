package updates

import (
	"context"
	"os"
	"testing"

	"github.com/asaidimu/updater"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/settings"
)

type stubProvider struct {
	info *updater.UpdateInfo
	err  error
}

func (p *stubProvider) CheckForUpdate(ctx context.Context, currentVersion string) (*updater.UpdateInfo, error) {
	return p.info, p.err
}

func (p *stubProvider) DownloadUpdate(ctx context.Context, info *updater.UpdateInfo, destPath string) error {
	return os.WriteFile(destPath, []byte("fake-binary"), 0755)
}

func newTestService(t *testing.T, provider *stubProvider, currentVersion string) (*UpdatesService, *Store) {
	t.Helper()
	p := testutil.NewPersistence(t)
	store := NewStore(settings.NewSettingsModel(p))
	u, err := updater.New(provider, updater.Config{
		Version: currentVersion,
		DataDir: t.TempDir(),
		Store:   store,
	})
	if err != nil {
		t.Fatalf("new updater: %v", err)
	}
	svc := NewService(u, store, nil, nil, zap.NewNop(), "http://localhost", false, false, currentVersion)
	return svc, store
}

func TestStatusNoPending(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")
	view, err := svc.Status(ctx, nil, nil)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if view.Version != "1.0.0" || view.StagedVersion != "" || view.Prepared || view.LastCheck != 0 {
		t.Fatalf("unexpected status view: %+v", view)
	}
}

func TestCheckStagesNewerVersion(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version:   "1.2.0",
		Changelog: "release notes",
	}}, "1.0.0")

	view, err := svc.Check(ctx, nil, nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !view.Checked || !view.Staged || view.Version != "1.2.0" || view.AutoApply {
		t.Fatalf("unexpected check view: %+v", view)
	}

	pending, err := store.PendingUpdate(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending == nil || pending.Version != "1.2.0" {
		t.Fatalf("expected staged 1.2.0, got %+v", pending)
	}

	// Last check recorded.
	v, err := svc.store.Settings.Get(ctx, "", lastCheckKey)
	if err != nil {
		t.Fatalf("last check not recorded: %v", err)
	}
	if _, ok := v.(map[string]any); !ok {
		t.Fatalf("last check value should be a record, got %T", v)
	}
}

func TestCheckUpToDate(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")
	view, err := svc.Check(ctx, nil, nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !view.Checked || view.Staged || view.Version != "" {
		t.Fatalf("expected up to date, got %+v", view)
	}
}

func TestCheckIgnoresOlderVersion(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{Version: "0.9.0"}}, "1.0.0")
	view, err := svc.Check(ctx, nil, nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if view.Staged {
		t.Fatalf("older version must not be staged: %+v", view)
	}
	if pending, _ := store.PendingUpdate(ctx); pending != nil {
		t.Fatalf("expected no pending, got %+v", pending)
	}
}

func TestChangelogReturnsPending(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version: "1.2.0", Changelog: "release notes", AssetName: "app"},
	}, "1.0.0")
	if _, err := svc.Check(ctx, nil, nil); err != nil {
		t.Fatalf("check: %v", err)
	}
	view, err := svc.Changelog(ctx, nil, nil)
	if err != nil {
		t.Fatalf("changelog: %v", err)
	}
	if view.Version != "1.2.0" || view.Changelog != "release notes" || view.AssetName != "app" {
		t.Fatalf("unexpected changelog view: %+v", view)
	}
	if pending, _ := store.PendingUpdate(ctx); pending == nil {
		t.Fatal("expected pending after check")
	}
}

func TestRunScheduledCheckStages(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{info: &updater.UpdateInfo{Version: "1.2.0"}}, "1.0.0")
	if err := svc.RunScheduledCheck(ctx); err != nil {
		t.Fatalf("scheduled check: %v", err)
	}
	if !svc.updater.HasPreparedUpdate() {
		t.Fatal("expected prepared update binary")
	}
}

func TestApplyFailsWithoutPreparedUpdate(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")
	if _, err := svc.Apply(ctx, nil, nil); err == nil {
		t.Fatal("expected apply to fail without a prepared update")
	}
}