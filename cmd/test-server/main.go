package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/asaidimu/hestia/internal/app"
	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/interface/api"

	"github.com/asaidimu/hestia"
)

func main() {
	port := "8070"

	tmpDir, err := os.MkdirTemp("", "hestiav2-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &core.Config{
		Port:              ":" + port,
		DataDir:           tmpDir,
		BlobsDir:          filepath.Join(tmpDir, "blobs"),
		DBPath:            ":memory:",
		JWTSecret:         "test-secret-do-not-use-in-production",
		LogPath:           filepath.Join(tmpDir, "server.log"),
		LogMaxSize:        100,
		LogMaxAge:         30,
		LogMaxBackups:     5,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		ForceBootstrapped: true,
		CookieConfig: core.CookieConfig{
			Domain:       "",
			Secure:       false,
			HTTPOnly:     true,
			SameSite:     http.SameSiteStrictMode,
			AccessName:   "access_token",
			AccessPath:   "/",
			RefreshName:  "refresh_token",
			RefreshPath:  "/api/auth/session",
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

	chain := systemMod.DispatcherChain(application.Dispatcher())
	rpcOrch := api.New(api.Options{
		Dispatcher:         chain,
		InternalDispatcher: application.Dispatcher(),
		Logger:             application.Loggers.File,
		Addr:               cfg.Port,
		Registrations:      application.Registrations,
		CookieConfig:       cfg.CookieConfig,
		AccessTokenTTL:     cfg.AccessTokenTTL,
		RefreshTokenTTL:    cfg.RefreshTokenTTL,
	})
	application.AddOrchestrator(rpcOrch)

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
