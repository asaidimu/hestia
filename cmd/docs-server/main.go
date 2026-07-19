package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/asaidimu/hestia"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/internal/app"
	_ "github.com/asaidimu/hestia/internal/boot"
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

	cfg := &core.Config{
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
		CookieConfig: core.CookieConfig{
			Domain:      "",
			Secure:      false,
			HTTPOnly:    true,
			SameSite:    1,
			SessionName: "session",
			SessionPath: "/",
		},
	}

	application, systemMod, err := hestia.Setup(hestia.SetupConfig{
		Config: cfg,
		Options: app.Options{
			ForceBootstrapped: true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer application.Close()

	if err := systemMod.SeedPolicies(context.Background()); err != nil {
		panic(err)
	}

	ifaces := hestia.BuildInterfaces(application, systemMod, "")
	application.AddInterface(ifaces.RPC)

	application.Start(systemMod.Bootstrapped())

	fmt.Println(port)
	os.Stdout.Sync()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	application.Shutdown(ctx)
}
