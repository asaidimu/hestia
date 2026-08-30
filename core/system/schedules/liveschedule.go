package schedules

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"github.com/asaidimu/hestia/core/runtime/templateutil"
	"github.com/asaidimu/hestia/core/system/schedules/model"
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
	model *model.SystemScheduledMessagess
	sched *scheduler.Scheduler
	disp  abstract.Dispatcher
	// S-1: optional late-bound chain and authorizer (nil in unit tests).
	dispFn func() abstract.Dispatcher
	auth   *ScheduleAuthorizer
	log    *zap.Logger
}

func NewLiveSchedule(model *model.SystemScheduledMessagess, sched *scheduler.Scheduler, disp abstract.Dispatcher, log *zap.Logger) *LiveSchedule {
	return &LiveSchedule{
		model: model,
		sched: sched,
		disp:  disp,
		log:   log,
	}
}

// WithAuthorizer wires the S-1 authorization model: create-time target
// checks plus fire-time resolution of the creator's current claims. Without
// it, fires degrade to a scope-less creator identity (unit-test behavior).
func (ls *LiveSchedule) WithAuthorizer(a *ScheduleAuthorizer) *LiveSchedule {
	ls.auth = a
	return ls
}

// WithDispatcherProvider wires a late-binding dispatcher accessor. The full
// middleware chain (sanitization → bootstrap → secure → ratelimit → throttle
// → tenant → blob → audit) is built after boot; schedule fires must traverse
// it so authorization, rate limiting and audit apply. Falls back to the raw
// dispatcher when the chain has not been built yet.
func (ls *LiveSchedule) WithDispatcherProvider(fn func() abstract.Dispatcher) *LiveSchedule {
	ls.dispFn = fn
	return ls
}

func (ls *LiveSchedule) dispatcher() abstract.Dispatcher {
	if ls.dispFn != nil {
		if d := ls.dispFn(); d != nil {
			return d
		}
	}
	return ls.disp
}

// fireClaims resolves the fire-time identity: the creator's current claims
// when the authorizer is wired, a restricted scope-less creator identity
// otherwise. Nil means fail closed — skip the fire.
func (ls *LiveSchedule) fireClaims(creatorID, storedTenantID string) *abstract.Claims {
	if ls.auth != nil {
		return ls.auth.FireClaims(creatorID, storedTenantID)
	}
	if creatorID == "" {
		return nil
	}
	return &abstract.Claims{UserID: creatorID, TenantID: storedTenantID}
}

func (ls *LiveSchedule) Init(ctx context.Context) error {
	docs, err := ls.model.ListSchedules(ctx)
	if err != nil {
		return common.SystemErrorFrom(err).WithOperation("LiveSchedule.Init").WithMessage("list schedules failed")
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
		ls.log.Warn("schedule: skipping dispatch — no message on schedule document", zap.String("schedule_id", doc.ID()))
		return nil
	}

	var inputMap map[string]any
	if raw, err := doc.Get("input"); err == nil && raw != nil {
		if m, ok := raw.(map[string]any); ok {
			inputMap = m
		}
	}

	resolvedInput := templateutil.ResolveMap(inputMap, scheduleTemplateData(doc))

	// Handler DTOs bind via input:"payload.*" tags — internal dispatches must
	// wrap the schedule's input map in the same envelope the HTTP layer builds.
	docInput, err := data.NewDocument(map[string]any{"payload": resolvedInput}, ctx)
	if err != nil {
		ls.log.Error("schedule: build input document failed",
			zap.String("schedule_id", doc.ID()), zap.String("message", message), zap.Error(err))
		return err
	}

	// S-1: fires run as the creator's CURRENT identity — never as SYSTEM.
	// The previous path (SystemContext + raw terminal dispatcher) bypassed
	// the entire policy chain: any authenticated user could schedule any
	// operation (user creation, policy edits, settings writes) and it
	// executed with system privileges, zero authorization and zero audit.
	creatorID, _ := doc.GetString("user_id")
	tenantID, _ := doc.GetString("tenant_id")
	claims := ls.fireClaims(creatorID, tenantID)
	if claims == nil {
		ls.log.Warn("schedule: dispatch skipped — creator is missing or no longer active",
			zap.String("schedule_id", doc.ID()), zap.String("message", message), zap.String("creator", creatorID))
		return nil
	}

	fireCtx := runtimecontext.ContextWithClaims(ctx, claims)
	ls.log.Info("schedule: dispatching", zap.String("schedule_id", doc.ID()), zap.String("message", message), zap.String("creator", creatorID))
	msg := dispatch.NewMessage(message, fireCtx, docInput)
	result, err := dispatch.Await(fireCtx, ls.dispatcher(), msg)
	if err != nil {
		// The cron chain only logs panics — returned errors would otherwise
		// vanish, so every failure is logged here with the offending schedule.
		ls.log.Error("schedule: dispatched operation failed",
			zap.String("schedule_id", doc.ID()), zap.String("message", message), zap.Error(err))
	}
	if result != nil {
		result.Release()
	}
	return err
}
// @note #scheduled-dispatch-errors-are-sw-616004cd lesson resolved P3 #schedules,#dispatch : Scheduled dispatch errors are swallowed by the cron chain
// Fixed in liveschedule.go dispatch(): every failure path now logs via ls.log — missing message (Warn), input-document build failure (Error), and dispatched-operation failure with schedule_id + message + error (Error). Result document released. Failures are now visible without instrumenting code.
//
// LiveSchedule.dispatch returns errors from dispatch.Await, but robfig/cron only logs panics (Recover wrapper) — returned errors vanish silently. A broken schedule (e.g. unregistered message, DTO binding failure) ticks forever with no trace. While fixing the payload-envelope bug (input must be wrapped in {"payload": ...} to satisfy input:"payload.*" DTO bindings, same as core/interface/cli/orchestrator.go does) this cost significant debugging time on the live server. Consider logging dispatch failures via ls.log in dispatch(), or a cron.PrintfLogger chain.
