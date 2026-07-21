package main

import (
	"log"

	"github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/pkg/dispatch"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{})
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	err = wails.Run(&options.App{
		Title:  "Hestia Desktop",
		Width:  1200,
		Height: 800,
		Bind: []any{
			dispatch.New(app.Dispatcher()),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
