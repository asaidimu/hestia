package api

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/abstract"
	httpserver "github.com/asaidimu/hestia/internal/interface/api/http"
)

type Options struct {
	Dispatcher        core.Dispatcher
	InternalDispatcher core.Dispatcher
	Logger            *zap.Logger
	Addr              string
	Registrations     []abstract.MessageRegistration
	CookieConfig      core.CookieConfig
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
}

type Interface struct {
	opts            Options
	trans           Transport
	disp            core.Dispatcher
	internalDisp    core.Dispatcher
	bootstrapped    bool
	regs            []abstract.MessageRegistration
	cookieCfg       core.CookieConfig
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func New(opts Options) *Interface {
	cfg := opts.CookieConfig
	if cfg.AccessName == "" {
		cfg.AccessName = "access_token"
	}
	if cfg.AccessPath == "" {
		cfg.AccessPath = "/"
	}
	if cfg.RefreshName == "" {
		cfg.RefreshName = "refresh_token"
	}
	if cfg.RefreshPath == "" {
		cfg.RefreshPath = "/api/auth/session"
	}
	accessTTL := opts.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = core.DefaultAccessTokenTTL
	}
	refreshTTL := opts.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = core.DefaultRefreshTokenTTL
	}
	o := &Interface{
		opts:            opts,
		disp:            opts.Dispatcher,
		internalDisp:    opts.InternalDispatcher,
		regs:            opts.Registrations,
		cookieCfg:       cfg,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
	o.trans = newHTTPTransport(opts)
	return o
}

func newHTTPTransport(opts Options) Transport {
	return httpserver.NewTransport(httpserver.TransportOptions{
		Addr:   opts.Addr,
		Logger: opts.Logger,
	})
}

func methodFromOp(op string) string {
	parts := strings.SplitN(op, " ", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

func pathFromOp(op string) string {
	parts := strings.SplitN(op, " ", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return op
}

func (o *Interface) Start(bootstrapped bool) {
	o.bootstrapped = bootstrapped
	o.registerRoutes()
	go func() {
		if err := o.trans.Start(); err != nil {
			o.opts.Logger.Error("HTTP transport stopped", zap.Error(err))
		}
	}()
}

func (o *Interface) Restart(bootstrapped bool) {
	_ = o.trans.Shutdown(context.Background())
	o.bootstrapped = bootstrapped
	o.trans = newHTTPTransport(o.opts)
	o.registerRoutes()
	go func() {
		if err := o.trans.Start(); err != nil {
			o.opts.Logger.Error("HTTP transport stopped", zap.Error(err))
		}
	}()
}

func (o *Interface) Shutdown(ctx context.Context) error {
	return o.trans.Shutdown(ctx)
}

// ── core.Interface compliance ───────────────────────────────────────────
var _ core.Interface = (*Interface)(nil)

// ── Route registration ─────────────────────────────────────────────────────

func (o *Interface) registerRoutes() {
	if o.bootstrapped {
		o.installDispatcherRegistrations()
	} else {
		o.installBootstrapSafeRegistrations()
	}
}

// ── Middleware stack ────────────────────────────────────────────────────────

type handlerFunc func(ctx context.Context, req Request) (Response, error)

func (o *Interface) wrap(fn handlerFunc) Handler {
	return func(ctx context.Context, req Request) (resp Response, err error) {
		ctx = core.ContextWithAuditTransport(ctx, req.ClientIP, req.UserAgent, req.RequestID)
		ctx = core.ContextWithTraceID(ctx, req.RequestID)
		resp, err = o.authMiddleware(ctx, req, fn)
		if v, _ := ctx.Value(clearAccessCookieKey).(bool); v {
			resp.Cookies = append(resp.Cookies, clearCookie(o.cookieCfg.AccessName, o.cookieCfg.AccessPath))
		}
		if v, _ := ctx.Value(clearRefreshCookieKey).(bool); v {
			resp.Cookies = append(resp.Cookies, clearCookie(o.cookieCfg.RefreshName, o.cookieCfg.RefreshPath))
		}
		return
	}
}

func clearCookie(name, path string) Cookie {
	return Cookie{
		Name:   name,
		Value:  "",
		Path:   path,
		MaxAge: -1,
	}
}
