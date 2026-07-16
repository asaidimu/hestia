package operations

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/internal/app/audit"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/internal/interface/api"
	corepkg "github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/abstract"
)

func NewSystemStatusHandler(bootstrapped func() bool) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"ok":           true,
				"bootstrapped": bootstrapped(),
			}, ctx),
		}, nil
	}
}

func NewDocumentationHandler(registrations *[]abstract.MessageRegistration) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		regs := *registrations
		docs := make(data.DocumentSet, 0, len(regs))
		for _, r := range regs {

			method := api.IntentToHTTPMethod(r.Intent)
			httpPath := api.DeriveRoute(r.Name, r.Input.Arguments)
			pattern := method + " " + api.IntentToHTTPPath(r.Intent, httpPath)
			doc := data.MustNewDocument(map[string]any{
				"name":           r.Name,
				"description":    r.Description,
				"enabled":        r.Enabled,
				"intent":         r.Intent,
				"bootstrap_safe": r.BootstrapSafe,
				"internal":       r.Internal,
				"http": map[string]string{
					"method":  method,
					"route":   api.IntentToHTTPPath(r.Intent, httpPath),
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
		return &registration.Result{Documents: docs}, nil
	}
}

func NewLogAccessHandler(model *audit.AuditModel) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		entry := extractAuditEntry(msg.Input())
		if err := model.Insert(ctx, entry); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewMarkBootstrappedHandler(onBootstrapped func()) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		if onBootstrapped != nil {
			go onBootstrapped()
		}
		return &registration.Result{}, nil
	}
}

func NewResetHandler(onReset func()) corepkg.MessageHandler {
	return func(ctx context.Context, msg corepkg.Message) (*registration.Result, error) {
		if onReset != nil {
			go onReset()
		}
		return &registration.Result{}, nil
	}
}

func extractAuditEntry(doc *data.Document) corepkg.AuditEntry {
	return corepkg.AuditEntry{
		EventID:      getStr(doc, "event_id"),
		OccurredAt:   getStr(doc, "occurred_at"),
		RecordedAt:   getStr(doc, "recorded_at"),
		TraceID:      getStr(doc, "trace_id"),
		RequestID:    getStr(doc, "request_id"),
		ActorID:      getStr(doc, "actor_id"),
		ActorType:    corepkg.ActorType(getStr(doc, "actor_type")),
		OnBehalfOfID: getStr(doc, "on_behalf_of_id"),
		AuthMethod:   corepkg.AuthMethod(getStr(doc, "auth_method")),
		SessionID:    getStr(doc, "session_id"),
		Operation:    corepkg.Operation(getStr(doc, "operation")),
		ResourceType: getStr(doc, "resource_type"),
		ResourceID:   getStr(doc, "resource_id"),
		EventName:    getStr(doc, "event_name"),
		Status:       corepkg.AuditStatus(getStr(doc, "status")),
		Severity:     corepkg.Severity(getStr(doc, "severity")),
		ErrorCode:    getStr(doc, "error_code"),
		ErrorMessage: getStr(doc, "error_message"),
		LatencyMs:    getInt64(doc, "latency_ms"),
		SourceIP:     getStr(doc, "source_ip"),
		UserAgent:    getStr(doc, "user_agent"),
		ServiceName:  getStr(doc, "service_name"),
		Region:       getStr(doc, "region"),
	}
}

func getStr(doc *data.Document, key string) string {
	if v := doc.GetOr(key, nil); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(doc *data.Document, key string) int64 {
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
