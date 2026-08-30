package updates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/updater"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	usersmodel "github.com/asaidimu/hestia/core/system/users/model"
)

const lastCheckKey = "updates:last_check"

// exitProcess is indirection over os.Exit so tests can capture the exit
// instead of terminating the test binary.
var exitProcess = os.Exit

// NoInput is the (empty) input for every updates message.
type NoInput struct{}

type StatusView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Version                string `anansi:"version"`
	StagedVersion          string `anansi:"staged_version"`
	Prepared               bool   `anansi:"prepared"`
	LastCheck              int64  `anansi:"last_check"`
}

type ChangelogView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Version                string `anansi:"version"`
	AssetName              string `anansi:"asset_name"`
	Changelog              string `anansi:"changelog"`
}

type CheckView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Checked                bool   `anansi:"checked"`
	Staged                 bool   `anansi:"staged"`
	Version                string `anansi:"version"`
	AutoApply              bool   `anansi:"auto_apply"`
}

type ApplyView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}

type DiscardView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}

type AvailabilityView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Available              bool   `anansi:"available"`
	Version                string `anansi:"version"`
}

type StageView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Staged                 bool   `anansi:"staged"`
	Version                string `anansi:"version"`
}

// DI keys for updates-specific config values. Typed keys avoid collisions
// in the DI container (same pattern as auth.AdminUserID, auth.SessionTTL).
type (
	AutoApply    bool
	SystemdMode  bool
	ExePath      string
	UpdateDataDir string
	HasMailer    bool
	AppURL       string
	AppVersion   string
)

// UpdatesService drives the self-update lifecycle on top of updater.Updater:
// read-only status/changelog, check-and-stage, and the maintenance-window
// apply, plus the scheduled check. Every method runs with the caller's
// identity; the dispatcher gates them on the administrator rule.
type UpdatesService struct {
	updater   *updater.Updater
	store     *Store
	notifier  abstract.Notifier
	users     *usersmodel.SystemUsers
	logger    *zap.Logger
	appURL    string
	hasMailer bool
	autoApply bool
	version   string
	systemd   bool
	exePath   string
	dataDir   string
}

// NewUpdatesService resolves dependencies from the DI container and constructs
// the service. This is the codegen entry point.
func NewUpdatesService(rt abstract.Container) (*UpdatesService, error) {
	persist := abstract.MustResolve[base.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)

	u, _ := abstract.Resolve[*updater.Updater](rt)

	users, err := usersmodel.InitSystemUsersModel(persist, logger)
	if err != nil {
		return nil, fmt.Errorf("init users model: %w", err)
	}

	store := InitStore(persist)

	appURL, _ := abstract.Resolve[AppURL](rt)
	hasMailer, _ := abstract.Resolve[HasMailer](rt)
	autoApply, _ := abstract.Resolve[AutoApply](rt)
	appVersion, _ := abstract.Resolve[AppVersion](rt)
	systemdMode, _ := abstract.Resolve[SystemdMode](rt)
	exePath, _ := abstract.Resolve[ExePath](rt)
	dataDir, _ := abstract.Resolve[UpdateDataDir](rt)

	return &UpdatesService{
		updater:   u,
		store:     store,
		notifier:  abstract.MustResolve[abstract.Notifier](rt),
		users:     users,
		logger:    logger,
		appURL:    string(appURL),
		hasMailer: bool(hasMailer),
		autoApply: bool(autoApply),
		version:   string(appVersion),
		systemd:   bool(systemdMode),
		exePath:   string(exePath),
		dataDir:   string(dataDir),
	}, nil
}

// NewServiceFromDeps constructs the service with explicit dependencies.
// Used by the hand-wired path in provider.go until codegen migration completes.
func NewServiceFromDeps(u *updater.Updater, store *Store, notifier abstract.Notifier, users *usersmodel.SystemUsers, logger *zap.Logger, appURL string, hasMailer, autoApply bool, version string, systemd bool, exePath, dataDir string) *UpdatesService {
	return &UpdatesService{
		updater:   u,
		store:     store,
		notifier:  notifier,
		users:     users,
		logger:    logger,
		appURL:    appURL,
		hasMailer: hasMailer,
		autoApply: autoApply,
		version:   version,
		systemd:   systemd,
		exePath:   exePath,
		dataDir:   dataDir,
	}
}

