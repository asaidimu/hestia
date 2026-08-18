package updates

import (
	"context"
	"testing"

	"github.com/asaidimu/updater"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/settings"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	p := testutil.NewPersistence(t)
	return NewStore(settings.NewSettingsModel(p))
}

func TestStorePendingUpdateRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	pending, err := s.PendingUpdate(ctx)
	if err != nil {
		t.Fatalf("pending on empty store: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected nil pending, got %+v", pending)
	}

	info := &updater.UpdateInfo{
		Version:   "1.2.0",
		URL:       "https://example.com/app",
		Changelog: "release notes",
		AssetName: "app_1.2.0",
		Checksum:  "abc123",
	}
	if err := s.SaveUpdate(ctx, info); err != nil {
		t.Fatalf("save: %v", err)
	}

	pending, err = s.PendingUpdate(ctx)
	if err != nil {
		t.Fatalf("pending after save: %v", err)
	}
	if pending == nil || pending.Version != info.Version || pending.URL != info.URL ||
		pending.Changelog != info.Changelog || pending.AssetName != info.AssetName || pending.Checksum != info.Checksum {
		t.Fatalf("round trip mismatch: %+v", pending)
	}

	if err := s.ClearUpdate(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	pending, err = s.PendingUpdate(ctx)
	if err != nil {
		t.Fatalf("pending after clear: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected nil pending after clear, got %+v", pending)
	}
}

func TestStoreSaveNilInfo(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	if err := s.SaveUpdate(ctx, nil); err != nil {
		t.Fatalf("save nil: %v", err)
	}
	pending, err := s.PendingUpdate(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected nil pending, got %+v", pending)
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.SaveUpdate(ctx, &updater.UpdateInfo{Version: "1.1.0"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Pending 1.1.0 is older than running 1.2.0 -> cleared.
	if err := s.Reconcile(ctx, "1.2.0"); err != nil {
		t.Fatalf("reconcile older: %v", err)
	}
	if pending, _ := s.PendingUpdate(ctx); pending != nil {
		t.Fatalf("older pending should be cleared, got %+v", pending)
	}

	if err := s.SaveUpdate(ctx, &updater.UpdateInfo{Version: "1.3.0"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Pending 1.3.0 is newer than running 1.2.0 -> kept.
	if err := s.Reconcile(ctx, "1.2.0"); err != nil {
		t.Fatalf("reconcile newer: %v", err)
	}
	if pending, _ := s.PendingUpdate(ctx); pending == nil || pending.Version != "1.3.0" {
		t.Fatalf("newer pending should be kept, got %+v", pending)
	}
}

func TestReconcileInvalidSemverKeepsRow(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.SaveUpdate(ctx, &updater.UpdateInfo{Version: "not-a-version"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Reconcile(ctx, "1.2.0"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if pending, _ := s.PendingUpdate(ctx); pending == nil || pending.Version != "not-a-version" {
		t.Fatalf("invalid version row should be kept, got %+v", pending)
	}
}

func TestIsNewerThan(t *testing.T) {
	if ok, newer := isNewerThan("1.0.0", "1.1.0"); !ok || !newer {
		t.Fatalf("expected newer, got ok=%v newer=%v", ok, newer)
	}
	if ok, newer := isNewerThan("1.1.0", "1.0.0"); !ok || newer {
		t.Fatalf("expected not newer, got ok=%v newer=%v", ok, newer)
	}
	if ok, newer := isNewerThan("1.1.0", "1.1.0"); !ok || newer {
		t.Fatalf("expected equal not newer, got ok=%v newer=%v", ok, newer)
	}
	if ok, _ := isNewerThan("1.1.0", "not-a-version"); ok {
		t.Fatalf("expected not ok for invalid version")
	}
	if ok, _ := isNewerThan("not-a-version", "1.1.0"); ok {
		t.Fatalf("expected not ok for invalid current")
	}
}