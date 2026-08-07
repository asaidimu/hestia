package main

import (
	"context"
	"fmt"
	"os"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/interface/cli"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/runtime"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

func main() {
	tmpDir, err := os.MkdirTemp("", "hestiav2-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := hestia.Setup(hestia.SetupConfig{
		DataDir:           tmpDir,
		DBPath:            ":memory:",
		SessionSecret:     "test-secret-do-not-use-in-production",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		BuildInterfaces: func(app *hestia.Application) []runtime.Interface {
			return []runtime.Interface{
				app.NewHTTPInterface(httpapi.Config{
					Port: 8070,
					Middleware: []httpapi.Middleware{
						func(ctx context.Context, req httpapi.Request, next httpapi.HandlerFunc) (httpapi.Response, error) {
							claims := &abstract.Claims{
								UserID:    "auth_disabled",
								Email:     "admin@test.local",
								Scopes:    []string{"administrator"},
								TokenType: "system",
							}
							ctx = runtimecontext.ContextWithClaims(ctx, claims)
							ctx = runtime.ContextWithAuditIdentity(ctx, claims.UserID, auditdomain.ActorTypeUser, auditdomain.AuthMethodPassword)
							return next(ctx, req)
						},
					},
				}),
				app.NewCLIInterface(cli.Config{}),
			}
		},
	})
	if err != nil {
		panic(err)
	}
	if err := app.Start(); err != nil {
		panic(err)
	}
	defer app.Close()

	fmt.Println("8070")
	os.Stdout.Sync()

	select {}
}
