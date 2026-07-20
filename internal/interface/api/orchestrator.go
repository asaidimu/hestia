package api

import (
	"context"
	"io/fs"
	"strings"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/users"
	httpserver "github.com/asaidimu/hestia/internal/interface/api/http"
)

type Middleware func(ctx context.Context, req Request, next handlerFunc) (Response, error)

type Options struct {
	Dispatcher         core.Dispatcher
	InternalDispatcher  core.Dispatcher
	CredentialsProvider abstract.CredentialsProvider
	Logger             *zap.Logger
	Addr               string
	Registrations      []abstract.MessageRegistration
	CookieConfig       core.CookieConfig
	SessionTTL         time.Duration
	IdleTTL            time.Duration
	RefreshTTL         time.Duration
	APIPrefix          string
	StaticFS           fs.FS
	UserModel          *users.UserModel
	Middleware         []Middleware
	NoRefreshCommands  []string
}

type Interface struct {
	opts           Options
	trans          Transport
	disp           core.Dispatcher
	internalDisp   core.Dispatcher
	identityProv   iam.IdentityProvider
	credProv       abstract.CredentialsProvider
	userModel      *users.UserModel
	bootstrapped   bool
	regs           []abstract.MessageRegistration
	cookieCfg      core.CookieConfig
	sessionTTL     time.Duration
	idleTTL        time.Duration
	refreshTTL     time.Duration
	middleware         []Middleware
	noRefreshCommands  map[string]struct{}
	noRefreshOps       map[string]struct{}
}

func New(opts Options) *Interface {
	cfg := opts.CookieConfig
	if cfg.SessionName == "" {
		cfg.SessionName = "session"
	}
	if cfg.SessionPath == "" {
		cfg.SessionPath = "/"
	}
	if opts.APIPrefix == "" {
		opts.APIPrefix = "/api"
	}
	sessionTTL := opts.SessionTTL
	if sessionTTL <= 0 {
		sessionTTL = core.DefaultSessionTTL
	}
	idleTTL := opts.IdleTTL
	if idleTTL <= 0 {
		idleTTL = core.DefaultIdleTTL
	}
	refreshTTL := opts.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = core.DefaultRefreshTTL
	}
	nrc := make(map[string]struct{}, len(opts.NoRefreshCommands))
	for _, p := range opts.NoRefreshCommands {
		nrc[p] = struct{}{}
	}

	o := &Interface{
		opts:              opts,
		disp:              opts.Dispatcher,
		internalDisp:      opts.InternalDispatcher,
		identityProv:      newIdentityProvider(opts.CredentialsProvider, opts.InternalDispatcher),
		credProv:          opts.CredentialsProvider,
		userModel:         opts.UserModel,
		regs:              opts.Registrations,
		cookieCfg:         cfg,
		sessionTTL:        sessionTTL,
		idleTTL:           idleTTL,
		refreshTTL:        refreshTTL,
		middleware:        opts.Middleware,
		noRefreshCommands: nrc,
		noRefreshOps:      make(map[string]struct{}),
	}
	o.trans = newHTTPTransport(opts)
	return o
}

func newHTTPTransport(opts Options) Transport {
	return httpserver.NewTransport(httpserver.TransportOptions{
		Addr:      opts.Addr,
		Logger:    opts.Logger,
		APIPrefix: opts.APIPrefix,
		StaticFS:  opts.StaticFS,
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

func (o *Interface) SetMiddleware(mw ...Middleware) {
	o.middleware = append(o.middleware, mw...)
}

var _ core.Interface = (*Interface)(nil)

func (o *Interface) registerRoutes() {
	if o.bootstrapped {
		o.installDispatcherRegistrations()
	} else {
		o.installBootstrapSafeRegistrations()
	}
}

type HandlerFunc func(ctx context.Context, req Request) (Response, error)
type handlerFunc = HandlerFunc

func (o *Interface) wrap(fn handlerFunc) Handler {
	var chain HandlerFunc
	chain = func(ctx context.Context, req Request) (Response, error) {
		return o.authMiddleware(ctx, req, fn)
	}
	for i := len(o.middleware) - 1; i >= 0; i-- {
		mw := o.middleware[i]
		next := chain
		chain = func(ctx context.Context, req Request) (Response, error) {
			return mw(ctx, req, next)
		}
	}

	return func(ctx context.Context, req Request) (resp Response, err error) {
		ctx = core.ContextWithAuditTransport(ctx, req.ClientIP, req.UserAgent, req.RequestID)
		ctx = core.ContextWithTraceID(ctx, req.RequestID)
		resp, err = chain(ctx, req)

		if v, _ := ctx.Value(setSessionCookieKey).(string); v != "" {
			resp.Cookies = append(resp.Cookies, cookie(o.cookieCfg.SessionName, v, o.cookieCfg.SessionPath, o.sessionTTL, o.cookieCfg))
		} else if v, _ := ctx.Value(clearSessionCookieKey).(bool); v {
			resp.Cookies = append(resp.Cookies, clearCookie(o.cookieCfg.SessionName, o.cookieCfg.SessionPath))
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

func cookie(name, value, path string, ttl time.Duration, cfg core.CookieConfig) Cookie {
	return Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HTTPOnly: cfg.HTTPOnly,
		SameSite: cfg.SameSite,
		MaxAge:   int(ttl.Seconds()),
	}
}
