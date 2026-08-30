package hestia

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/interface/cli"
	"github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/internal/boot"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/updates"
	"github.com/asaidimu/hestia/core/system/users/model"
)

func projectName(projectName string) string {
	if projectName != "" {
		return projectName
	}
	return "hestia"
}

type SystemModule interface {
	DispatcherChain(next abstract.Dispatcher) abstract.Dispatcher
	CredentialsProvider() abstract.CredentialsProvider
	Bootstrapped() bool
	AdminUserID() string
	AdminEmail() string
	UserModel() *model.SystemUsers
}

type Middleware = http.Middleware
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
	return dispatch.MustFromJSON(data)
}

type Application struct {
	inner *boot.Application
}

func (a *Application) Persistence() base.Persistence {
	return a.inner.Persistence()
}
func (a *Application) Dispatcher() abstract.Dispatcher { return a.inner.Dispatcher() }
func (a *Application) SystemModule() SystemModule {
	if a.inner.SystemModule() == nil {
		return nil
	}
	return a.inner.SystemModule()
}
func (a *Application) Registrations() []abstract.MessageRegistration { return a.inner.Registrations }
func (a *Application) RegisterModules(m ...Module) error             { return a.inner.RegisterModules(m...) }

func (a *Application) NewHTTPInterface(cfg http.Config) runtime.Interface {
	mod := a.inner.SystemModule()
	chain := mod.DispatcherChain(a.inner.Dispatcher())
	// Append access-log middleware so it runs closest to the handler
	// (after auth), capturing user_id, request_id, and timing.
	allMiddleware := append(cfg.Middleware, http.AccessLog(a.inner.Loggers.File))
	return http.New(http.Options{
		Dispatcher:          chain,
		InternalDispatcher:  a.inner.Dispatcher(),
		CredentialsProvider: mod.CredentialsProvider(),
		Logger:              a.inner.Loggers.File,
		Addr:                cfg.Addr(),
		Registrations:       a.inner.Registrations,
		CookieConfig:        cfg.CookieConfig,
		SessionTTL:          cfg.SessionTTL,
		IdleTTL:             cfg.IdleTTL,
		RefreshTTL:          cfg.RefreshTTL,
		APIPrefix:           cfg.APIPrefix,
		StaticFS:            cfg.StaticFS,
		UserModel:           mod.UserModel(),
		Middleware:          allMiddleware,
		AllowedOrigins:      cfg.AllowedOrigins,
		TrustedProxyHops:    cfg.TrustedProxyHops,
	})
}

func (a *Application) NewCLIInterface(cfg cli.Config) runtime.Interface {
	mod := a.inner.SystemModule()
	chain := mod.DispatcherChain(a.inner.Dispatcher())
	stdin := cfg.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	return cli.New(cli.Options{
		Dispatcher:  chain,
		Logger:      a.inner.Loggers.File,
		AdminUserID: mod.AdminUserID(),
		AdminEmail:  mod.AdminEmail(),
		Version:     cfg.Version,
		Stdin:       stdin,
		Stdout:      stdout,
	})
}
func (a *Application) Start() error {
	if err := a.SeedPolicies(); err != nil {
		return err
	}
	a.inner.Start()
	return nil
}
func (a *Application) Shutdown(ctx context.Context) error { return a.inner.Shutdown(ctx) }
func (a *Application) Close()                             { a.inner.Close() }
func (a *Application) SeedPolicies() error {
	if sysMod := a.inner.SystemModule(); sysMod != nil {
		return sysMod.SeedPolicies(context.Background())
	}
	return nil
}