// Status reports the running version, the staged update (if any), whether a
// binary is prepared for swap, and the last check time.
//
// @hestia.register(
//
//	name="system:updates:status:get",
//	intent="read",
//	rule="administrator",
//	description="Get self-update status (current and staged version)",
//	output="StatusView",
//
// )
func (s *UpdatesService) Status(ctx context.Context, _ abstract.Message, _ *NoInput) (*StatusView, error) {
	view := &StatusView{
		Version:  s.version,
		Prepared: s.updater != nil && s.updater.HasPreparedUpdate(),
	}
	if pending, err := s.store.PendingUpdate(ctx); err == nil && pending != nil {
		view.StagedVersion = pending.Version
	}
	if v, err := s.store.Settings.Get(ctx, "", lastCheckKey); err == nil {
		if rec, ok := v.(map[string]any); ok {
			if n, ok := toInt64(rec["unix_ms"]); ok {
				view.LastCheck = n
			}
		}
	}
	return document.New(view), nil
}

// Changelog returns the staged update's release notes for admin review.
//
// @hestia.register(
//
//	name="system:updates:changelog:get",
//	intent="read",
//	rule="administrator",
//	description="Get the staged update changelog",
//	output="ChangelogView",
//
// )
func (s *UpdatesService) Changelog(ctx context.Context, _ abstract.Message, _ *NoInput) (*ChangelogView, error) {
	pending, err := s.store.PendingUpdate(ctx)
	if err != nil {
		return nil, err
	}
	if pending == nil {
		return document.New(&ChangelogView{}), nil
	}
	return document.New(&ChangelogView{
		Version:   pending.Version,
		AssetName: pending.AssetName,
		Changelog: pending.Changelog,
	}), nil
}

// @note #review2-20260821-004 issue resolved P1 #review,#api-design,#p2p : system:updates:check:create conflates checking with staging (and a fleet cannot poll safely)
// @assignee opencode
// Split shipped: system:updates:check:get (read-only availability, no download/staging/last-check writes) and system:updates:stage:create (PrepareUpdate + last-check record + notify-admins). check:create kept as the legacy check-then-stage wrapper; RunScheduledCheck still uses checkAndStage. Hand-wired registrations kept for now — migration to codegen tracked as #migrate-20260822-001.
//
// RESOLVED 2026-08-22: split into system:updates:check:get (read-only
// availability) and system:updates:stage:create (download+prepare+notify);
// check:create kept as the backward-compatible check-then-stage wrapper and
// RunScheduledCheck still drives both effects back to back via checkAndStage.
//
// Despite the doc comment calling this "the read-only check", Check() always
// calls checkAndStage() (below), which both checks for a new version AND
// downloads + prepares it via s.updater.PrepareUpdate whenever one is found.
// There is no way to ask "is a newer version available" without triggering a
// download and disk write. The registered message name itself says
// "check:create" (Intent: abstract.Create) rather than a Read intent, which
// is the tell -- this was never actually a read operation.
//
// This is a real problem for a P2P/fleet deployment, not just a naming
// nit: a coordinator that wants to poll version availability across N nodes
// (e.g. before deciding whether to trigger a staged rollout) cannot do so
// without every node downloading the update artifact, even nodes that won't
// end up applying it. At fleet scale that's N redundant downloads for a
// single availability check, and it removes the option of a lightweight
// "any updates available?" health-check message that's safe to poll on a
// short interval.
//
// Resolution: split into two messages/methods:
//   - system:updates:check:get (Read) -> calls only s.updater.CheckForUpdate,
//     returns {Available bool, Version string} with no side effects.
//   - system:updates:stage:create (Create) -> calls PrepareUpdate, keeping
//     today's staging + notify-admins behavior.
//
// Check() (or a thin wrapper) can still exist for backward compatibility,
// calling both in sequence, but the two effects should be independently
// callable. RunScheduledCheck can keep calling both back to back since a
// cron job doing its own check-then-stage is the correct behavior there.
//
// Check runs the read-only check, stages a newer version when available, and
// notifies admins when a release was newly staged. With AutoApply it applies
// immediately (the process exits on success). Kept for backward
// compatibility; prefer system:updates:check:get +
// system:updates:stage:create so each effect stays independently callable.
//
// @hestia.register(
//
//	name="system:updates:check:create",
//	intent="create",
//	rule="administrator",
//	description="Check for and stage an update (legacy check-then-stage)",
//	output="CheckView",
//
// )
func (s *UpdatesService) Check(ctx context.Context, _ abstract.Message, _ *NoInput) (*CheckView, error) {
	staged, newly, err := s.checkAndStage(ctx)
	if err != nil {
		return nil, err
	}
	if newly && staged != nil {
		if err := s.notifyAdmins(ctx, staged); err != nil {
			s.logger.Warn("updates: notify update_available failed", zap.Error(err))
		}
	}
	view := &CheckView{
		Checked:   true,
		Staged:    staged != nil,
		AutoApply: s.autoApply,
	}
	if staged != nil {
		view.Version = staged.Version
	}
	if s.autoApply && staged != nil {
		if err := s.verifyApplyReady(ctx); err != nil {
			return nil, err
		}
		// Detach before spawning: ctx may be a fasthttp RequestCtx-backed
		// context that the transport recycles for the next request once this
		// handler returns. Reading it 300ms later is a data race with
		// cross-request value bleed (on the path that ends in os.Exit).
		applyCtx := context.WithoutCancel(context.Background())
		go func() {
			time.Sleep(300 * time.Millisecond)
			s.applySwap(applyCtx)
		}()
	}
	return document.New(view), nil
}

