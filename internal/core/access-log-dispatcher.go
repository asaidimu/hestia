package core

import (
	"context"
	"errors"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/internal/core/registration"
)

// Context keys for access-log metadata.
// These are set by the orchestrator before dispatching a message.
type accessLogCtxKey string

const (
	AccessLogUserIDKey       accessLogCtxKey = "accesslog.user_id"
	AccessLogCredentialKey   accessLogCtxKey = "accesslog.credential"
	AccessLogClientIPKey     accessLogCtxKey = "accesslog.client_ip"
	AccessLogUserAgentKey    accessLogCtxKey = "accesslog.user_agent"
	AccessLogHTTPMethodKey   accessLogCtxKey = "accesslog.http_method"
	AccessLogHTTPPathKey     accessLogCtxKey = "accesslog.http_path"
	AccessLogRequestIDKey    accessLogCtxKey = "accesslog.request_id"
)

// ContextWithAccessLogIdentity stores user/credential info in the context
// for retrieval by the AccessLogDispatcher.
func ContextWithAccessLogIdentity(ctx context.Context, userID, credential string) context.Context {
	ctx = context.WithValue(ctx, AccessLogUserIDKey, userID)
	ctx = context.WithValue(ctx, AccessLogCredentialKey, credential)
	return ctx
}

// ContextWithTransportMetadata stores HTTP transport metadata in the context
// for retrieval by the AccessLogDispatcher.
func ContextWithTransportMetadata(ctx context.Context, clientIP, userAgent, httpMethod, httpPath, requestID string) context.Context {
	ctx = context.WithValue(ctx, AccessLogClientIPKey, clientIP)
	ctx = context.WithValue(ctx, AccessLogUserAgentKey, userAgent)
	ctx = context.WithValue(ctx, AccessLogHTTPMethodKey, httpMethod)
	ctx = context.WithValue(ctx, AccessLogHTTPPathKey, httpPath)
	ctx = context.WithValue(ctx, AccessLogRequestIDKey, requestID)
	return ctx
}

// AccessLogDispatcher wraps a Dispatcher and persists an access log entry
// for every message that passes through the chain.
type AccessLogDispatcher struct {
	next      Dispatcher
	persister AccessLogPersister
}

func NewAccessLogDispatcher(next Dispatcher, persister AccessLogPersister) *AccessLogDispatcher {
	return &AccessLogDispatcher{next: next, persister: persister}
}

func (d *AccessLogDispatcher) Send(msg Message) (*registration.Result, error) {
	start := time.Now()
	result, err := d.next.Send(msg)
	latency := time.Since(start)

	d.log(msg, result, err, latency)

	return result, err
}

func (d *AccessLogDispatcher) log(msg Message, result *registration.Result, handlerErr error, latency time.Duration) {
	ctx := msg.Context()

	entry := AccessLogEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		MessageName: msg.Name(),
		MessageID:   msg.ID(),
		Status:      AccessStatusOK,
		LatencyMs:   latency.Milliseconds(),
	}

	if v, _ := ctx.Value(AccessLogRequestIDKey).(string); v != "" {
		entry.RequestID = v
	}
	if v, _ := ctx.Value(AccessLogUserIDKey).(string); v != "" {
		entry.UserID = v
	}
	if v, _ := ctx.Value(AccessLogCredentialKey).(string); v != "" {
		entry.Credential = v
	}

	switch {
	case handlerErr != nil:
		entry.Status = AccessStatusError
		entry.Error = handlerErr.Error()
		var sysErr *common.SystemError
		if errors.As(handlerErr, &sysErr) && sysErr.Code == "ERR_ACCESS_DENIED" {
			entry.Status = AccessStatusDenied
		}
	}

	transport := make(map[string]any)
	if v, _ := ctx.Value(AccessLogHTTPMethodKey).(string); v != "" {
		transport["method"] = v
	}
	if v, _ := ctx.Value(AccessLogHTTPPathKey).(string); v != "" {
		transport["path"] = v
	}
	if v, _ := ctx.Value(AccessLogClientIPKey).(string); v != "" {
		transport["client_ip"] = v
	}
	if v, _ := ctx.Value(AccessLogUserAgentKey).(string); v != "" {
		transport["user_agent"] = v
	}
	if len(transport) > 0 {
		entry.TransportMetadata = transport
	}

	if err := d.persister.Insert(ctx, entry); err != nil {
		// Persistence failure is non-fatal; swallow to avoid cascading
		// failures down the dispatch chain.
	}
}
