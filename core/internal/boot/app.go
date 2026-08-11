package boot

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/interface/cli"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/feature"
	"github.com/asaidimu/hestia/core/internal/migrations"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func validateMessageName(name string) error {
	parts := strings.Split(name, ":")
	if len(parts) != 4 {
		return fmt.Errorf("message name %q has %d segments, want 4 (module:feature:scope:action)", name, len(parts))
	}
	return nil
}

type Application struct {
	Config             *runtime.Config
	Loggers            *Loggers
	PersistenceManager *PersistenceManager
	Disp               *runtime.LocalDispatcher
	Interfaces         []runtime.Interface
	Registrations      []abstract.MessageRegistration
	Modules            []abstract.Module
	systemMod          *feature.SystemModule
}

func (a *Application) SystemModule() *feature.SystemModule { return a.systemMod }

func (a *Application) SetSystemModule(m *feature.SystemModule) { a.systemMod = m }

func New(cfg *runtime.Config) *Application {
	loggers := NewLoggers(cfg)
	loggers.Stdout.Banner()

	return &Application{
		Config:  cfg,
		Loggers: loggers,
		Disp:    runtime.NewLocalDispatcher(),
	}
}

func (a *Application) Boot(ctx context.Context, opts dispatch.SystemOptions) error {
	if opts.Logger == nil {
		opts.Logger = a.Loggers.File
	}

	pm, err := NewPersistenceManager(a.Config, a.Loggers.File)
	if err != nil {
		return fmt.Errorf("persistence manager: %w", err)
	}
	a.PersistenceManager = pm

	if err := migrations.Apply(ctx, a.Persistence()); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	mod := feature.New(a.Config, a.Dispatcher(), opts)
	if err := a.RegisterModules(mod); err != nil {
		return err
	}
	a.SetSystemModule(mod)
	return nil
}

// Create is kept for backwards compatibility. Prefer New + Boot.
func Create(cfg *runtime.Config) *Application {
	app := New(cfg)
	if err := app.Boot(context.Background(), dispatch.SystemOptions{Logger: app.Loggers.File}); err != nil {
		app.Loggers.File.Fatal("Failed to boot application", zap.Error(err))
	}
	return app
}

func (a *Application) Persistence() base.Persistence {
	if a.PersistenceManager == nil {
		return nil
	}
	return a.PersistenceManager.Persistence()
}

func (a *Application) Dispatcher() *runtime.LocalDispatcher {
	return a.Disp
}

func (a *Application) RegisterModules(modules ...abstract.Module) error {
	ctx := context.Background()

	// Topological sort by Dependencies using Kahn's algorithm
	sorted, err := topoSort(modules)
	if err != nil {
		return fmt.Errorf("module dependency sort: %w", err)
	}
	a.Modules = append(a.Modules, sorted...)

	for _, mod := range sorted {
		if err := mod.Setup(ctx, a.PersistenceManager.Persistence()); err != nil {
			return fmt.Errorf("module %s setup: %w", mod.Name(), err)
		}
		for _, cap := range mod.Capabilities() {
			for _, mr := range cap.Messages {
				if err := validateMessageName(mr.Name); err != nil {
					a.Loggers.File.Warn("Message name grammar violation", zap.String("module", mod.Name()), zap.String("name", mr.Name), zap.Error(err))
				}

				if err := a.Disp.RegisterHandler(mr.Name, mr.Handler, abstract.HandlerInfo{
					Name:          mr.Name,
					Description:   mr.Description,
					Enabled:       mr.Enabled,
					BootstrapSafe: mr.BootstrapSafe,
				}); err != nil {
					a.Loggers.File.Warn("Failed to register handler", zap.String("module", mod.Name()), zap.String("name", mr.Name), zap.Error(err))
				}

				if mr.Input.Schema != nil {
					if issues, ok := dispatch.ValidateInputSchema(mr.Input.Schema); !ok && len(issues) > 0 {
						return fmt.Errorf("module %s message %q input schema validation failed: %w", mod.Name(), mr.Name, common.NewSystemError("invalid input schema").WithIssues(issues))
					}
				}

				a.Registrations = append(a.Registrations, mr)
			}
		}
	}

	return nil
}

