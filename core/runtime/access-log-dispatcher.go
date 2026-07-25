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
	"github.com/asaidimu/hestia/core/registration"
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

func ContextWithAuditResourceID(ctx context.Context, resourceID string) context.Context {
	return abstract.ContextWithResourceID(ctx, resourceID)
}

func ContextWithAuditIdentity(ctx context.Context, actorID string, actorType ActorType, authMethod AuthMethod) context.Context {
	ctx = context.WithValue(ctx, AuditActorIDKey, actorID)
	ctx = context.WithValue(ctx, AuditActorTypeKey, actorType)
	ctx = context.WithValue(ctx, AuditAuthMethodKey, authMethod)
	return ctx
}

func ContextWithAuditTransport(ctx context.Context, sourceIP, userAgent, requestID string) context.Context {
	ctx = abstract.ContextWithSourceIP(ctx, sourceIP)
	ctx = abstract.ContextWithUserAgent(ctx, userAgent)
	ctx = abstract.ContextWithRequestID(ctx, requestID)
	return ctx
}

func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return abstract.ContextWithTraceID(ctx, traceID)
}

func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return abstract.ContextWithTenantID(ctx, tenantID)
}

func GetTenantID(ctx context.Context) string {
	return abstract.GetTenantID(ctx)
}

func ContextWithAuditSessionID(ctx context.Context, sessionID string) context.Context {
	return abstract.ContextWithSessionID(ctx, sessionID)
}

func GetTraceID(ctx context.Context) string {
	return abstract.GetTraceID(ctx)
}

func deriveOperation(msgName string) Operation {
	parts := strings.Split(msgName, ":")
	if len(parts) == 0 {
		return OperationOther
	}
	action := parts[len(parts)-1]
	switch action {
	case "create", "register", "upload":
		return OperationCreate
	case "read", "get", "head", "list", "download", "query", "search":
		return OperationRead
	case "update", "set", "patch":
		return OperationUpdate
	case "delete", "remove", "clear":
		return OperationDelete
	case "login", "authenticate":
		return OperationLogin
	case "logout":
		return OperationLogout
	case "grant":
		return OperationGrant
	case "revoke":
		return OperationRevoke
	default:
		return OperationExecute
	}
}

func deriveResourceType(msgName string) string {
	parts := strings.Split(msgName, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "unknown"
}

func deriveActorType(ctx context.Context) ActorType {
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		return ActorTypeAnonymous
	}
	props, _ := ident.Properties.(map[string]any)
	if len(ident.Permissions) == 0 && len(props) == 0 {
		return ActorTypeAnonymous
	}
	if v, _ := props["system"].(string); v == "http" {
		return ActorTypeSystem
	}
	return ActorTypeUser
}

type AuditDispatcher struct {
	next      Dispatcher
	persister AuditPersister
	logger    *zap.Logger
	buffer    *AuditBuffer
}

func NewAuditDispatcher(next Dispatcher, persister AuditPersister) *AuditDispatcher {
	return &AuditDispatcher{
		next:      next,
		persister: persister,
	}
}

func NewAuditDispatcherWithLogger(next Dispatcher, persister AuditPersister, logger *zap.Logger) *AuditDispatcher {
	return &AuditDispatcher{
		next:      next,
		persister: persister,
		logger:    logger,
	}
}

func (d *AuditDispatcher) Wrap(next Dispatcher) Dispatcher {
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

func (d *AuditDispatcher) Send(msg Message) (*registration.Result, error) {
	start := time.Now()
	result, err := d.next.Send(msg)
	latency := time.Since(start)

	d.log(msg, result, err, latency)

	return result, err
}

func (d *AuditDispatcher) log(msg Message, result *registration.Result, handlerErr error, latency time.Duration) {
	now := time.Now().UTC()

	entry := AuditEntry{
		EventID:      now.Format("20060102150405") + "-" + msg.ID(),
		OccurredAt:   now.Format(time.RFC3339Nano),
		RecordedAt:   now.Format(time.RFC3339Nano),
		EventName:    msg.Name(),
		Operation:    deriveOperation(msg.Name()),
		ResourceType: deriveResourceType(msg.Name()),
		Status:       AuditStatusSuccess,
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
	if v, _ := msg.Context().Value(AuditActorTypeKey).(ActorType); v != "" {
		entry.ActorType = v
	} else {
		entry.ActorType = deriveActorType(msg.Context())
	}
	if v, _ := msg.Context().Value(AuditAuthMethodKey).(AuthMethod); v != "" {
		entry.AuthMethod = v
	}
	if v, _ := msg.Context().Value(AuditOnBehalfOfIDKey).(string); v != "" {
		entry.OnBehalfOfID = v
	}

	switch {
	case handlerErr != nil:
		entry.Status = AuditStatusError
		entry.ErrorMessage = handlerErr.Error()
		var sysErr *common.SystemError
		if errors.As(handlerErr, &sysErr) && sysErr.Code == "ERR_ACCESS_DENIED" {
			entry.Status = AuditStatusDenied
			entry.ErrorCode = "ERR_ACCESS_DENIED"
		}
	}

	d.Buffer().Write(msg.Context(), entry)
}
