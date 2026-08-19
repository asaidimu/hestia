package main

import (
	"log"

	hestia "github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/utils/wails"

	wailsruntime "github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		ProjectName:       "wails-demo-app",
		ForceBootstrapped: true,
		AdminEmail:        "admin@test.local",
		AdminPassword:     "password123",
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
		Title:  "Hestia Wails Test",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
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
