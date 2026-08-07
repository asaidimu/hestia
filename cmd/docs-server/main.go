package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/interface/cli"
	httpapi "github.com/asaidimu/hestia/core/interface/http"
	"github.com/asaidimu/hestia/core/runtime"
)

//go:embed static
var staticFiles embed.FS

func main() {
	port := 8080
	// Allow PORT env override (LoadConfig in runtime also handles this)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	tmpDir, err := os.MkdirTemp("", "hestia-docs-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := hestia.Setup(hestia.SetupConfig{
		DataDir:           tmpDir,
		DBPath:            ":memory:",
		SessionSecret:     "docs-secret-do-not-use-in-production",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		BuildInterfaces: func(app *hestia.Application) []runtime.Interface {
			return []runtime.Interface{
				app.NewHTTPInterface(httpapi.Config{
					Port:      port,
					APIPrefix: "/api",
					StaticFS:  staticFS,
				}),
				app.NewCLIInterface(cli.Config{}),
			}
		},
	})
	if err != nil {
		panic(err)
	}
	defer app.Close()

	app.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	app.Shutdown(ctx)
}
