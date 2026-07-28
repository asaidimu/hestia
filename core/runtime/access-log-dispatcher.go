package runtime

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/audit"
)

type auditCtxKey string

const (
	AuditActorIDKey      auditCtxKey = "audit.actor_id"
	AuditActorTypeKey    auditCtxKey = "audit.actor_type"
	AuditAuthMethodKey   auditCtxKey = "audit.auth_method"
	AuditOnBehalfOfIDKey auditCtxKey = "audit.on_behalf_of_id"
	AuditSessionIDKey    auditCtxKey = "audit.session_id"
	AuditTraceIDKey      auditCtxKey = "audit.trace_id"
	AuditSourceIPKey     auditCtxKey = "audit.source_ip"
	AuditUserAgentKey    auditCtxKey = "audit.user_agent"
	AuditRequestIDKey    auditCtxKey = "audit.request_id"
	AuditResourceIDKey   auditCtxKey = "audit.resource_id"
	TenantIDKey          auditCtxKey = "tenant_id"
)

func ContextWithAuditIdentity(ctx context.Context, actorID string, actorType audit.ActorType, authMethod audit.AuthMethod) context.Context {
	ctx = context.WithValue(ctx, AuditActorIDKey, actorID)
	ctx = context.WithValue(ctx, AuditActorTypeKey, actorType)
	ctx = context.WithValue(ctx, AuditAuthMethodKey, authMethod)
	return ctx
}

func ContextWithAuditTransport(ctx context.Context, sourceIP, userAgent, requestID string) context.Context {
	ctx = runtimecontext.ContextWithSourceIP(ctx, sourceIP)
	ctx = runtimecontext.ContextWithUserAgent(ctx, userAgent)
	ctx = runtimecontext.ContextWithRequestID(ctx, requestID)
	return ctx
}

func deriveOperation(msgName string) audit.Operation {
	parts := strings.Split(msgName, ":")
	if len(parts) == 0 {
		return audit.OperationOther
	}
	action := parts[len(parts)-1]
	switch action {
	case "create", "register", "upload":
		return audit.OperationCreate
	case "read", "get", "head", "list", "download", "query", "search":
		return audit.OperationRead
	case "update", "set", "patch":
		return audit.OperationUpdate
	case "delete", "remove", "clear":
		return audit.OperationDelete
	case "login", "authenticate":
		return audit.OperationLogin
	case "logout":
		return audit.OperationLogout
	case "grant":
		return audit.OperationGrant
	case "revoke":
		return audit.OperationRevoke
	default:
		return audit.OperationExecute
	}
}

func deriveResourceType(msgName string) string {
	parts := strings.Split(msgName, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "unknown"
}

func deriveActorType(ctx context.Context) audit.ActorType {
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		return audit.ActorTypeAnonymous
	}
	props, _ := ident.Properties.(map[string]any)
	if len(ident.Permissions) == 0 && len(props) == 0 {
		return audit.ActorTypeAnonymous
	}
	if v, _ := props["system"].(string); v == "http" {
		return audit.ActorTypeSystem
	}
	return audit.ActorTypeUser
}

type AuditDispatcher struct {
	next      abstract.Dispatcher
	persister audit.AuditPersister
	logger    *zap.Logger
	buffer    *AuditBuffer
}

func NewAuditDispatcher(next abstract.Dispatcher, persister audit.AuditPersister) *AuditDispatcher {
	return &AuditDispatcher{
		next:      next,
		persister: persister,
	}
}

func NewAuditDispatcherWithLogger(next abstract.Dispatcher, persister audit.AuditPersister, logger *zap.Logger) *AuditDispatcher {
	return &AuditDispatcher{
		next:      next,
		persister: persister,
		logger:    logger,
	}
}

func (d *AuditDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &AuditDispatcher{
		next:      next,
		persister: d.persister,
		logger:    d.logger,
		buffer:    d.buffer,
	}
}

// Buffer returns the shared audit buffer, initialising it on first call.
func (d *AuditDispatcher) Buffer() *AuditBuffer {
	if d.buffer == nil {
		d.buffer = NewAuditBuffer(d.persister, d.logger)
	}
	return d.buffer
}

// Sync blocks until all queued audit entries are flushed.
func (d *AuditDispatcher) Sync() {
	if d.buffer != nil {
		d.buffer.Sync()
	}
}

// Close flushes and stops the audit buffer.
func (d *AuditDispatcher) Close() {
	if d.buffer != nil {
		d.buffer.Close()
	}
}

func (d *AuditDispatcher) Send(msg abstract.Message) (*abstract.Result, error) {
	start := time.Now()
	result, err := d.next.Send(msg)
	latency := time.Since(start)

	d.log(msg, result, err, latency)

	return result, err
}

func (d *AuditDispatcher) log(msg abstract.Message, result *abstract.Result, handlerErr error, latency time.Duration) {
	now := time.Now().UTC()

	entry := audit.AuditEntry{
		EventID:      now.Format("20060102150405") + "-" + msg.ID(),
		OccurredAt:   now.Format(time.RFC3339Nano),
		RecordedAt:   now.Format(time.RFC3339Nano),
		EventName:    msg.Name(),
		Operation:    deriveOperation(msg.Name()),
		ResourceType: deriveResourceType(msg.Name()),
		Status:       audit.AuditStatusSuccess,
		LatencyMs:    latency.Milliseconds(),
		ServiceName:  "hestia",
		RequestID:    msg.RequestID(),
		TraceID:      msg.TraceID(),
		SessionID:    msg.SessionID(),
		SourceIP:     msg.SourceIP(),
		UserAgent:    msg.UserAgent(),
		ResourceID:   msg.ResourceID(),
	}

	if v, _ := msg.Context().Value(AuditActorIDKey).(string); v != "" {
		entry.ActorID = v
	}
	if v, _ := msg.Context().Value(AuditActorTypeKey).(audit.ActorType); v != "" {
		entry.ActorType = v
	} else {
		entry.ActorType = deriveActorType(msg.Context())
	}
	if v, _ := msg.Context().Value(AuditAuthMethodKey).(audit.AuthMethod); v != "" {
		entry.AuthMethod = v
	}
	if v, _ := msg.Context().Value(AuditOnBehalfOfIDKey).(string); v != "" {
		entry.OnBehalfOfID = v
	}

	switch {
	case handlerErr != nil:
		entry.Status = audit.AuditStatusError
		entry.ErrorMessage = handlerErr.Error()
		var sysErr *common.SystemError
		if errors.As(handlerErr, &sysErr) && sysErr.Code == "ERR_ACCESS_DENIED" {
			entry.Status = audit.AuditStatusDenied
			entry.ErrorCode = "ERR_ACCESS_DENIED"
		}
	}

	d.Buffer().Write(msg.Context(), entry)
}
