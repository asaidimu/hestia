package operations

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/system/audit"
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
			httpPath := httpapi.DeriveRoute(r.Name, r.Input.Args())
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

func NewLogAccessHandler(model *audit.AuditModel) abstract.MessageHandler {
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
