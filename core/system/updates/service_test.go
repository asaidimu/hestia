package updates

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	svc := NewService(u, store, nil, nil, zap.NewNop(), "http://localhost", false, false, currentVersion, false, "", "")
	return svc, store
}

// newSystemdService returns a service in SystemdMode whose staged update lands
// in dataDir and whose swap target is exePath.
func newSystemdService(t *testing.T, provider *stubProvider, currentVersion, dataDir, exePath string) (*UpdatesService, *Store) {
	t.Helper()
	p := testutil.NewPersistence(t)
	store := NewStore(settings.NewSettingsModel(p))
	u, err := updater.New(provider, updater.Config{
		Version: currentVersion,
		DataDir: dataDir,
		Store:   store,
	})
	if err != nil {
		t.Fatalf("new updater: %v", err)
	}
	svc := NewService(u, store, nil, nil, zap.NewNop(), "http://localhost", false, false, currentVersion, true, exePath, dataDir)
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

// TestCheckAvailabilityHasNoSideEffects pins the read-only contract of
// system:updates:check:get: a newer release is reported but nothing may be
// downloaded, staged, or recorded — that's what makes it safe for fleet
// coordinators to poll (see #review2-20260821-004).
func TestCheckAvailabilityHasNoSideEffects(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version:   "1.2.0",
		Changelog: "release notes",
	}}, "1.0.0")

	view, err := svc.CheckAvailability(ctx, nil, nil)
	if err != nil {
		t.Fatalf("check availability: %v", err)
	}
	if !view.Available || view.Version != "1.2.0" {
		t.Fatalf("unexpected availability view: %+v", view)
	}

	if pending, _ := store.PendingUpdate(ctx); pending != nil {
		t.Fatalf("availability check must not stage an update, got %+v", pending)
	}
	if svc.updater.HasPreparedUpdate() {
		t.Fatal("availability check must not download/prepare the binary")
	}
	if _, err := store.Settings.Get(ctx, "", lastCheckKey); err == nil {
		t.Fatal("availability check must not record the last-check time")
	}
}

func TestCheckAvailabilityUpToDate(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")
	view, err := svc.CheckAvailability(ctx, nil, nil)
	if err != nil {
		t.Fatalf("check availability: %v", err)
	}
	if view.Available || view.Version != "" {
		t.Fatalf("expected no availability, got %+v", view)
	}
}

func TestStageDownloadsWithoutApplying(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version: "1.2.0",
	}}, "1.0.0")

	view, err := svc.Stage(ctx, nil, nil)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if !view.Staged || view.Version != "1.2.0" {
		t.Fatalf("unexpected stage view: %+v", view)
	}
	if !svc.updater.HasPreparedUpdate() {
		t.Fatal("expected prepared update binary")
	}
	pending, err := store.PendingUpdate(ctx)
	if err != nil || pending == nil || pending.Version != "1.2.0" {
		t.Fatalf("expected staged 1.2.0 pending row, got %+v (err=%v)", pending, err)
	}
	// Staging records the last-check time; applying must stay manual.
	v, err := store.Settings.Get(ctx, "", lastCheckKey)
	if err != nil {
		t.Fatalf("last check not recorded: %v", err)
	}
	if _, ok := v.(map[string]any); !ok {
		t.Fatalf("last check value should be a record, got %T", v)
	}
}

func TestStageUpToDateIsNoOp(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")
	view, err := svc.Stage(ctx, nil, nil)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if view.Staged || view.Version != "" {
		t.Fatalf("expected a staging no-op, got %+v", view)
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

// TestSystemdApplySwapsExecutableAndExits verifies the systemd-native apply
// path: after check+stage, apply copies the staged binary over the executable
// and schedules a clean process exit instead of the spawn-and-swap handoff.
func TestSystemdApplySwapsExecutableAndExits(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	exePath := filepath.Join(t.TempDir(), "hestia")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("seed executable: %v", err)
	}

	svc, _ := newSystemdService(t, &stubProvider{info: &updater.UpdateInfo{Version: "1.2.0"}}, "1.0.0", dataDir, exePath)
	if _, err := svc.Check(ctx, nil, nil); err != nil {
		t.Fatalf("check: %v", err)
	}
	if !svc.updater.HasPreparedUpdate() {
		t.Fatal("expected prepared update binary")
	}

	exited := make(chan int, 1)
	exitProcess = func(code int) { exited <- code }
	defer func() { exitProcess = os.Exit }()

	view, err := svc.Apply(ctx, nil, nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if view.Message == "" {
		t.Fatal("expected an ack message")
	}
	select {
	case code := <-exited:
		if code != 0 {
			t.Fatalf("expected clean exit code 0, got %d", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a scheduled process exit after apply")
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read swapped executable: %v", err)
	}
	if string(got) != "fake-binary" {
		t.Fatalf("executable not swapped: got %q, want fake-binary", got)
	}
	if pending, _ := svc.store.PendingUpdate(ctx); pending == nil {
		t.Fatal("expected pending metadata to survive the swap")
	}
}

// TestSystemdApplyUpToDateDoesNothing verifies apply is a no-op when the app
// is already running the latest version.
func TestSystemdApplyUpToDateDoesNothing(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	exePath := filepath.Join(t.TempDir(), "hestia")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("seed executable: %v", err)
	}
	svc, _ := newSystemdService(t, &stubProvider{}, "1.0.0", dataDir, exePath)
	view, err := svc.Apply(ctx, nil, nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if view.Message == "" {
		t.Fatal("expected an ack message")
	}
	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if string(got) != "old-binary" {
		t.Fatalf("executable must not change when up to date: got %q", got)
	}
}