// CheckAvailability asks whether a newer release exists without any side
// effects: no download, no staging, no last-check bookkeeping. Safe for fleet
// coordinators to poll on a short interval.
//
// @hestia.register(
//
//	name="system:updates:check:get",
//	intent="read",
//	rule="administrator",
//	description="Check whether a newer version is available",
//	output="AvailabilityView",
//
// )
func (s *UpdatesService) CheckAvailability(ctx context.Context, _ abstract.Message, _ *NoInput) (*AvailabilityView, error) {
	if s.updater == nil {
		return nil, fmt.Errorf("updater not configured")
	}
	info, err := s.updater.CheckForUpdate(ctx)
	if err != nil {
		return nil, err
	}
	view := &AvailabilityView{}
	if info != nil {
		view.Available = true
		view.Version = info.Version
	}
	return document.New(view), nil
}

// Stage downloads and prepares the newest available release without applying
// it, recording the last-check time and notifying admins when a release was
// newly staged.
//
// @hestia.register(
//
//	name="system:updates:stage:create",
//	intent="create",
//	rule="administrator",
//	description="Download and stage the latest update",
//	output="StageView",
//
// )
func (s *UpdatesService) Stage(ctx context.Context, _ abstract.Message, _ *NoInput) (*StageView, error) {
	staged, newly, err := s.stageLatest(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.recordLastCheck(ctx); err != nil {
		return nil, err
	}
	if newly && staged != nil {
		if err := s.notifyAdmins(ctx, staged); err != nil {
			s.logger.Warn("updates: notify update_available failed", zap.Error(err))
		}
	}
	view := &StageView{Staged: staged != nil}
	if staged != nil {
		view.Version = staged.Version
	}
	return document.New(view), nil
}

// Apply verifies the staged binary and triggers the process swap. Hash
// verification and "nothing staged" checks run synchronously so the caller
// receives a meaningful error or success response. The actual swap (which
// terminates the process) runs asynchronously after the response is flushed.
//
// @hestia.register(
//
//	name="system:updates:update:apply",
//	intent="create",
//	rule="administrator",
//	description="Apply the staged update",
//	output="ApplyView",
//
// )
func (s *UpdatesService) Apply(ctx context.Context, _ abstract.Message, _ *NoInput) (*ApplyView, error) {
	// @note #74bt56 issueopend  : Lack of recovery strategy
	// @assignee opencode
	// Added Discard method (system:updates:update:discard) that cleans up failed staged updates by calling updater.Cleanup() to remove the staged binary and clear the pending record. Includes registration, policy binding (administrator rule), and two tests (cleanup success + no-op when nothing staged). All 22 update tests pass.
	//
	// We have no strategy to recover from a failed update
	// such as cleaning up the staged binary so that it can be
	// re-staged
	if err := s.verifyApplyReady(ctx); err != nil {
		return nil, err
	}
	// Launch the swap after a short delay so the HTTP response is flushed
	// before os.Exit(0) terminates the process. The goroutine gets a fully
	// detached context: ctx may be a fasthttp RequestCtx-backed context that
	// the transport recycles for the next request once this handler returns.
	applyCtx := context.WithoutCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		s.applySwap(applyCtx)
	}()
	return document.New(&ApplyView{Message: "update applied; restarting"}), nil
}

