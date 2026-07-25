package boot

import (
	"context"
	"fmt"
	"os"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/internal/feature"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/interface/cli"
	"github.com/asaidimu/hestia/core/migrations"
)

func BuildApp(cfg *runtime.Config, opts abstract.SystemOptions) (*Application, error) {
	application := Create(cfg)

	if opts.Logger == nil {
		opts.Logger = application.Loggers.File
	}

	if err := migrations.Apply(context.Background(), application.Persistence()); err != nil {
		application.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	mod := feature.New(cfg, application.Dispatcher(), opts)

	if err := application.RegisterModules(mod); err != nil {
		application.Close()
		return nil, err
	}

	application.SetSystemModule(mod)
	return application, nil
}

func BuildInterfaces(a *Application, version string, apiCfg httpapi.Config, cliCfg cli.Config) (*httpapi.Interface, *cli.Interface) {
	mod := a.SystemModule()
	chain := mod.DispatcherChain(a.Dispatcher())

	rpcOrch := httpapi.New(httpapi.Options{
		Dispatcher:          chain,
		InternalDispatcher:  a.Dispatcher(),
		CredentialsProvider: mod.CredentialsProvider(),
		Logger:              a.Loggers.File,
		Addr:                apiCfg.Addr(),
		Registrations:       a.Registrations,
		CookieConfig:        apiCfg.CookieConfig,
		SessionTTL:          apiCfg.SessionTTL,
		IdleTTL:             apiCfg.IdleTTL,
		RefreshTTL:          apiCfg.RefreshTTL,
		APIPrefix:           apiCfg.APIPrefix,
		StaticFS:            apiCfg.StaticFS,
		UserModel:           mod.UserModel(),
		Middleware:          apiCfg.Middleware,
		AllowedOrigins:      apiCfg.AllowedOrigins,
	})

	stdin := cliCfg.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := cliCfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	cliOrch := cli.New(cli.Options{
		Dispatcher:  chain,
		Logger:      a.Loggers.File,
		AdminUserID: mod.AdminUserID(),
		AdminEmail:  mod.AdminEmail(),
		Version:     version,
		Stdin:       stdin,
		Stdout:      stdout,
	})

	return rpcOrch, cliOrch
}
