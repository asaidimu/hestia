package api

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/interface"
	"github.com/asaidimu/hestia/internal/abstract"
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

type Orchestrator struct {
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

func New(opts Options) *Orchestrator {
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
	o := &Orchestrator{
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

func (o *Orchestrator) Start(bootstrapped bool) {
	o.bootstrapped = bootstrapped
	o.registerRoutes()
	go func() {
		if err := o.trans.Start(); err != nil {
			o.opts.Logger.Error("HTTP transport stopped", zap.Error(err))
		}
	}()
}

func (o *Orchestrator) Restart(bootstrapped bool) {
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

func (o *Orchestrator) Shutdown(ctx context.Context) error {
	return o.trans.Shutdown(ctx)
}

// ── orchestrator.Orchestrator compliance ───────────────────────────────────
var _ orchestrator.Orchestrator = (*Orchestrator)(nil)

// ── Route registration ─────────────────────────────────────────────────────

func (o *Orchestrator) registerRoutes() {
	if o.bootstrapped {
		o.installDispatcherRegistrations()
	} else {
		o.installBootstrapSafeRegistrations()
	}
}

// ── Middleware stack ────────────────────────────────────────────────────────

type handlerFunc func(ctx context.Context, req Request) (Response, error)

func (o *Orchestrator) wrap(fn handlerFunc) Handler {
	return func(ctx context.Context, req Request) (resp Response, err error) {
		ctx = core.ContextWithTransportMetadata(ctx, req.ClientIP, req.UserAgent,
			methodFromOp(req.Operation), pathFromOp(req.Operation), req.RequestID)
		resp, err = o.authMiddleware(ctx, req, fn)
		return
	}
}
