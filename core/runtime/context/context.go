package context

import "context"

type ctxKey string

const (
	ctxKeyTenantID   ctxKey = "tenant_id"
	ctxKeyTraceID    ctxKey = "trace_id"
	ctxKeyRequestID  ctxKey = "request_id"
	ctxKeySourceIP   ctxKey = "source_ip"
	ctxKeyUserAgent  ctxKey = "user_agent"
	ctxKeyResourceID ctxKey = "resource_id"
	ctxKeySessionID  ctxKey = "session_id"
)

func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ctxKeyTenantID, tenantID)
}

func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, traceID)
}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

func ContextWithSourceIP(ctx context.Context, sourceIP string) context.Context {
	return context.WithValue(ctx, ctxKeySourceIP, sourceIP)
}

func ContextWithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, ctxKeyUserAgent, userAgent)
}

func ContextWithResourceID(ctx context.Context, resourceID string) context.Context {
	return context.WithValue(ctx, ctxKeyResourceID, resourceID)
}

func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ctxKeySessionID, sessionID)
}

func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTenantID).(string); ok {
		return v
	}
	return ""
}

func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTraceID).(string); ok {
		return v
	}
	return ""
}

func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}

func GetSourceIP(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeySourceIP).(string); ok {
		return v
	}
	return ""
}

func GetUserAgent(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyUserAgent).(string); ok {
		return v
	}
	return ""
}

func GetResourceID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyResourceID).(string); ok {
		return v
	}
	return ""
}

func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeySessionID).(string); ok {
		return v
	}
	return ""
}
