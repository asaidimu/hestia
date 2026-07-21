package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/identity"
	"github.com/asaidimu/hestia/core/interface/api"
)

func main() {
	port := "8070"

	tmpDir, err := os.MkdirTemp("", "hestiav2-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &runtime.Config{
		Port:              ":" + port,
		DataDir:           tmpDir,
		BlobsDir:          filepath.Join(tmpDir, "blobs"),
		DBPath:            ":memory:",
		SessionSecret:     "test-secret-do-not-use-in-production",
		LogPath:           filepath.Join(tmpDir, "server.log"),
		LogMaxSize:        100,
		LogMaxAge:         30,
		LogMaxBackups:     5,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		ForceBootstrapped: true,
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
		Middlewares: []hestia.Middleware{
			func(ctx context.Context, req api.Request, next api.HandlerFunc) (api.Response, error) {
				claims := &identity.Claims{
					UserID:    "auth_disabled",
					Email:     "admin@test.local",
					Scopes:    []string{"administrator"},
					TokenType: "system",
				}
				ctx = identity.ContextWithClaims(ctx, claims)
				ctx = runtime.ContextWithAuditIdentity(ctx, claims.UserID, runtime.ActorTypeUser, runtime.AuthMethodPassword)
				return next(ctx, req)
			},
		},
	})
	if err != nil {
		panic(err)
	}
	if err := app.Start(); err != nil {
		panic(err)
	}
	defer app.Close()

	fmt.Println(port)
	os.Stdout.Sync()

	select {}
}
