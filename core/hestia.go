package hestia

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/schema"
	"github.com/asaidimu/hestia/core/interface/api"
	"github.com/asaidimu/hestia/core/internal/boot"
)

func projectName(projectName string) string {
	if projectName != "" {
		return projectName
	}
	return "hestia"
}

// SystemModule provides access to system-level services.
// Use with caution — direct access bypasses normal API safety guarantees.
// Prefer Application-level methods when available.
type SystemModule interface {
	DispatcherChain(next runtime.Dispatcher) runtime.Dispatcher
	CredentialsProvider() abstract.CredentialsProvider
	Bootstrapped() bool
	AdminUserID() string
	AdminEmail() string
}

type Middleware = api.Middleware
type Module = abstract.Module
type Capability = abstract.Capability
type MessageRegistration = abstract.MessageRegistration
type MessageHandler = abstract.MessageHandler
type Input = abstract.Input
type Verb = abstract.Verb
type Message = abstract.Message
type Result = abstract.Result
type Blob = abstract.Blob
type Page = abstract.Page

const (
	Create Verb = abstract.Create
	Read        = abstract.Read
	Update      = abstract.Update
	Delete      = abstract.Delete
	Query       = abstract.Query
	Stream      = abstract.Stream
)

func MustFromJSON(data []byte) *definition.Schema {
	return schema.MustFromJSON(data)
}

type Application struct {
	inner *boot.Application
}

func (a *Application) Persistence() base.Persistence         { return a.inner.Persistence() }
func (a *Application) Dispatcher() runtime.Dispatcher        { return a.inner.Dispatcher() }
func (a *Application) SystemModule() SystemModule            { return a.inner.SystemModule() }
func (a *Application) Registrations() []abstract.MessageRegistration { return a.inner.Registrations }
func (a *Application) RegisterModules(m ...Module) error     { return a.inner.RegisterModules(m...) }
func (a *Application) Start() error {
	if sysMod := a.inner.SystemModule(); sysMod != nil {
		if err := sysMod.SeedPolicies(context.Background()); err != nil {
			return err
		}
	}
	a.inner.Start()
	return nil
}
func (a *Application) Shutdown(ctx context.Context) error    { return a.inner.Shutdown(ctx) }
func (a *Application) Close()                                { a.inner.Close() }
func (a *Application) SeedPolicies() error {
	if sysMod := a.inner.SystemModule(); sysMod != nil {
		return sysMod.SeedPolicies(context.Background())
	}
	return nil
}

type SetupConfig struct {
	// Port is the HTTP server port (0 means DefaultPort / 8090).
	Port int
	// SessionSecret is the signing key for JWT tokens.
	// Falls back to SESSION_SECRET / JWT_SECRET env vars.
	SessionSecret string
	// DataDir is the directory for DB, logs, and blob storage.
	// Falls back to XDG_DATA_HOME/<ProjectName> / ~/.local/share/<ProjectName>.
	DataDir  string
	DBPath   string
	LogPath  string
	BlobsDir string
	// ProjectName is used for the data directory sub-path and DB filename.
	ProjectName string
	// Version is baked into the binary at build time via -ldflags.
	Version string

	// BcryptCost is the cost factor for password hashing (default 12).
	BcryptCost int
	// SessionTTL is the absolute session lifetime (default 8h).
	SessionTTL time.Duration
	// IdleTTL is the max idle time before session expiry (default 30m).
	IdleTTL time.Duration
	// RefreshTTL is the idle threshold for cookie refresh (default 15m).
	RefreshTTL time.Duration
	// ForceBootstrapped skips the bootstrap flow.
	ForceBootstrapped bool

	// Log rotation (defaults: MaxSize=100MB, MaxAge=30d, MaxBackups=5).
	LogMaxSize    int
	LogMaxAge     int
	LogMaxBackups int

	// Admin credentials for the initial seed.
	AdminEmail    string
	AdminPassword string

	// APIPrefix is the URL prefix for all API routes (default "/api").
	APIPrefix string
	// StaticFS serves static files / SPA at the root path.
	StaticFS fs.FS

	// Cookie settings
	CookieDomain      string
	CookieSecure      *bool
	CookieHTTPOnly    *bool
	CookieSameSite    abstract.SameSite
	CookieSessionName string
	CookieSessionPath string

	// Modules registered with the application.
	Modules   []Module
	// Middlewares applied to every API request.
	Middlewares  []Middleware
	// DispatcherHooks wrap the dispatcher chain.
	DispatcherHooks []func(abstract.Dispatcher) abstract.Dispatcher
	// Interfaces register custom runtime interfaces.
	Interfaces []func(runtime.Dispatcher) runtime.Interface

	// OnBootstrapped is called after the system is bootstrapped.
	OnBootstrapped func()
	// OnReset is called after a full system reset.
	OnReset func()
	// Migrate is a user-provided migration function.
	Migrate func(ctx context.Context, p base.Persistence) error

	// PersistenceFactory gives full control over persistence setup.
	// Receives an anansi.SetupConfig and returns a base.Persistence.
	PersistenceFactory func(cfg *anansi.SetupConfig) (base.Persistence, error)

	// Flags to disable built-in interfaces.
	DisableRPC bool
	DisableCLI bool

	// Logger overrides the default zap logger.
	Logger *zap.Logger
}