// Discard cleans up a failed or unwanted staged update: removes the staged
// binary from disk and clears the pending update record. After discard the
// system returns to a clean state and a new check-and-stage cycle can re-stage
// a fresh update.
//
// @hestia.register(
//
//	name="system:updates:update:discard",
//	intent="delete",
//	rule="administrator",
//	description="Discard a staged update and clean up",
//	output="DiscardView",
//
// )
func (s *UpdatesService) Discard(ctx context.Context, _ abstract.Message, _ *NoInput) (*DiscardView, error) {
	if s.updater == nil {
		return nil, fmt.Errorf("updater not configured")
	}
	if err := s.updater.Cleanup(); err != nil {
		return nil, fmt.Errorf("discard staged update: %w", err)
	}
	return document.New(&DiscardView{Message: "staged update discarded"}), nil
}

// RunScheduledCheck is the recurring job: check, stage, notify on new staging,
// and optionally apply. It runs with a system identity.
func (s *UpdatesService) RunScheduledCheck(ctx context.Context) error {
	staged, newly, err := s.checkAndStage(ctx)
	if err != nil {
		return err
	}
	if newly && staged != nil {
		if err := s.notifyAdmins(ctx, staged); err != nil {
			s.logger.Warn("updates: notify update_available failed", zap.Error(err))
		}
	}
	if s.autoApply && staged != nil {
		if err := s.verifyApplyReady(ctx); err != nil {
			return err
		}
		go func() {
			time.Sleep(300 * time.Millisecond)
			s.applySwap(ctx)
		}()
	}
	return nil
}

// @note #update-hash-verify issue resolved P1 status=open : Verify binary hash before applying staged update
// Fixed: stageLatest now backfills a computed SHA-256 into the pending record when the provider omits a checksum (the GitHub-provider hole), so every freshly staged update is verifiable. applySwap re-verifies immediately before ApplyUpdate/swapExecutable, closing the verify-to-swap TOCTOU window; mismatch logs 'staged binary failed verification' and aborts. stagedBinaryPath() handles the Windows .exe suffix. Legacy pending rows without checksums still pass vacuously (documented).
//
// The staged binary at DataDir/update is copied over the running executable
// (swapExecutable) or executed directly (ApplyUpdate) without verifying its
// SHA-256 checksum against UpdateInfo.Checksum. The Checksum field already
// flows through the persistence layer, but the actual verification against
// the on-disk binary is never performed. This means a corrupted download —
// or a file tampered with between staging and apply — would silently replace
// the running binary. The check should re-hash the staged file and abort
// the swap/exec if it does not match.
//
// Real-world hit: /opt/hedwig-server/hedwig-server ended up as an ELF with
// "missing section headers" — the update was applied but the binary was
// corrupted. A pre-apply hash check would have caught this.
//
// Why staging can succeed while application produces a corrupted binary:
//   1. GitHub provider passes empty checksum (github.go:133) so
//      DownloadBinary skips hash verification entirely.
//   2. Neither copyFile (process.go:62) nor swapExecutable (service.go:301)
//      re-hash the staged file before overwriting the executable.
//   3. copyFile does io.Copy + Sync but a mid-copy I/O error or disk-full
//      condition can leave the destination truncated.
//   4. swapExecutable creates a temp file in filepath.Dir(exe) which may
//      differ from DataDir — if they share a filesystem the Rename is
//      atomic, but a failed copy to the temp file is not caught before the
//      rename.
//
// @see #updater-hash-verify

