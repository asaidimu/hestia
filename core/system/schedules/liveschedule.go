package schedules

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"github.com/asaidimu/hestia/core/runtime/templateutil"
)

func scheduleTemplateData(doc data.Documenter) map[string]any {
	m := map[string]any{"_id": doc.ID(), "now": time.Now()}
	if v, _ := doc.GetString("message"); v != "" {
		m["message"] = v
	}
	if v, _ := doc.GetString("cron"); v != "" {
		m["cron"] = v
	}
	if v, _ := doc.GetString("user_id"); v != "" {
		m["user_id"] = v
	}
	if v, _ := doc.GetString("tenant_id"); v != "" {
		m["tenant_id"] = v
	}
	if v, _ := doc.GetInt("created_at"); v != 0 {
		m["created_at"] = v
	}
	return m
}

type LiveSchedule struct {
	model *ScheduleModel
	sched *scheduler.Scheduler
	disp  abstract.Dispatcher
	log   *zap.Logger
}

func NewLiveSchedule(model *ScheduleModel, sched *scheduler.Scheduler, disp abstract.Dispatcher, log *zap.Logger) *LiveSchedule {
	return &LiveSchedule{
		model: model,
		sched: sched,
		disp:  disp,
		log:   log,
	}
}

func (ls *LiveSchedule) Init(ctx context.Context) error {
	docs, err := ls.model.List(ctx)
	if err != nil {
		return fmt.Errorf("list schedules: %w", err)
	}
	for _, doc := range docs {
		ls.register(ctx, doc)
	}
	ls.log.Info("live schedule: initialized", zap.Int("count", len(docs)))
	return nil
}

func (ls *LiveSchedule) Register(ctx context.Context, doc data.Documenter) {
	ls.register(ctx, doc)
}

func (ls *LiveSchedule) UnregisterByID(_ context.Context, id string) {
	ls.sched.Remove("schedule:" + id)
}

func (ls *LiveSchedule) ReRegister(ctx context.Context, doc data.Documenter) {
	ls.UnregisterByID(ctx, doc.ID())
	ls.register(ctx, doc)
}

func (ls *LiveSchedule) register(ctx context.Context, doc data.Documenter) {
	disabled, _ := doc.Get("disabled")
	if disabled == true {
		return
	}
	cronExpr, err := doc.GetString("cron")
	if err != nil || cronExpr == "" {
		return
	}
	if doc.ID() == "" {
		return
	}

	docCopy := doc.Clone()
	ls.sched.Register("schedule:"+doc.ID(), cronExpr, func(ctx context.Context) error {
		return ls.dispatch(ctx, docCopy)
	})
}

func (ls *LiveSchedule) dispatch(ctx context.Context, doc data.Documenter) error {
	message, err := doc.GetString("message")
	if err != nil || message == "" {
		return nil
	}

	var inputMap map[string]any
	if raw, err := doc.Get("input"); err == nil && raw != nil {
		if m, ok := raw.(map[string]any); ok {
			inputMap = m
		}
	}

	resolvedInput := templateutil.ResolveMap(inputMap, scheduleTemplateData(doc))

	docInput, err := data.NewDocument(resolvedInput, ctx)
	if err != nil {
		return err
	}

	sysCtx := runtimecontext.SystemContext(ctx)
	msg := dispatch.NewMessage(message, sysCtx, docInput)
	_, err = dispatch.Await(sysCtx, ls.disp, msg)
	return err
}
