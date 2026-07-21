package main

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/runtime"
)

//go:embed static
var staticFiles embed.FS

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
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

	cfg := &runtime.Config{
		Port:              ":" + port,
		DataDir:           tmpDir,
		BlobsDir:          filepath.Join(tmpDir, "blobs"),
		DBPath:            ":memory:",
		SessionSecret:     "docs-secret-do-not-use-in-production",
		LogPath:           filepath.Join(tmpDir, "server.log"),
		LogMaxSize:        100,
		LogMaxAge:         30,
		LogMaxBackups:     5,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		ForceBootstrapped: true,
		StaticFS:          staticFS,
		APIPrefix:         "/api",
		CookieConfig: runtime.CookieConfig{
			Domain:      "",
			Secure:      false,
			HTTPOnly:    true,
			SameSite:    1,
			SessionName: "session",
			SessionPath: "/",
		},
	}

	app, err := hestia.Setup(hestia.SetupConfig{
		Config: cfg,
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