type SetupConfig struct {
	SessionSecret     string
	DataDir           string
	DBPath            string
	LogPath           string
	BlobsDir          string
	ProjectName       string
	Version           string
	SelfUpdate        *runtime.SelfUpdateConfig
	BcryptCost        int
	SessionTTL        time.Duration
	IdleTTL           time.Duration
	RefreshTTL        time.Duration
	ForceBootstrapped bool
	LogMaxSize        int
	LogMaxAge         int
	LogMaxBackups     int
	AdminEmail        string
	AdminPassword     string
	Mailer            runtime.MailerConfig
	AppURL            string
	StaticFS          fs.FS

	Modules             []Module
	DispatcherChainFunc func(chain abstract.ChainEditor)
	Interfaces          []func(abstract.Dispatcher) runtime.Interface
	BuildInterfaces     func(app *Application, cfg ...*runtime.Config) []runtime.Interface

	OnBootstrapped func()
	OnReset        func()
	Migrate        func(ctx context.Context, p base.Persistence) error

	PersistenceFactory func(cfg *anansi.SetupConfig) (base.Persistence, error)

	Bootstrap *bool
	Logger    *zap.Logger
}

func (cfg SetupConfig) applyTo(conf *runtime.Config) {
	pn := projectName(cfg.ProjectName)

	if cfg.SessionSecret != "" {
		conf.SessionSecret = cfg.SessionSecret
	}
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
	if cfg.Mailer.SMTPHost != "" {
		conf.Mailer = cfg.Mailer
	}
	if cfg.AppURL != "" {
		conf.AppURL = cfg.AppURL
	}
	if cfg.PersistenceFactory != nil {
		conf.PersistenceFactory = cfg.PersistenceFactory
	}
	if cfg.Version != "" {
		conf.Version = cfg.Version
	}
	if cfg.SelfUpdate != nil {
		conf.SelfUpdate = cfg.SelfUpdate
	}
	if cfg.StaticFS != nil {
		conf.StaticFS = cfg.StaticFS
	}
}

func Setup(cfg SetupConfig) (*Application, error) {
	if cfg.ProjectName != "" {
		boot.ProjectName = cfg.ProjectName
	}

	pn := projectName(cfg.ProjectName)
	conf, err := runtime.LoadConfig(pn)
	if err != nil {
		return nil, err
	}
	cfg.applyTo(conf)
	if err := runtime.ApplyEnvOverrides(conf); err != nil {
		return nil, err
	}

	// Phase 1: wiring — no I/O
	application := boot.New(conf)
	appWrapper := &Application{inner: application}

	// Pre-boot self-update: consume a --perform-update launch (waiting for
	// the old process, swapping the executable) and remove leftover staged
	// binaries. Runs before migrations and before the CLI arg parse.
	if conf.SelfUpdate != nil {
		if err := updates.HandleStartup(conf.SelfUpdate, conf); err != nil {
			application.Close()
			return nil, err
		}
	}

	bootstrap := true
	if cfg.Bootstrap != nil {
		bootstrap = *cfg.Bootstrap
	}
	if !bootstrap {
		return appWrapper, nil
	}

	// Phase 2: boot — persistence, migrations, module hydration
	forceBootstrapped := cfg.ForceBootstrapped || conf.ForceBootstrapped

	opts := dispatch.SystemOptions{
		OnBootstrapped: func() {
			application.RestartAll(true)
			if cfg.OnBootstrapped != nil {
				cfg.OnBootstrapped()
			}
		},
		OnReset: func() {
			application.Reset(conf, conf.Version)
			if cfg.OnReset != nil {
				cfg.OnReset()
			}
		},
		ForceBootstrapped:   forceBootstrapped,
		Logger:              cfg.Logger,
		DispatcherChainFunc: cfg.DispatcherChainFunc,
	}

	if err := application.Boot(context.Background(), opts); err != nil {
		application.Close()
		return nil, err
	}

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

	if cfg.BuildInterfaces != nil {
		for _, i := range cfg.BuildInterfaces(appWrapper, conf) {
			application.AddInterface(i)
		}
	} else {
		application.AddInterface(appWrapper.NewHTTPInterface(http.ConfigFromRuntime(conf)))
		application.AddInterface(appWrapper.NewCLIInterface(cli.Config{Version: conf.Version}))
		for _, fn := range cfg.Interfaces {
			application.AddInterface(fn(application.Dispatcher()))
		}
	}

	return appWrapper, nil
}
