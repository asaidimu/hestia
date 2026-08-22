package notifications

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/notifications/model"
)

// TestCreateNotificationGoroutineStability pins a regression guard for the
// go-anansi transaction-watcher leak (#tx-watcher-goroutine-leak): every
// top-level write used to park one goroutine forever on ctx.Done() for
// uncancelable contexts. Fixed in go-anansi v8.5.10.
func TestCreateNotificationGoroutineStability(t *testing.T) {
	p := testutil.NewPersistence(t)
	m, err := model.InitSystemNotificationssModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("init model: %v", err)
	}
	svc := &NotificationsService{model: m}
	ctx := context.Background()

	runtime.GC()
	before := runtime.NumGoroutine()
	for i := 0; i < 100; i++ {
		if _, err := svc.CreateNotification(ctx, nil, &model.NotificationCreateInput{
			NotificationCreate: model.NotificationCreate{
				UserID:  "goroutine-stability",
				Subject: fmt.Sprintf("note %d", i),
			},
		}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	runtime.GC()
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutines before=%d after=%d: write path leaks goroutines", before, after)
	}
}
