package runtime

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"

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
)

func ContextWithAuditResourceID(ctx context.Context, resourceID string) context.Context {
	return context.WithValue(ctx, AuditResourceIDKey, resourceID)
}

func ContextWithAuditIdentity(ctx context.Context, actorID string, actorType ActorType, authMethod AuthMethod) context.Context {
	ctx = context.WithValue(ctx, AuditActorIDKey, actorID)
	ctx = context.WithValue(ctx, AuditActorTypeKey, actorType)
	ctx = context.WithValue(ctx, AuditAuthMethodKey, authMethod)
	return ctx
}

func ContextWithAuditTransport(ctx context.Context, sourceIP, userAgent, requestID string) context.Context {
	ctx = context.WithValue(ctx, AuditSourceIPKey, sourceIP)
	ctx = context.WithValue(ctx, AuditUserAgentKey, userAgent)
	ctx = context.WithValue(ctx, AuditRequestIDKey, requestID)
	return ctx
}

func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, AuditTraceIDKey, traceID)
}

func ContextWithAuditOnBehalfOf(ctx context.Context, onBehalfOfID string) context.Context {
	return context.WithValue(ctx, AuditOnBehalfOfIDKey, onBehalfOfID)
}

func ContextWithAuditSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, AuditSessionIDKey, sessionID)
}

func GetTraceID(ctx context.Context) string {
	v, _ := ctx.Value(AuditTraceIDKey).(string)
	return v
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
}

func NewAuditDispatcher(next Dispatcher, persister AuditPersister) *AuditDispatcher {
	return &AuditDispatcher{next: next, persister: persister}
}

func (d *AuditDispatcher) Send(msg Message) (*registration.Result, error) {
	start := time.Now()
	result, err := d.next.Send(msg)
	latency := time.Since(start)

	d.log(msg, result, err, latency)

	return result, err
}

func (d *AuditDispatcher) log(msg Message, result *registration.Result, handlerErr error, latency time.Duration) {
	ctx := msg.Context()
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
	}

	if v, _ := ctx.Value(AuditRequestIDKey).(string); v != "" {
		entry.RequestID = v
	}
	if v, _ := ctx.Value(AuditActorIDKey).(string); v != "" {
		entry.ActorID = v
	}
	if v, _ := ctx.Value(AuditActorTypeKey).(ActorType); v != "" {
		entry.ActorType = v
	} else {
		entry.ActorType = deriveActorType(ctx)
	}
	if v, _ := ctx.Value(AuditAuthMethodKey).(AuthMethod); v != "" {
		entry.AuthMethod = v
	}
	if v, _ := ctx.Value(AuditTraceIDKey).(string); v != "" {
		entry.TraceID = v
	}
	if v, _ := ctx.Value(AuditOnBehalfOfIDKey).(string); v != "" {
		entry.OnBehalfOfID = v
	}
	if v, _ := ctx.Value(AuditSessionIDKey).(string); v != "" {
		entry.SessionID = v
	}
	if v, _ := ctx.Value(AuditSourceIPKey).(string); v != "" {
		entry.SourceIP = v
	}
	if v, _ := ctx.Value(AuditUserAgentKey).(string); v != "" {
		entry.UserAgent = v
	}

	if v, _ := ctx.Value(AuditResourceIDKey).(string); v != "" {
		entry.ResourceID = v
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

	if err := d.persister.Insert(ctx, entry); err != nil {
		// Persistence failure is non-fatal; swallow to avoid cascading
		// failures down the dispatch chain.
	}
}
