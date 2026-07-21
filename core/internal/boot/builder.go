package boot

import (
	"context"
	"fmt"
	"os"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/internal/feature"
	"github.com/asaidimu/hestia/core/interface/api"
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

func BuildInterfaces(a *Application, version string, middlewares []api.Middleware) (*api.Interface, *cli.Interface) {
	mod := a.SystemModule()
	chain := mod.DispatcherChain(a.Dispatcher())

	rpcOrch := api.New(api.Options{
		Dispatcher:          chain,
		InternalDispatcher:  a.Dispatcher(),
		CredentialsProvider: mod.CredentialsProvider(),
		Logger:              a.Loggers.File,
		Addr:                a.Config.Port,
		Registrations:       a.Registrations,
		CookieConfig:        a.Config.CookieConfig,
		SessionTTL:          a.Config.SessionTTL,
		IdleTTL:             a.Config.IdleTTL,
		RefreshTTL:          a.Config.RefreshTTL,
		APIPrefix:           a.Config.APIPrefix,
		StaticFS:            a.Config.StaticFS,
		UserModel:           mod.UserModel(),
		Middleware:          middlewares,
	})

	cliOrch := cli.New(cli.Options{
		Dispatcher:  chain,
		Logger:      a.Loggers.File,
		AdminUserID: mod.AdminUserID(),
		AdminEmail:  mod.AdminEmail(),
		Version:     version,
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
	})

	return rpcOrch, cliOrch
}
