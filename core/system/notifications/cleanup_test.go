package notifications_test

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/notifications/model"
	"go.uber.org/zap"
)

func TestCleanupExpired(t *testing.T) {
	// Reset the notifications model singleton
	model.DangerouslyResetSystemNotificationssModel()

	p := testutil.NewPersistence(t)
	m, err := model.InitSystemNotificationssModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("init model: %v", err)
	}

	ctx := context.Background()

	// Create an expired notification (expires_at in the past)
	past := time.Now().Add(-1 * time.Hour).UnixMilli()
	_, err = m.Create(ctx, &model.SystemNotifications{
		UserID:    "user-1",
		Type:      "test",
		Subject:   "expired notification",
		ExpiresAt: &past,
	})
	if err != nil {
		t.Fatalf("create expired notification: %v", err)
	}

	// Create a non-expired notification (no expires_at)
	_, err = m.Create(ctx, &model.SystemNotifications{
		UserID:    "user-1",
		Type:      "test",
		Subject:   "active notification",
	})
	if err != nil {
		t.Fatalf("create active notification: %v", err)
	}

	// Verify both exist (List filters expired, so we need to check differently)
	docs, err := m.List(ctx, "user-1", 100, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// List filters expired, so we should only see the active one
	if len(docs) != 1 {
		t.Fatalf("expected 1 active notification, got %d", len(docs))
	}

	// Delete expired
	deleted, err := m.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify only active remains
	docs, err = m.List(ctx, "user-1", 100, 0)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 notification after delete, got %d", len(docs))
	}
}