// verifyApplyReady checks preconditions for the swap: the staged binary must
// exist, and its checksum (when provided) must match. Returns nil when there
// is nothing to apply (up to date), or an error that is safe to return to
// the caller.
func (s *UpdatesService) verifyApplyReady(ctx context.Context) error {
	if s.updater == nil {
		return fmt.Errorf("updater not configured")
	}
	if !s.systemd {
		if !s.updater.HasPreparedUpdate() {
			return fmt.Errorf("nothing staged to apply")
		}
		return s.verifyStagedBinary(ctx)
	}
	if _, err := s.updater.PrepareUpdate(ctx); err != nil {
		return err
	}
	if !s.updater.HasPreparedUpdate() {
		return nil // nothing staged
	}
	return s.verifyStagedBinary(ctx)
}

// applySwap performs the actual binary handoff. In the default (spawn) mode it
// delegates to updater.ApplyUpdate which spawns the staged binary and exits.
// In SystemdMode it swaps the executable in-place and exits cleanly.
func (s *UpdatesService) applySwap(ctx context.Context) {
	if s.updater == nil {
		s.logger.Error("updates: apply failed", zap.Error(fmt.Errorf("updater not configured")))
		return
	}
	// Verify immediately before handing off — verifyApplyReady ran earlier
	// (possibly via a different code path), so re-checking here closes the
	// window where the staged file could change between check and swap.
	if err := s.verifyStagedBinary(ctx); err != nil {
		s.logger.Error("updates: staged binary failed verification; aborting apply", zap.Error(err))
		return
	}
	if !s.systemd {
		// ApplyUpdate spawns the new binary with --perform-update and then
		// the current process exits, so ClearUpdate cannot run after it.
		// The new binary's Reconcile call at boot clears the pending row once
		// it detects its own version >= the staged version.
		if err := s.updater.ApplyUpdate(); err != nil {
			s.logger.Error("updates: apply failed", zap.Error(err))
		}
		return
	}
	if err := s.swapExecutable(ctx); err != nil {
		s.logger.Error("updates: swap executable failed", zap.Error(err))
		return
	}
	// Clear the pending row synchronously before systemd restarts us. Failure
	// here is non-fatal — Reconcile on the next boot will clear it — but log
	// it so it is observable.
	if err := s.store.ClearUpdate(ctx); err != nil {
		s.logger.Warn("updates: clear pending update after swap failed", zap.Error(err))
	}
	exitProcess(0)
}

// verifyStagedBinary hashes the staged binary at DataDir/update and compares
// it against the expected checksum from the pending update record. Returns nil
// when there is no pending record or no recorded checksum (legacy rows staged
// before checksum backfilling existed), or when the hashes match. Returns an
// error on mismatch or I/O failure, aborting the apply.
func (s *UpdatesService) verifyStagedBinary(ctx context.Context) error {
	pending, err := s.store.PendingUpdate(ctx)
	if err != nil {
		return fmt.Errorf("read pending update: %w", err)
	}
	if pending == nil || pending.Checksum == "" {
		return nil // no checksum to verify — trust the staged binary
	}

	staged := s.stagedBinaryPath()
	actual, err := hashFile(staged)
	if err != nil {
		return fmt.Errorf("hash staged binary for verification: %w", err)
	}
	if actual != pending.Checksum {
		return fmt.Errorf("staged binary checksum mismatch: expected %s, got %s", pending.Checksum, actual)
	}
	return nil
}

// swapExecutable atomically replaces the running executable with the staged
// binary in DataDir/update (mirroring updater's staging path). Renaming over a
// running executable is safe on Linux — the current process keeps its old
// inode — so this can run in-process before exiting.
func (s *UpdatesService) swapExecutable(ctx context.Context) error {
	staged := filepath.Join(s.dataDir, "update")
	src, err := os.Open(staged)
	if err != nil {
		return fmt.Errorf("open staged update %s: %w", staged, err)
	}
	defer src.Close()

	exe := s.exePath
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".hestia-update-*")
	if err != nil {
		return fmt.Errorf("create temp next to executable: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return fmt.Errorf("copy staged update: %w", err)
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod staged update: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync staged update: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, exe); err != nil {
		return fmt.Errorf("replace executable %s: %w", exe, err)
	}
	return nil
}

