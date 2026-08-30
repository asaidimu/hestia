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
	settingsmodel "github.com/asaidimu/hestia/core/system/settings/model"
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
	settingsmodel.DangerouslyResetSystemSettingssModel()
	settingsM, err := settingsmodel.InitSystemSettingssModel(p, nil)
	if err != nil {
		t.Fatalf("InitSystemSettingssModel: %v", err)
	}
	store := NewStore(settingsM)
	dataDir := t.TempDir()
	u, err := updater.New(provider, updater.Config{
		Version: currentVersion,
		DataDir: dataDir,
		Store:   store,
	})
	if err != nil {
		t.Fatalf("new updater: %v", err)
	}
	svc := NewServiceFromDeps(u, store, nil, nil, zap.NewNop(), "http://localhost", false, false, currentVersion, false, "", dataDir)
	return svc, store
}

// newSystemdService returns a service in SystemdMode whose staged update lands
// in dataDir and whose swap target is exePath.
func newSystemdService(t *testing.T, provider *stubProvider, currentVersion, dataDir, exePath string) (*UpdatesService, *Store) {
	t.Helper()
	p := testutil.NewPersistence(t)
	settingsmodel.DangerouslyResetSystemSettingssModel()
	settingsM, err := settingsmodel.InitSystemSettingssModel(p, nil)
	if err != nil {
		t.Fatalf("InitSystemSettingssModel: %v", err)
	}
	store := NewStore(settingsM)
	u, err := updater.New(provider, updater.Config{
		Version: currentVersion,
		DataDir: dataDir,
		Store:   store,
	})
	if err != nil {
		t.Fatalf("new updater: %v", err)
	}
	svc := NewServiceFromDeps(u, store, nil, nil, zap.NewNop(), "http://localhost", false, false, currentVersion, true, exePath, dataDir)
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
	if pending, _ := svc.store.PendingUpdate(ctx); pending != nil {
		t.Fatalf("expected pending metadata to be cleared after swap, got %+v", pending)
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

func TestStageBackfillsChecksumWhenProviderOmitsIt(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version: "1.2.0",
	}}, "1.0.0")

	staged, _, err := svc.stageLatest(ctx)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if staged == nil {
		t.Fatal("expected staged update")
	}

	pending, err := store.PendingUpdate(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending == nil || pending.Checksum == "" {
		t.Fatal("expected checksum to be backfilled into pending record")
	}
	actual, err := hashFile(svc.stagedBinaryPath())
	if err != nil {
		t.Fatalf("hash staged: %v", err)
	}
	if pending.Checksum != actual {
		t.Errorf("backfilled checksum %q does not match file digest %q", pending.Checksum, actual)
	}

	// Verification passes with the backfilled checksum.
	if err := svc.verifyStagedBinary(ctx); err != nil {
		t.Fatalf("verify with backfilled checksum: %v", err)
	}
}

func TestDiscardCleansUpStagedUpdate(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version: "1.2.0",
	}}, "1.0.0")

	// Stage an update first.
	if _, err := svc.Check(ctx, nil, nil); err != nil {
		t.Fatalf("check: %v", err)
	}
	if !svc.updater.HasPreparedUpdate() {
		t.Fatal("expected prepared update binary")
	}
	pending, err := store.PendingUpdate(ctx)
	if err != nil || pending == nil || pending.Version != "1.2.0" {
		t.Fatalf("expected staged 1.2.0, got %+v (err=%v)", pending, err)
	}

	// Discard it.
	view, err := svc.Discard(ctx, nil, nil)
	if err != nil {
		t.Fatalf("discard: %v", err)
	}
	if view.Message == "" {
		t.Fatal("expected a success message")
	}

	// Verify cleanup: staged binary removed and pending record cleared.
	if svc.updater.HasPreparedUpdate() {
		t.Fatal("expected staged binary to be removed after discard")
	}
	if pending, _ := store.PendingUpdate(ctx); pending != nil {
		t.Fatalf("expected pending record to be cleared after discard, got %+v", pending)
	}
}

func TestDiscardWhenNothingStaged(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")

	// Discard with nothing staged should succeed (no-op cleanup).
	view, err := svc.Discard(ctx, nil, nil)
	if err != nil {
		t.Fatalf("discard: %v", err)
	}
	if view.Message == "" {
		t.Fatal("expected a success message")
	}
}

func TestVerifyStagedBinaryDetectsTampering(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{info: &updater.UpdateInfo{
		Version:  "1.2.0",
		Checksum: "SHA256:deadbeef",
	}}, "1.0.0")

	if _, _, err := svc.stageLatest(ctx); err != nil {
		t.Fatalf("stage: %v", err)
	}

	// Wrong recorded checksum must fail verification.
	if err := svc.verifyStagedBinary(ctx); err == nil {
		t.Fatal("expected checksum mismatch error")
	}

	// Recording the real digest makes verification pass.
	digest, err := hashFile(svc.stagedBinaryPath())
	if err != nil {
		t.Fatalf("hash staged: %v", err)
	}
	pending, _ := store.PendingUpdate(ctx)
	pending.Checksum = digest
	if err := store.SaveUpdate(ctx, pending); err != nil {
		t.Fatalf("save pending: %v", err)
	}
	if err := svc.verifyStagedBinary(ctx); err != nil {
		t.Fatalf("verify should pass with correct checksum: %v", err)
	}
}

// TestVerifyStagedBinaryFailsClosedOnMissingRecord pins the S-9 contract: a
// staged binary with NO pending record must never pass verification.
func TestVerifyStagedBinaryFailsClosedOnMissingRecord(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, &stubProvider{}, "1.0.0")

	// A staged binary exists (as after an unclean shutdown or an old
	// version's staging), but the pending record is gone.
	if err := os.WriteFile(svc.stagedBinaryPath(), []byte("stale-binary"), 0755); err != nil {
		t.Fatalf("write staged binary: %v", err)
	}
	if err := svc.verifyStagedBinary(ctx); err == nil {
		t.Fatal("expected refusal to apply with no pending record (S-9 vacuous pass removed)")
	}
}

// TestVerifyStagedBinaryFailsClosedOnEmptyChecksum pins the other half of
// S-9: a pending record without a checksum must fail verification, not pass.
func TestVerifyStagedBinaryFailsClosedOnEmptyChecksum(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService(t, &stubProvider{}, "1.0.0")

	if err := os.WriteFile(svc.stagedBinaryPath(), []byte("legacy-staged-binary"), 0755); err != nil {
		t.Fatalf("write staged binary: %v", err)
	}
	if err := store.SaveUpdate(ctx, &updater.UpdateInfo{Version: "1.2.0"}); err != nil {
		t.Fatalf("save pending without checksum: %v", err)
	}
	if err := svc.verifyStagedBinary(ctx); err == nil {
		t.Fatal("expected refusal to apply with empty recorded checksum (S-9 vacuous pass removed)")
	}
}
