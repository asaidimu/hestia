package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/asaidimu/hestia/core"
)

//go:embed static
var staticFiles embed.FS

func main() {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}

	tmpDir, err := os.MkdirTemp("", "hestia-docs-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	app, err := hestia.Setup(hestia.SetupConfig{
		Port:              port,
		DataDir:           tmpDir,
		DBPath:            ":memory:",
		SessionSecret:     "docs-secret-do-not-use-in-production",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		StaticFS:          staticFS,
		APIPrefix:         "/api",
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
