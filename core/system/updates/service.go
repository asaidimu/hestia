package updates

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"
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

func NewService(u *updater.Updater, store *Store, notifier abstract.Notifier, users *usersmodel.SystemUsers, logger *zap.Logger, appURL string, hasMailer, autoApply bool, version string, systemd bool, exePath, dataDir string) *UpdatesService {
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
func (s *UpdatesService) Status(ctx context.Context, _ abstract.Message, _ *NoInput) (*StatusView, error) {
	view := &StatusView{
		Version:  s.version,
		Prepared: s.updater.HasPreparedUpdate(),
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

// @note #review2-20260821-004 issue P1 #review,#api-design,#p2p : system:updates:check:create conflates checking with staging (and a fleet cannot poll safely)
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
// Check() (or a thin wrapper) can still exist for backward compatibility,
// calling both in sequence, but the two effects should be independently
// callable. RunScheduledCheck can keep calling both back to back since a
// cron job doing its own check-then-stage is the correct behavior there.
//
// Check runs the read-only check, stages a newer version when available, and
// notifies admins when a release was newly staged. With AutoApply it applies
// immediately (the process exits on success).
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
		if err := s.apply(ctx); err != nil {
			return nil, err
		}
	}
	return document.New(view), nil
}

// Apply performs the maintenance-window apply. On success the process exits
// and the new binary takes over; on failure (e.g. nothing staged) it returns
// the error.
func (s *UpdatesService) Apply(ctx context.Context, _ abstract.Message, _ *NoInput) (*ApplyView, error) {
	if err := s.apply(ctx); err != nil {
		return nil, err
	}
	return document.New(&ApplyView{Message: "update applied; restarting"}), nil
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
		return s.apply(ctx)
	}
	return nil
}

// apply performs the update handoff. In the default (spawn) mode it delegates
// to updater.ApplyUpdate, which spawns the staged binary with --perform-update
// and never returns. In SystemdMode it swaps the staged binary over the
// running executable in place and exits cleanly, letting systemd restart the
// new binary as a tracked process. Returns nil when there is nothing to apply.
func (s *UpdatesService) apply(ctx context.Context) error {
	if !s.systemd {
		return s.updater.ApplyUpdate()
	}
	if _, err := s.updater.PrepareUpdate(ctx); err != nil {
		return err
	}
	if !s.updater.HasPreparedUpdate() {
		return nil // up to date; nothing staged to apply
	}
	if err := s.swapExecutable(ctx); err != nil {
		return err
	}
	// Give the HTTP layer a moment to flush the success response before the
	// process exits; the unit restarts the swapped binary.
	go func() {
		time.Sleep(300 * time.Millisecond)
		exitProcess(0)
	}()
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
	prev, _ := s.store.PendingUpdate(ctx)
	staged, err = s.updater.PrepareUpdate(ctx)
	if err != nil {
		return nil, false, err
	}
	if staged == nil {
		return nil, false, nil
	}
	return staged, prev == nil || prev.Version != staged.Version, nil
}

func (s *UpdatesService) recordLastCheck(ctx context.Context) error {
	return s.store.Settings.Set(ctx, "", lastCheckKey, map[string]any{"unix_ms": time.Now().UnixMilli()}, "updater")
}

// notifyAdmins sends an update_available notification (in-app, plus email when
// a mailer is configured) to every enabled user holding the administrator
// permission.
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
		if err := s.notifier.Send(ctx, abstract.Notification{
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
		}); err != nil {
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