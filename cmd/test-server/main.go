package main

import (
	"context"
	"fmt"
	"os"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/identity"
	"github.com/asaidimu/hestia/core/interface/api"
	"github.com/asaidimu/hestia/core/runtime"
)

func main() {
	tmpDir, err := os.MkdirTemp("", "hestiav2-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := hestia.Setup(hestia.SetupConfig{
		Port:              8070,
		DataDir:           tmpDir,
		DBPath:            ":memory:",
		SessionSecret:     "test-secret-do-not-use-in-production",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
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

	fmt.Println("8070")
	os.Stdout.Sync()

	select {}
}
