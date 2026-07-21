package main

import (
	// "embed"
	"log"
	"path/filepath"
	"runtime"

	hestia "github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/pkg/dispatch"
	// hr "github.com/asaidimu/hestia/core/runtime"
	"github.com/joho/godotenv"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go: embed frontend/dist
// var assets embed.FS

func init() {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	godotenv.Load(filepath.Join(dir, ".env"))
}

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		ForceBootstrapped: true,
		// Config: &hr.Config{
		// 	AdminEmail:    "admin@test.local",
		// 	AdminPassword: "password123",
		// },
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Start(); err != nil {
		panic(err)
	}

	defer app.Close()

	err = wails.Run(&options.App{
		Title:  "Hestia Wails Test",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			// Assets: assets,
		},
		Bind: []any{
			dispatch.New(app.Dispatcher()),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
