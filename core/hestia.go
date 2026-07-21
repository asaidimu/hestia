package hestia

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/schema"
	"github.com/asaidimu/hestia/core/interface/api"
	"github.com/asaidimu/hestia/core/internal/boot"
)

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
	Config  *runtime.Config
	Version string
	Modules []Module

	OnBootstrapped    func()
	OnReset           func()
	ForceBootstrapped bool
	Logger            *zap.Logger

	ProjectName string

	Migrate func(ctx context.Context, p base.Persistence) error

	DispatcherHooks []func(abstract.Dispatcher) abstract.Dispatcher

	DisableRPC bool
	DisableCLI bool
	Interfaces []func(runtime.Dispatcher) runtime.Interface
	Middlewares  []Middleware
}

func Setup(cfg SetupConfig) (*Application, error) {
	if cfg.Config == nil {
		if cfg.ProjectName != "" {
			boot.ProjectName = cfg.ProjectName
		}
		var err error
		cfg.Config, err = boot.NewConfig()
		if err != nil {
			return nil, err
		}
	}

	forceBootstrapped := cfg.ForceBootstrapped
	if !forceBootstrapped && cfg.Config != nil {
		forceBootstrapped = cfg.Config.ForceBootstrapped
	}

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
				app.Reset(cfg.Config, cfg.Version)
			}
			if wrapReset != nil {
				wrapReset()
			}
		},
		ForceBootstrapped: forceBootstrapped,
		Logger:            cfg.Logger,
		DispatcherHooks:   cfg.DispatcherHooks,
	}

	application, err := boot.BuildApp(cfg.Config, opts)
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
