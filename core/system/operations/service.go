package operations

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/route"
	auditmodel "github.com/asaidimu/hestia/core/system/audit/model"
	"github.com/asaidimu/hestia/core/runtime"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"github.com/asaidimu/hestia/core/system/operations/model"
)

// Named callback types so the DI container can distinguish the three
// boot-time function instances registered under distinct reflect types.
type (
	BootstrappedFunc func() bool
	OnBootstrapFunc  func()
	OnResetFunc      func()
)

// OperationsService is the service for system-level core operations: health,
// capability registry, audit sink, endpoint documentation, bootstrap marking,
// reset, and scheduler introspection. It resolves shared boot wiring from the
// runtime DI container — the same LocalDispatcher, audit model, callback
// functions, registration slice, and scheduler the module builds and mutates
// during boot, so the documented list always reflects the live registry.
type OperationsService struct {
	log           *zap.Logger
	disp          *runtime.LocalDispatcher
	bootstrapped  BootstrappedFunc
	onBootstrap   OnBootstrapFunc
	onReset       OnResetFunc
	auditModel    *auditmodel.SystemAuditLogs
	registrations *[]abstract.MessageRegistration
	scheduler     *scheduler.Scheduler
}

func NewOperationsService(rt abstract.Container) (*OperationsService, error) {
	disp := abstract.MustResolve[*runtime.LocalDispatcher](rt)
	bootstrapped := abstract.MustResolve[BootstrappedFunc](rt)
	onBootstrap := abstract.MustResolve[OnBootstrapFunc](rt)
	onReset := abstract.MustResolve[OnResetFunc](rt)
	auditModel := abstract.MustResolve[*auditmodel.SystemAuditLogs](rt)
	registrations := abstract.MustResolve[*[]abstract.MessageRegistration](rt)
	sched := abstract.MustResolve[*scheduler.Scheduler](rt)
	log, _ := abstract.Resolve[*zap.Logger](rt)

	return &OperationsService{
		log:           log,
		disp:          disp,
		bootstrapped:  bootstrapped,
		onBootstrap:   onBootstrap,
		onReset:       onReset,
		auditModel:    auditModel,
		registrations: registrations,
		scheduler:     sched,
	}, nil
}

// Heartbeat is a session keepalive — it does not count as a health check.
//
// @hestia.register(
//   name="system:core:heartbeat",
//   intent="read",
//   rule="authenticated",
//   description="Session keepalive — does not count as a health check",
//   bootstrap_safe="true",
// )
func (s *OperationsService) Heartbeat(ctx context.Context, msg abstract.Message) error {
	return nil
}

// HealthCheck reports system health and bootstrap status.
//
// @hestia.register(
//   name="system:core:health:check",
//   intent="read",
//   rule="public",
//   description="Check system health and bootstrap status",
//   bootstrap_safe="true",
// )
func (s *OperationsService) HealthCheck(ctx context.Context, msg abstract.Message) (*model.HealthView, error) {
	return document.New(&model.HealthView{
		Ok:           true,
		Bootstrapped: s.bootstrapped(),
	}), nil
}

// ListCapabilities lists all registered handlers.
//
// @hestia.register(
//   name="system:core:capability:list",
//   intent="read",
//   rule="administrator",
//   description="List all registered commands and queries with descriptions and enabled status",
// )
func (s *OperationsService) ListCapabilities(ctx context.Context, msg abstract.Message) (*runtime.CapabilitiesDocument, error) {
	all := s.disp.ListHandlers()
	items := make([]runtime.CapabilityItem, len(all))
	for i, h := range all {
		items[i] = runtime.CapabilityItem{
			Name:          h.Name,
			IntentType:    string(h.IntentType),
			Description:   h.Description,
			Enabled:       h.Enabled,
			BootstrapSafe: h.BootstrapSafe,
		}
	}
	return document.New(&runtime.CapabilitiesDocument{Capabilities: items}), nil
}

