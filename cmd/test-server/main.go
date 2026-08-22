package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/interface/cli"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/runtime"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

// version is set via -ldflags "-X main.version=..." (used by the self-update
// E2E to bake distinct releases); defaults to dev.
var version = "dev"

// serverPort lets the E2E run an isolated instance on a free port instead of
// colliding with the always-on :8070 test server.
func serverPort() int {
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 8070
}

func main() {
	tmpDir, err := os.MkdirTemp("", "hestiav2-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := hestia.Setup(hestia.SetupConfig{
		Version:           version,
		DataDir:           tmpDir,
		DBPath:            ":memory:",
		SessionSecret:     "test-secret-do-not-use-in-production",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		BuildInterfaces: func(app *hestia.Application, cfg ...*runtime.Config) []runtime.Interface {
			return []runtime.Interface{
				app.NewHTTPInterface(httpapi.Config{
					Port: serverPort(),
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

	// Dev-only pprof sidecar for latency/CPU investigations.
	if p := os.Getenv("PPROF_PORT"); p != "" {
		go func() {
			_ = http.ListenAndServe("localhost:"+p, nil)
		}()
	}

	fmt.Println("8070")
	os.Stdout.Sync()

	select {}
}