func Setup(cfg SetupConfig) (*Application, error) {
	if cfg.ProjectName != "" {
		boot.ProjectName = cfg.ProjectName
	}

	conf, err := boot.NewConfig()
	if err != nil {
		return nil, err
	}

	if cfg.Port > 0 {
		conf.Port = cfg.Port
	}
	if cfg.SessionSecret != "" {
		conf.SessionSecret = cfg.SessionSecret
	}
	pn := projectName(cfg.ProjectName)
	if cfg.DataDir != "" {
		conf.DataDir = cfg.DataDir
		if cfg.DBPath == "" {
			conf.DBPath = filepath.Join(cfg.DataDir, pn+".db")
		}
		if cfg.LogPath == "" {
			conf.LogPath = filepath.Join(cfg.DataDir, "server.log")
		}
		if cfg.BlobsDir == "" {
			conf.BlobsDir = filepath.Join(cfg.DataDir, "blobs")
		}
	}
	if cfg.DBPath != "" {
		conf.DBPath = cfg.DBPath
	}
	if cfg.LogPath != "" {
		conf.LogPath = cfg.LogPath
	}
	if cfg.BlobsDir != "" {
		conf.BlobsDir = cfg.BlobsDir
	}
	if cfg.BcryptCost > 0 {
		conf.BcryptCost = cfg.BcryptCost
	}
	if cfg.SessionTTL > 0 {
		conf.SessionTTL = cfg.SessionTTL
	}
	if cfg.IdleTTL > 0 {
		conf.IdleTTL = cfg.IdleTTL
	}
	if cfg.RefreshTTL > 0 {
		conf.RefreshTTL = cfg.RefreshTTL
	}
	if cfg.LogMaxSize > 0 {
		conf.LogMaxSize = cfg.LogMaxSize
	}
	if cfg.LogMaxAge > 0 {
		conf.LogMaxAge = cfg.LogMaxAge
	}
	if cfg.LogMaxBackups > 0 {
		conf.LogMaxBackups = cfg.LogMaxBackups
	}
	if cfg.AdminEmail != "" {
		conf.AdminEmail = cfg.AdminEmail
	}
	if cfg.AdminPassword != "" {
		conf.AdminPassword = cfg.AdminPassword
	}
	if cfg.APIPrefix != "" {
		conf.APIPrefix = cfg.APIPrefix
	}
	if cfg.StaticFS != nil {
		conf.StaticFS = cfg.StaticFS
	}
	if cfg.PersistenceFactory != nil {
		conf.PersistenceFactory = cfg.PersistenceFactory
	}
	if cfg.CookieDomain != "" {
		conf.CookieConfig.Domain = cfg.CookieDomain
	}
	if cfg.CookieSecure != nil {
		conf.CookieConfig.Secure = *cfg.CookieSecure
	}
	if cfg.CookieHTTPOnly != nil {
		conf.CookieConfig.HTTPOnly = *cfg.CookieHTTPOnly
	}
	if cfg.CookieSameSite != 0 {
		conf.CookieConfig.SameSite = cfg.CookieSameSite
	}
	if cfg.CookieSessionName != "" {
		conf.CookieConfig.SessionName = cfg.CookieSessionName
	}
	if cfg.CookieSessionPath != "" {
		conf.CookieConfig.SessionPath = cfg.CookieSessionPath
	}

	if conf.SessionSecret == "" {
		return nil, fmt.Errorf("SessionSecret is required: set it via SetupConfig.SessionSecret, SESSION_SECRET, or JWT_SECRET env var")
	}

	forceBootstrapped := cfg.ForceBootstrapped || conf.ForceBootstrapped

	var app *boot.Application
	wrapBootstrapped := cfg.OnBootstrapped
	wrapReset := cfg.OnReset

	opts := abstract.SystemOptions{
		OnBootstrapped: func() {
			if app != nil {
				app.RestartAll(true)
			}
			if wrapBootstrapped != nil {
				wrapBootstrapped()
			}
		},
		OnReset: func() {
			if app != nil {
				app.Reset(conf, cfg.Version)
			}
			if wrapReset != nil {
				wrapReset()
			}
		},
		ForceBootstrapped: forceBootstrapped,
		Logger:            cfg.Logger,
		DispatcherHooks:   cfg.DispatcherHooks,
	}

	application, err := boot.BuildApp(conf, opts)
	if err != nil {
		return nil, err
	}
	app = application

	if cfg.Migrate != nil {
		if err := cfg.Migrate(context.Background(), application.Persistence()); err != nil {
			application.Close()
			return nil, fmt.Errorf("user migrations: %w", err)
		}
	}
	for _, m := range cfg.Modules {
		if err := application.RegisterModules(m); err != nil {
			application.Close()
			return nil, fmt.Errorf("register module %s: %w", m.Name(), err)
		}
	}

	if !cfg.DisableRPC || !cfg.DisableCLI {
		rpc, cli := boot.BuildInterfaces(application, cfg.Version, cfg.Middlewares)
		if !cfg.DisableRPC {
			application.AddInterface(rpc)
		}
		if !cfg.DisableCLI {
			application.AddInterface(cli)
		}
	}
	for _, fn := range cfg.Interfaces {
		application.AddInterface(fn(application.Dispatcher()))
	}

	return &Application{inner: application}, nil
}
