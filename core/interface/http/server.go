package http

import (
	"context"
	"io/fs"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

type cookieAction struct {
	SetToken string
	Clear    bool
}

type contextKey string

const cookieActionKey contextKey = "cookie_action"

type Middleware func(ctx context.Context, req Request, next handlerFunc) (Response, error)

type Options struct {
	Dispatcher          abstract.Dispatcher
	InternalDispatcher  abstract.Dispatcher
	CredentialsProvider abstract.CredentialsProvider
	Logger              *zap.Logger
	Addr                string
	Registrations       []abstract.MessageRegistration
	CookieConfig        runtime.CookieConfig
	SessionTTL          time.Duration
	IdleTTL             time.Duration
	RefreshTTL          time.Duration
	APIPrefix           string
	StaticFS            fs.FS
	UserModel           abstract.UserResolver
	Middleware          []Middleware
	NoRefreshCommands   []string
	AllowedOrigins      []string
}

type Interface struct {
	opts              Options
	trans             Transport
	disp              abstract.Dispatcher
	internalDisp      abstract.Dispatcher
	identityProv      iam.IdentityProvider
	credProv          abstract.CredentialsProvider
	userModel         abstract.UserResolver
	bootstrapped      bool
	regs              []abstract.MessageRegistration
	cookieCfg         runtime.CookieConfig
	sessionTTL        time.Duration
	idleTTL           time.Duration
	refreshTTL        time.Duration
	middleware        []Middleware
	noRefreshCommands map[string]struct{}
	noRefreshOps      map[string]struct{}
	// authLimitedOps holds every route pattern ("POST /api/...") and message
	// name that is subject to the auth brute-force limiter — see
	// authMiddleware. Populated in installRegistration.
	authLimitedOps map[string]struct{}
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
		sessionTTL = runtime.DefaultSessionTTL
	}
	idleTTL := opts.IdleTTL
	if idleTTL <= 0 {
		idleTTL = runtime.DefaultIdleTTL
	}
	refreshTTL := opts.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = runtime.DefaultRefreshTTL
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
		authLimitedOps:    make(map[string]struct{}),
	}
	o.trans = newHTTPTransport(opts)
	return o
}

func newHTTPTransport(opts Options) Transport {
	return NewTransport(TransportOptions{
		Addr:           opts.Addr,
		Logger:         opts.Logger,
		APIPrefix:      opts.APIPrefix,
		StaticFS:       opts.StaticFS,
		AllowedOrigins: opts.AllowedOrigins,
	})
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := o.trans.Shutdown(ctx); err != nil {
		o.opts.Logger.Error("HTTP transport shutdown failed during restart", zap.Error(err))
	}
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

var _ runtime.Interface = (*Interface)(nil)

func (o *Interface) registerRoutes() {
	o.installDispatcherRegistrations()
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
		action := &cookieAction{}
		ctx = context.WithValue(ctx, cookieActionKey, action)
		ctx = runtime.ContextWithAuditTransport(ctx, req.ClientIP, req.UserAgent, req.RequestID)
		ctx = runtimecontext.ContextWithTraceID(ctx, req.RequestID)
		resp, err = chain(ctx, req)

		if action.SetToken != "" {
			resp.Cookies = append(resp.Cookies, newSessionCookie(o.cookieCfg.SessionName, action.SetToken, o.cookieCfg.SessionPath, o.sessionTTL, o.cookieCfg))
		} else if action.Clear {
			resp.Cookies = append(resp.Cookies, clearSessionCookie(o.cookieCfg.SessionName, o.cookieCfg.SessionPath, o.cookieCfg))
		}
		return
	}
}
