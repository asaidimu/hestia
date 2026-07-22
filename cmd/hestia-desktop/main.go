package main

import (
	"log"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/utils/wails"

	wailsruntime "github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		DisableRPC: true,
		DisableCLI: true,
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
		Title:  "Hestia Desktop",
		Width:  1200,
		Height: 800,
		Bind: []any{
			adapter,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
