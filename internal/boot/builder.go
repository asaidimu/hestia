package boot

import (
	"context"
	"fmt"
	"os"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/app"
	"github.com/asaidimu/hestia/internal/interface/api"
	"github.com/asaidimu/hestia/internal/interface/cli"
	"github.com/asaidimu/hestia/migrations"
)

func BuildApp(cfg *core.Config, opts app.Options) (*Application, *app.SystemModule, error) {
	application := Create(cfg)

	if opts.Logger == nil {
		opts.Logger = application.Loggers.File
	}

	if err := migrations.Apply(context.Background(), application.Persistence()); err != nil {
		application.Close()
		return nil, nil, fmt.Errorf("migrations: %w", err)
	}

	mod := app.New(cfg, application.Dispatcher(), opts)

	if err := application.RegisterModules(mod); err != nil {
		application.Close()
		return nil, nil, err
	}

	return application, mod, nil
}

func BuildInterfaces(a *Application, mod *app.SystemModule, version string) (*api.Interface, *cli.Interface) {
	chain := mod.DispatcherChain(a.Dispatcher())

	rpcOrch := api.New(api.Options{
		Dispatcher:          chain,
		InternalDispatcher:  a.Dispatcher(),
		CredentialsProvider: mod.CredentialsProvider(),
		Logger:              a.Loggers.File,
		Addr:                a.Config.Port,
		Registrations:       a.Registrations,
		CookieConfig:        a.Config.CookieConfig,
		AccessTokenTTL:      a.Config.AccessTokenTTL,
		RefreshTokenTTL:     a.Config.RefreshTokenTTL,
		APIPrefix:           a.Config.APIPrefix,
		StaticFS:            a.Config.StaticFS,
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
