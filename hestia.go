package hestia

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/internal/app"
	"github.com/asaidimu/hestia/internal/abstract"
	"github.com/asaidimu/hestia/internal/boot"
	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/schema"
	"github.com/asaidimu/hestia/internal/interface/api"
	"github.com/asaidimu/hestia/internal/interface/cli"
)

// ---------------------------------------------------------------------------
// Re-exported abstractions (aliases so downstream modules import only "hestia")
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// SetupConfig — all knobs for building an application
// ---------------------------------------------------------------------------

type SetupConfig struct {
	Config  *core.Config
	Version string
	Modules []Module
	Options app.Options

	// Migrate applies user-defined migrations after hestia's built-in migrations.
	// Called with the initialized persistence layer before any module is set up.
	// Use to create your own collections and schemas.
	Migrate func(ctx context.Context, p base.Persistence) error

	// DispatcherHooks wraps the dispatcher chain with additional layers.
	// Applied after the default chain (Secure→Blob→AccessLog→Local).
	// Each hook receives and returns a Dispatcher.
	// Example: rate limiting, custom audit, request validation middleware.
	DispatcherHooks []func(abstract.Dispatcher) abstract.Dispatcher
}

// ---------------------------------------------------------------------------
// Setup — build app + register modules, return ready-to-use handles
// ---------------------------------------------------------------------------

func Setup(cfg SetupConfig) (*boot.Application, *app.SystemModule, error) {
	if cfg.Config == nil {
		var err error
		cfg.Config, err = boot.NewConfig()
		if err != nil {
			return nil, nil, err
		}
	}
	cfg.Options.DispatcherHooks = cfg.DispatcherHooks

	application, systemMod, err := boot.BuildApp(cfg.Config, cfg.Options)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Migrate != nil {
		if err := cfg.Migrate(context.Background(), application.Persistence()); err != nil {
			application.Close()
			return nil, nil, fmt.Errorf("user migrations: %w", err)
		}
	}
	for _, m := range cfg.Modules {
		if err := application.RegisterModules(m); err != nil {
			application.Close()
			return nil, nil, fmt.Errorf("register module %s: %w", m.Name(), err)
		}
	}
	return application, systemMod, nil
}

// ---------------------------------------------------------------------------
// Run — Setup + orchestration + signal lifecycle (blocking)
// ---------------------------------------------------------------------------

func Run(cfg SetupConfig) error {
	opts := cfg.Options

	var application *boot.Application
	var systemMod *app.SystemModule

	opts.OnBootstrapped = wrapCallback(opts.OnBootstrapped, func() { application.RestartAll(true) })
	opts.OnReset = wrapCallback(opts.OnReset, func() { application.Reset(cfg.Config, cfg.Version) })

	cfg.Options = opts

	application, systemMod, err := Setup(cfg)
	if err != nil {
		return err
	}
	defer application.Close()

	orch := BuildOrchestrators(application, systemMod, cfg.Version)
	application.AddOrchestrator(orch.RPC)
	application.AddOrchestrator(orch.CLI)

	PrintBootstrapStatus(application, systemMod)

	application.Start(systemMod.Bootstrapped())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return application.Shutdown(ctx)
}

func wrapCallback(user, fallback func()) func() {
	if user != nil {
		return user
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Orchestrators
// ---------------------------------------------------------------------------

type Orchestrators struct {
	RPC *api.Orchestrator
	CLI *cli.Orchestrator
}

func BuildOrchestrators(a *boot.Application, mod *app.SystemModule, version string) Orchestrators {
	rpc, cli := boot.BuildOrchestrators(a, mod, version)
	return Orchestrators{RPC: rpc, CLI: cli}
}

// ---------------------------------------------------------------------------
// Bootstrap status
// ---------------------------------------------------------------------------

func PrintBootstrapStatus(a *boot.Application, mod *app.SystemModule) {
	if !mod.Bootstrapped() && mod.EphemeralKey() != "" {
		a.Loggers.Stdout.Printf("\n  First-Time Setup\n\n")
		a.Loggers.Stdout.Printf("  Ephemeral API Key:  \n\n%s\n", mod.EphemeralKey())
		a.Loggers.Stdout.Printf("  Admin Email:        %s\n", mod.AdminEmail())
		a.Loggers.Stdout.Printf("\n")
		a.Loggers.Stdout.Printf("  Authenticate with X-API-Key header, then:\n")
		a.Loggers.Stdout.Printf("    PUT /api/bootstrap/password  { \"password\": \"...\", \"email\": \"...\" }\n")
		a.Loggers.Stdout.Printf("\n")
	}

	if mod.Bootstrapped() {
		a.Loggers.File.Info("System is bootstrapped — all routes available")
	} else {
		a.Loggers.File.Warn("System is NOT bootstrapped — only bootstrap routes available")
	}
}

var _ Module = (*app.SystemModule)(nil)
