package http

import (
	"context"
	"strings"
	"time"

	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"go.uber.org/zap"
)

// AccessLog returns a Middleware that logs every request with structured fields
// for operation, duration, request_id, client_ip, user_agent, user_id, email,
// and error code (when present).
func AccessLog(logger *zap.Logger) Middleware {
	return func(ctx context.Context, req Request, next handlerFunc) (Response, error) {
		start := time.Now()
		resp, err := next(ctx, req)
		duration := time.Since(start)

		fields := []zap.Field{
			zap.String("operation", req.Operation),
			zap.Duration("duration", duration),
		}

		// Parse method and path from operation (e.g. "POST /api/system/session")
		if parts := strings.SplitN(req.Operation, " ", 2); len(parts) == 2 {
			fields = append(fields, zap.String("method", parts[0]))
			fields = append(fields, zap.String("path", parts[1]))
		}

		// Request-level context fields (set by transport before middleware)
		if req.RequestID != "" {
			fields = append(fields, zap.String("request_id", req.RequestID))
		} else if rid := runtimecontext.GetRequestID(ctx); rid != "" {
			fields = append(fields, zap.String("request_id", rid))
		}
		if req.ClientIP != "" {
			fields = append(fields, zap.String("client_ip", req.ClientIP))
		} else if ip := runtimecontext.GetSourceIP(ctx); ip != "" {
			fields = append(fields, zap.String("client_ip", ip))
		}
		if req.UserAgent != "" {
			fields = append(fields, zap.String("user_agent", req.UserAgent))
		} else if ua := runtimecontext.GetUserAgent(ctx); ua != "" {
			fields = append(fields, zap.String("user_agent", ua))
		}

		// Tenant
		if tid := runtimecontext.GetTenantID(ctx); tid != "" {
			fields = append(fields, zap.String("tenant_id", tid))
		}

		// Claims (user_id, email) — set by auth middleware which runs inside `next`
		if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok && claims != nil {
			if claims.UserID != "" {
				fields = append(fields, zap.String("user_id", claims.UserID))
			}
			if claims.Email != "" {
				fields = append(fields, zap.String("email", claims.Email))
			}
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Warn("request", fields...)
		} else {
			logger.Info("request", fields...)
		}

		return resp, err
	}
}
