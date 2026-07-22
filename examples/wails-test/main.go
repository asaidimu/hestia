package main

import (
	// "embed"
	"log"
	"path/filepath"
	"runtime"

	hestia "github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/utils/wails"

	// hr "github.com/asaidimu/hestia/core/runtime"
	"github.com/joho/godotenv"

	wailsruntime "github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// go:embed all:frontend/dist
// var assets embed.FS

func init() {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	godotenv.Load(filepath.Join(dir, ".env"))
}

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		ProjectName: "wails-demo-app",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
		DisableRPC:        true,
		DisableCLI:        true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	mod := app.SystemModule()
	adapter := wails.New(wails.Options{
		Dispatcher:    mod.DispatcherChain(app.Dispatcher()),
		Internal:      app.Dispatcher(),
		CredProvider:  mod.CredentialsProvider(),
		Registrations: app.Registrations(),
	})

	err = wailsruntime.Run(&options.App{
		Title:       "Hestia Wails Test",
		Width:       1024,
		Height:      768,
		AssetServer: &assetserver.Options{
			// Assets: assets,
			Handler: adapter.Handler(),
		},
		Bind: []any{
			adapter,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
