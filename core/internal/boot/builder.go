package boot

import (
	"context"
	"os"

	"github.com/asaidimu/hestia/core/interface/cli"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func BuildApp(cfg *runtime.Config, opts dispatch.SystemOptions) (*Application, error) {
	application := New(cfg)

	if err := application.Boot(context.Background(), opts); err != nil {
		application.Close()
		return nil, err
	}

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