// SetCapability enables or disables a registered handler.
//
// @hestia.register(
//   name="system:core:capability:set",
//   intent="update",
//   rule="administrator",
//   description="Enable or disable a registered command or query",
//   resource_id="name",
// )
func (s *OperationsService) SetCapability(ctx context.Context, msg abstract.Message, input *model.CapabilityNameInput) error {
	if input.Name == "" {
		return runtime.ErrValidation.WithOperation("system:core:capability:set")
	}
	return s.disp.SetHandlerEnabled(input.Name, input.Enabled)
}

// LogAccess records an audit log entry.
//
// @hestia.register(
//   name="system:core:audit:log",
//   intent="create",
//   rule="authenticated",
//   description="Log an API access entry",
//   internal="true",
// )
func (s *OperationsService) LogAccess(ctx context.Context, msg abstract.Message) error {
	entry := extractAuditEntry(msg.Input())
	if s.log != nil {
		s.log.Debug("recording audit entry", zap.String("operation", string(entry.Operation)))
	}
	return s.auditModel.Insert(ctx, entry)
}

// DocsList returns endpoint documentation for all registered handlers.
//
// @hestia.register(
//   name="system:core:docs:list",
//   intent="read",
//   rule="public",
//   description="List all registered API endpoints with metadata",
//   bootstrap_safe="true",
// )
func (s *OperationsService) DocsList(ctx context.Context, msg abstract.Message) ([]*document.Document, error) {
	regs := *s.registrations
	docs := make([]*document.Document, 0, len(regs))
	for _, r := range regs {
		method := route.IntentToHTTPMethod(r.Intent)
		httpPath := route.DeriveRoute(r.Name, r.Input.Arguments())
		pattern := method + " " + route.IntentToHTTPPath(r.Intent, httpPath)
		view := &model.EndpointDoc{
			Name:          r.Name,
			Description:   r.Description,
			Enabled:       r.Enabled,
			Intent:        r.Intent.String(),
			BootstrapSafe: r.BootstrapSafe,
			Internal:      r.Internal,
			HTTP: model.HTTPMapping{
				Method:  method,
				Route:   route.IntentToHTTPPath(r.Intent, httpPath),
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
	return docs, nil
}

// MarkBootstrapped marks the system as bootstrapped.
//
// @hestia.register(
//   name="system:core:bootstrap:mark",
//   intent="create",
//   rule="public",
//   description="Mark system as bootstrapped",
//   internal="true",
//   bootstrap_safe="true",
// )
func (s *OperationsService) MarkBootstrapped(ctx context.Context, msg abstract.Message) error {
	if s.onBootstrap != nil {
		go s.onBootstrap()
	}
	return nil
}

// Reset resets the system to its initial state.
//
// @hestia.register(
//   name="system:core:reset",
//   intent="read",
//   rule="administrator",
//   description="Reset system to initial state",
// )
func (s *OperationsService) Reset(ctx context.Context, msg abstract.Message) error {
	if s.onReset != nil {
		go s.onReset()
	}
	return nil
}

// SchedulerJobs lists all registered scheduler jobs.
//
// @hestia.register(
//   name="system:scheduler:job:list",
//   intent="read",
//   rule="administrator",
//   description="List all registered scheduler jobs",
// )
func (s *OperationsService) SchedulerJobs(ctx context.Context, msg abstract.Message) ([]*document.Document, error) {
	jobs := s.scheduler.List()
	docs := make([]*document.Document, 0, len(jobs))
	for _, j := range jobs {
		doc, err := document.New(&model.SchedulerJobInfo{
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
	return docs, nil
}

func extractAuditEntry(doc interface{ GetOr(string, any) any }) auditdomain.AuditEntry {
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

func getStr(doc interface{ GetOr(string, any) any }, key string) string {
	if v := doc.GetOr(key, nil); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(doc interface{ GetOr(string, any) any }, key string) int64 {
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