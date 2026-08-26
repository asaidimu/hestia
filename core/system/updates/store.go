package updates

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Masterminds/semver/v3"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/updater"

	"github.com/asaidimu/hestia/core/system/settings/model"
)

const pendingKey = "updates:pending"

// pendingUpdate mirrors updater.UpdateInfo with JSON tags so the persisted row
// stays readable in the settings collection (updater.UpdateInfo has no json
// tags).
type pendingUpdate struct {
	Version   string `json:"version"`
	URL       string `json:"url,omitempty"`
	Changelog string `json:"changelog,omitempty"`
	AssetName string `json:"asset_name,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
}

// Store persists the single pending update row over the settings collection
// (key updates:pending, tenant ""), satisfying updater.Store. The settings
// `value` field is a record, so each value is stored as a JSON-shaped object.
type Store struct {
	Settings *model.SystemSettingss
}

func NewStore(settingsModel *model.SystemSettingss) *Store {
	return &Store{Settings: settingsModel}
}

// InitStore creates a Store from the DI-resolved persistence layer.
func InitStore(persist base.Persistence) *Store {
	model.DangerouslyResetSystemSettingssModel()
	m, err := model.InitSystemSettingssModel(persist, nil)
	if err != nil {
		panic("init system settings model: " + err.Error())
	}
	return NewStore(m)
}

func (s *Store) SaveUpdate(ctx context.Context, info *updater.UpdateInfo) error {
	if info == nil {
		return nil
	}
	raw, err := json.Marshal(toPendingUpdate(info))
	if err != nil {
		return err
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		return err
	}
	return s.Settings.Set(ctx, "", pendingKey, rec, "updater")
}

func (s *Store) PendingUpdate(ctx context.Context) (*updater.UpdateInfo, error) {
	v, err := s.Settings.Get(ctx, "", pendingKey)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var p pendingUpdate
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Version == "" {
		return nil, nil
	}
	return toUpdateInfo(&p), nil
}

func (s *Store) ClearUpdate(ctx context.Context) error {
	return s.Settings.Unset(ctx, "", pendingKey)
}

// Reconcile drops a pending update that is obsolete relative to the running
// version — e.g. the row left behind by a pre-boot swap that ran before the
// DB was available. When either version is not valid semver, the row is left
// untouched. Call once at boot.
func (s *Store) Reconcile(ctx context.Context, currentVersion string) error {
	pending, err := s.PendingUpdate(ctx)
	if err != nil || pending == nil {
		return err
	}
	if ok, newer := isNewerThan(currentVersion, pending.Version); ok && !newer {
		return s.ClearUpdate(ctx)
	}
	return nil
}

// isNewerThan reports whether latest is semantically greater than current.
// ok is false when either version fails to parse.
func isNewerThan(current, latest string) (ok, newer bool) {
	cur, err := semver.NewVersion(current)
	if err != nil {
		return false, false
	}
	lat, err := semver.NewVersion(latest)
	if err != nil {
		return false, false
	}
	return true, lat.GreaterThan(cur)
}

func toPendingUpdate(info *updater.UpdateInfo) *pendingUpdate {
	if info == nil {
		return nil
	}
	return &pendingUpdate{
		Version:   info.Version,
		URL:       info.URL,
		Changelog: info.Changelog,
		AssetName: info.AssetName,
		Checksum:  info.Checksum,
	}
}

func toUpdateInfo(p *pendingUpdate) *updater.UpdateInfo {
	if p == nil {
		return nil
	}
	return &updater.UpdateInfo{
		Version:   p.Version,
		URL:       p.URL,
		Changelog: p.Changelog,
		AssetName: p.AssetName,
		Checksum:  p.Checksum,
	}
}