// topoSort returns modules sorted by Dependencies using Kahn's algorithm.
// Returns an error if a cycle or missing dependency is detected.
func topoSort(modules []abstract.Module) ([]abstract.Module, error) {
	byName := make(map[string]abstract.Module, len(modules))
	inDegree := make(map[string]int, len(modules))
	deps := make(map[string][]string, len(modules))

	for _, m := range modules {
		byName[m.Name()] = m
		inDegree[m.Name()] = 0
	}

	for _, m := range modules {
		for _, dep := range m.Dependencies() {
			deps[dep] = append(deps[dep], m.Name())
		}
	}

	for _, m := range modules {
		for _, dep := range m.Dependencies() {
			if _, ok := byName[dep]; ok {
				inDegree[m.Name()]++
			}
		}
	}

	var queue []string
	for _, m := range modules {
		if inDegree[m.Name()] == 0 {
			queue = append(queue, m.Name())
		}
	}

	var sorted []abstract.Module
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if m, ok := byName[name]; ok {
			sorted = append(sorted, m)
		}
		for _, dependent := range deps[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(modules) {
		unresolved := make([]string, 0)
		for _, m := range modules {
			found := false
			for _, s := range sorted {
				if s.Name() == m.Name() {
					found = true
					break
				}
			}
			if !found {
				unresolved = append(unresolved, m.Name())
			}
		}
		return nil, fmt.Errorf("circular or missing dependency detected for modules: %v", unresolved)
	}

	return sorted, nil
}

func (a *Application) AddInterface(o runtime.Interface) {
	a.Interfaces = append(a.Interfaces, o)
}

func (a *Application) Start() {
	ctx := context.Background()
	for _, m := range a.Modules {
		if err := m.Start(ctx); err != nil {
			a.Loggers.File.Error("module start failed", zap.String("module", m.Name()), zap.Error(err))
		}
	}

	bootstrapped := false
	if a.systemMod != nil {
		bootstrapped = a.systemMod.Bootstrapped()
	}
	for _, i := range a.Interfaces {
		i.Start(bootstrapped)
	}
}

func (a *Application) RestartAll(bootstrapped bool) {
	for _, o := range a.Interfaces {
		o.Restart(bootstrapped)
	}
}

func (a *Application) Shutdown(ctx context.Context) error {
	var lastErr error

	// Stop modules in reverse dependency order
	for i := len(a.Modules) - 1; i >= 0; i-- {
		if err := a.Modules[i].Stop(ctx); err != nil {
			a.Loggers.File.Error("module stop failed", zap.String("module", a.Modules[i].Name()), zap.Error(err))
			lastErr = err
		}
	}

	for _, o := range a.Interfaces {
		if err := o.Shutdown(ctx); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (a *Application) Close() {
	_ = a.Loggers.File.Sync()
	if a.PersistenceManager != nil {
		_ = a.PersistenceManager.Close()
	}
	_ = a.Loggers.Close()
}

func (a *Application) Reset(cfg *runtime.Config, version string) {
	_ = a.Loggers.File.Sync()
	if a.PersistenceManager != nil {
		_ = a.PersistenceManager.Close()
	}
	_ = a.Loggers.Close()

	if err := os.RemoveAll(a.Config.DataDir); err != nil {
		a.Loggers.File.Error("Reset: failed to remove data dir", zap.Error(err))
		return
	}
	if err := os.MkdirAll(a.Config.DataDir, 0700); err != nil {
		a.Loggers.File.Error("Reset: failed to create data dir", zap.Error(err))
		return
	}

	newApp := New(cfg)
	a.Config = newApp.Config
	a.Loggers = newApp.Loggers
	a.Disp = newApp.Disp

	pm, err := NewPersistenceManager(cfg, a.Loggers.File)
	if err != nil {
		a.Loggers.File.Fatal("Failed to setup persistence manager on reset", zap.Error(err))
	}
	a.PersistenceManager = pm

	if err := migrations.Apply(context.Background(), a.PersistenceManager.Persistence()); err != nil {
		a.Loggers.File.Fatal("Failed to apply migrations on reset", zap.Error(err))
	}

	mod := feature.New(cfg, a.Dispatcher(), dispatch.SystemOptions{
		Logger:            a.Loggers.File,
		AdminEmail:        cfg.AdminEmail,
		AdminPassword:     cfg.AdminPassword,
		ForceBootstrapped: cfg.ForceBootstrapped,
		OnBootstrapped: func() {
			a.RestartAll(true)
		},
	})
	a.RegisterModules(mod)
	a.SetSystemModule(mod)

	rpcIface, cliIface := BuildInterfaces(a, version, httpapi.ConfigFromRuntime(cfg), cli.Config{})
	a.Interfaces = nil
	a.AddInterface(rpcIface)
	a.AddInterface(cliIface)

	a.Loggers.File.Info("Reset: restarting…")
	a.RestartAll(mod.Bootstrapped())
}
