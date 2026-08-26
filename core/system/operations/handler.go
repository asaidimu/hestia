// @note #cruft-20260821-017 observation resolved status=open priority=P2 tags=#cruft,#note : Old-style handler functions in operations/handler.go
// @see #8uuufn
// No action needed — these handlers are used by bootstrap code, not the dispatch system.
//
// This file contains NewHeartbeatHandler, NewSystemStatusHandler,
// NewDocumentationHandler, NewLogAccessHandler, NewMarkBootstrappedHandler,
// NewResetHandler, and NewSchedulerListHandler — all using the old pattern
// of returning abstract.MessageHandler directly.
//
// Unlike other packages, these handlers are NOT superseded by generated
// registrations. They are used directly by the runtime bootstrap code
// (core/internal/boot/app.go) and are not registered via the message
// dispatch system. These are legitimate old-style handlers that serve a
// different purpose.
//
// Resolution: no action needed — these handlers are not part of the
// generated registration pattern.
package operations

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	auditmodel "github.com/asaidimu/hestia/core/system/audit/model"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func NewHeartbeatHandler() abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}
}

func NewSystemStatusHandler(bootstrapped func() bool) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return dispatch.NewDocumentResultFrom(&HealthView{
			Ok:           true,
			Bootstrapped: bootstrapped(),
		})
	}
}

func NewDocumentationHandler(registrations *[]abstract.MessageRegistration, apiPrefix string) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		regs := *registrations
		docs := make([]*document.Document, 0, len(regs))
		for _, r := range regs {

			method := httpapi.IntentToHTTPMethod(r.Intent)
			httpPath := httpapi.DeriveRoute(r.Name, r.Input.Arguments())
			if apiPrefix != "" {
				httpPath = apiPrefix + httpPath
			}
			pattern := method + " " + httpapi.IntentToHTTPPath(r.Intent, httpPath)
			view := &EndpointDoc{
				Name:          r.Name,
				Description:   r.Description,
				Enabled:       r.Enabled,
				Intent:        r.Intent.String(),
				BootstrapSafe: r.BootstrapSafe,
				Internal:      r.Internal,
				HTTP: HTTPMapping{
					Method:  method,
					Route:   httpapi.IntentToHTTPPath(r.Intent, httpPath),
					Pattern: pattern,
				},
			}
			if r.Input.Schema != nil {
				view.Input = r.Input.Schema.AsMap()
			}
			if r.Output != nil {
				view.Output = r.Output.AsMap()
			}
			doc, err := document.New(view).Document()
			if err != nil {
				return nil, err
			}
			docs = append(docs, doc)
		}
		return &abstract.Result{Documents: docs}, nil
	}
}

func NewLogAccessHandler(model *auditmodel.SystemAuditLogs) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		entry := extractAuditEntry(msg.Input())
		if err := model.Insert(ctx, entry); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewMarkBootstrappedHandler(onBootstrapped func()) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		if onBootstrapped != nil {
			go onBootstrapped()
		}
		return &abstract.Result{}, nil
	}
}

func NewResetHandler(onReset func()) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		if onReset != nil {
			go onReset()
		}
		return &abstract.Result{}, nil
	}
}

func NewSchedulerListHandler(sched *scheduler.Scheduler) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		jobs := sched.List()
		docs := make([]*document.Document, 0, len(jobs))
		for _, j := range jobs {
			doc, err := document.New(&SchedulerJobInfo{
				Name:   j.Name,
				Expr:   j.Expr,
				Next:   j.Next.Format(time.RFC3339),
				Prev:   j.Prev.Format(time.RFC3339),
				Paused: j.Paused,
				Tags:   j.Tags,
			}).Document()
			if err != nil {
				return nil, err
			}
			docs = append(docs, doc)
		}
		return &abstract.Result{Documents: docs}, nil
	}
}
