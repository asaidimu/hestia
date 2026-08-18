package main

import (
	"fmt"
	"os"

	hestia "github.com/asaidimu/hestia/core"
)

var version = "dev"

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		Version:       version,
		SessionSecret: "my-test-secret",
		// SelfUpdate: &runtime.SelfUpdateConfig{
		// 	Provider: &updater.Github{
		// 		Owner:       "your-org",
		// 		Repo:        "your-repo",
		// 		AssetPattern: "your-app-{{ .version }}-{{ .os }}_{{ .arch }}",
		// 	},
		// 	CheckSchedule: "@daily",
		// },
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting on :9090...")
	if err := app.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Start failed: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	select {}
}
