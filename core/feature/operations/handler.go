package operations

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/feature/audit"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
)

func NewHeartbeatHandler() abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}
}

func NewSystemStatusHandler(bootstrapped func() bool) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{
			Document: data.MustNewDocument(map[string]any{
				"ok":           true,
				"bootstrapped": bootstrapped(),
			}, ctx),
		}, nil
	}
}

func NewDocumentationHandler(registrations *[]abstract.MessageRegistration, apiPrefix string) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		regs := *registrations
		docs := make(data.DocumentSet, 0, len(regs))
		for _, r := range regs {

			method := httpapi.IntentToHTTPMethod(r.Intent)
			httpPath := httpapi.DeriveRoute(r.Name, r.Input.Arguments)
			if apiPrefix != "" {
				httpPath = apiPrefix + httpPath
			}
			pattern := method + " " + httpapi.IntentToHTTPPath(r.Intent, httpPath)
			doc := data.MustNewDocument(map[string]any{
				"name":           r.Name,
				"description":    r.Description,
				"enabled":        r.Enabled,
				"intent":         r.Intent,
				"bootstrap_safe": r.BootstrapSafe,
				"internal":       r.Internal,
				"http": map[string]string{
					"method":  method,
					"route":   httpapi.IntentToHTTPPath(r.Intent, httpPath),
					"pattern": pattern,
				},
			}, ctx)
			if r.Input.Schema != nil {
				doc.Set("input", r.Input.Schema)
			}
			if r.Output != nil {
				doc.Set("output", r.Output)
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
		docs := make(data.DocumentSet, 0, len(jobs))
		for _, j := range jobs {
			docs = append(docs, data.MustNewDocument(map[string]any{
				"name":   j.Name,
				"expr":   j.Expr,
				"next":   j.Next.Format(time.RFC3339),
				"prev":   j.Prev.Format(time.RFC3339),
				"paused": j.Paused,
				"tags":   j.Tags,
			}, ctx))
		}
		return &abstract.Result{Documents: docs}, nil
	}
}

func extractAuditEntry(doc data.Documenter) auditdomain.AuditEntry {
	return auditdomain.AuditEntry{
		EventID:      getStr(doc, "event_id"),
		OccurredAt:   getStr(doc, "occurred_at"),
		RecordedAt:   getStr(doc, "recorded_at"),
		TraceID:      getStr(doc, "trace_id"),
		RequestID:    getStr(doc, "request_id"),
		ActorID:      getStr(doc, "actor_id"),
		ActorType:    auditdomain.ActorType(getStr(doc, "actor_type")),
		OnBehalfOfID: getStr(doc, "on_behalf_of_id"),
		AuthMethod:   auditdomain.AuthMethod(getStr(doc, "auth_method")),
		SessionID:    getStr(doc, "session_id"),
		Operation:    auditdomain.Operation(getStr(doc, "operation")),
		ResourceType: getStr(doc, "resource_type"),
		ResourceID:   getStr(doc, "resource_id"),
		EventName:    getStr(doc, "event_name"),
		Status:       auditdomain.AuditStatus(getStr(doc, "status")),
		Severity:     auditdomain.Severity(getStr(doc, "severity")),
		ErrorCode:    getStr(doc, "error_code"),
		ErrorMessage: getStr(doc, "error_message"),
		LatencyMs:    getInt64(doc, "latency_ms"),
		SourceIP:     getStr(doc, "source_ip"),
		UserAgent:    getStr(doc, "user_agent"),
		ServiceName:  getStr(doc, "service_name"),
		Region:       getStr(doc, "region"),
	}
}

func getStr(doc data.Documenter, key string) string {
	if v := doc.GetOr(key, nil); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(doc data.Documenter, key string) int64 {
	if v := doc.GetOr(key, nil); v != nil {
		switch n := v.(type) {
		case int64:
			return n
		case float64:
			return int64(n)
		}
	}
	return 0
}
