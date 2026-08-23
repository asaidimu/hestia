package runtime

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
	"github.com/asaidimu/hestia/core/runtime/templateutil"
)

type ThrottleTemplateData struct {
	Claims    map[string]any
	Input     map[string]any
	SourceIP  string
	Operation string
	Timestamp time.Time
}

func (d ThrottleTemplateData) toMap() map[string]any {
	return map[string]any{
		"claims":    d.Claims,
		"input":     d.Input,
		"sourceIP":  d.SourceIP,
		"operation": d.Operation,
		"timestamp": d.Timestamp,
	}
}

func buildThrottleTemplateData(ctx context.Context, msg abstract.Message) ThrottleTemplateData {
	data := ThrottleTemplateData{
		SourceIP:  msg.SourceIP(),
		Operation: msg.Name(),
		Timestamp: time.Now(),
	}

	if ident, ok := iam.GetIdentity(ctx); ok {
		claims := make(map[string]any)
		if props, ok := ident.Properties.(map[string]any); ok {
			for k, v := range props {
				claims[k] = v
			}
		}
		data.Claims = claims
	}

	if input := msg.Input(); input != nil {
		data.Input = input.Data()
	}

	return data
}

type ThrottleLookup func(operation string) *ThrottlePolicy

type ThrottleDispatcher struct {
	next   abstract.Dispatcher
	lookup ThrottleLookup
	disp   abstract.Dispatcher
	store  RateLimitStore
	logger *zap.Logger
}

func NewThrottleDispatcher(lookup ThrottleLookup, disp abstract.Dispatcher, logger *zap.Logger) *ThrottleDispatcher {
	if lookup == nil {
		lookup = func(string) *ThrottlePolicy { return nil }
	}
	return &ThrottleDispatcher{
		lookup: lookup,
		disp:   disp,
		store:  ratestore.New(),
		logger: logger,
	}
}

func (d *ThrottleDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &ThrottleDispatcher{
		next:   next,
		lookup: d.lookup,
		disp:   d.disp,
		store:  d.store,
		logger: d.logger,
	}
}

func (d *ThrottleDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	op := msg.Name()
	throttle := d.lookup(op)
	if throttle == nil || throttle.Limit <= 0 {
		return d.next.Send(ctx, msg, onComplete)
	}

	key := "throttle:" + op
	count, err := d.store.Increment(msg.Context(), key, time.Duration(throttle.Window)*time.Second)
	if err != nil {
		return d.next.Send(ctx, msg, onComplete)
	}
	if int64(count) > throttle.Limit && throttle.Action != nil {
		if err := d.dispatchAction(ctx, msg, throttle.Action); err != nil {
			d.logger.Warn("throttle action failed",
				zap.String("operation", op),
				zap.String("action", throttle.Action.Message),
				zap.Error(err),
			)
		}
	}

	return d.next.Send(ctx, msg, onComplete)
}

// dispatchAction fires the configured throttle action as fire-and-forget:
// the action message is accepted for asynchronous execution and its outcome
// is discarded.
func (d *ThrottleDispatcher) dispatchAction(ctx context.Context, originalMsg abstract.Message, action *ThrottleActionPolicy) error {
	if action.Message == "" {
		return nil
	}

	tplData := buildThrottleTemplateData(originalMsg.Context(), originalMsg)
	resolvedInput := templateutil.ResolveMap(action.Input, tplData.toMap())

	doc, err := data.NewDocument(resolvedInput, originalMsg.Context())
	if err != nil {
		return err
	}

	sysCtx := runtimecontext.SystemContext(originalMsg.Context())
	actionMsg := dispatch.NewMessage(action.Message, sysCtx, doc)
	return d.disp.Send(sysCtx, actionMsg, nil)
}

var _ abstract.Dispatcher = (*ThrottleDispatcher)(nil)
var _ abstract.DispatcherLink = (*ThrottleDispatcher)(nil)