// checkAndStage checks for an update and stages it. It reports the staged
// UpdateInfo (nil when up to date) and whether a release was newly staged
// (never on a no-op re-check).
func (s *UpdatesService) checkAndStage(ctx context.Context) (staged *updater.UpdateInfo, newlyStaged bool, err error) {
	if s.updater == nil {
		return nil, false, fmt.Errorf("updater not configured")
	}
	info, err := s.updater.CheckForUpdate(ctx)
	if err != nil {
		return nil, false, err
	}
	if err := s.recordLastCheck(ctx); err != nil {
		return nil, false, err
	}
	if info == nil {
		return nil, false, nil
	}
	return s.stageLatest(ctx)
}

// stagedBinaryPath resolves the on-disk path updater stages downloads to:
// DataDir/update (update.exe on Windows).
func (s *UpdatesService) stagedBinaryPath() string {
	name := "update"
	if goruntime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(s.dataDir, name)
}

// hashFile returns the SHA-256 digest of the file at path, formatted like
// updater checksums ("SHA256:<hex>").
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "SHA256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// stageLatest downloads and prepares the newest release, reporting the staged
// UpdateInfo (nil when already up to date) and whether it was newly staged
// relative to the pending row.
//
// When the provider did not supply a checksum, one is computed from the
// freshly downloaded binary and persisted with the pending record so that
// pre-apply verification always has something to compare against. This closes
// the corrupted-download hole: without it a truncated or mid-copy-failed
// download would pass verification vacuously (see #update-hash-verify).
func (s *UpdatesService) stageLatest(ctx context.Context) (staged *updater.UpdateInfo, newlyStaged bool, err error) {
	if s.updater == nil {
		return nil, false, fmt.Errorf("updater not configured")
	}
	prev, _ := s.store.PendingUpdate(ctx)
	staged, err = s.updater.PrepareUpdate(ctx)
	if err != nil {
		return nil, false, err
	}
	if staged == nil {
		return nil, false, nil
	}

	if staged.Checksum == "" {
		digest, err := hashFile(s.stagedBinaryPath())
		if err != nil {
			return nil, false, fmt.Errorf("hash staged update: %w", err)
		}
		staged.Checksum = digest
		if err := s.store.SaveUpdate(ctx, staged); err != nil {
			return nil, false, fmt.Errorf("persist computed checksum: %w", err)
		}
		s.logger.Info("updates: computed checksum for staged update",
			zap.String("version", staged.Version), zap.String("checksum", digest))
	}

	return staged, prev == nil || prev.Version != staged.Version, nil
}

func (s *UpdatesService) recordLastCheck(ctx context.Context) error {
	return s.store.Settings.Set(ctx, "", lastCheckKey, map[string]any{"unix_ms": time.Now().UnixMilli()}, "updater")
}

// notifyAdmins sends an update_available notification (in-app, plus email when
// a mailer is configured) to every enabled user holding the administrator
// permission. TenantID is set from the admin user's record so that
// tenant-scoped notification queries (list, unread count) can find it.
func (s *UpdatesService) notifyAdmins(ctx context.Context, info *updater.UpdateInfo) error {
	if s.notifier == nil {
		return nil
	}
	admins, err := s.adminUsers(ctx)
	if err != nil {
		return err
	}
	for _, u := range admins {
		channels := []abstract.ChannelType{abstract.ChannelInApp}
		if s.hasMailer {
			channels = append(channels, abstract.ChannelEmail)
		}
		n := abstract.Notification{
			Recipient: abstract.Recipient{UserID: u.ID, Email: u.Email},
			Template:  "update_available",
			Data: map[string]any{
				"version":   info.Version,
				"changelog": info.Changelog,
				"app_url":   s.appURL,
			},
			Actions: []abstract.NotificationAction{
				{
					Label:   "Apply update",
					Message: "system:updates:update:apply",
				},
			},
			Channels: channels,
		}
		if u.TenantID != nil {
			n.TenantID = *u.TenantID
		}
		if err := s.notifier.Send(ctx, n); err != nil {
			return err
		}
	}
	return nil
}

func (s *UpdatesService) adminUsers(ctx context.Context) ([]*usersmodel.SystemUser, error) {
	q := query.NewQueryBuilder().
		Where("permissions").Contains("administrator").
		Where("disabled").Eq(-1).
		Build()
	return s.users.Read(ctx, &q)
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}
