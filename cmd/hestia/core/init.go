package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmdMain string

func init() {
	InitCmd.Flags().StringVar(&initCmdMain, "cmd", "./main.go", "Path to the entry point")
}

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise a new hestia project",
	Long: `Creates hestia.json, generates a stub module registry, writes an entry point, and runs go mod tidy.

Flags:
  --cmd  path to the entry point (default ./main.go)`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat("hestia.json"); err == nil {
			fmt.Println("hestia.json already exists")
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			dir = "."
		}

		modPath := detectModulePath(dir)
		if modPath == "" {
			fmt.Fprintln(os.Stderr, "Cannot detect module path: ensure go.mod exists in this directory or a parent directory")
			os.Exit(1)
		}

		cfg := Config{
			Module:        modPath,
			ModuleSources: []string{"module"},
			ModuleTarget:  "module",
		}
		writeConfig(dir, cfg)

		// Seed global vars so helpers can use them
		rootDir = dir
		modulePath = modPath
		moduleSources = cfg.ModuleSources
		moduleTarget = cfg.ModuleTarget
		autogenTarget = cfg.AutogenTarget
		if autogenTarget == "" {
			autogenTarget = "internal/autogen"
		}

		// Create module source directory
		for _, src := range moduleSources {
			srcDir := filepath.Join(rootDir, src)
			os.MkdirAll(srcDir, 0755)
		}

		// Generate stub module registry
		genModuleRegistry()

		// Write entry point
		mainPath := filepath.Join(dir, initCmdMain)
		mainDir := filepath.Dir(mainPath)
		os.MkdirAll(mainDir, 0755)

		modName := modulePath
		if idx := strings.LastIndex(modName, "/"); idx >= 0 {
			modName = modName[idx+1:]
		}
		mainContent := fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	hestia "github.com/asaidimu/hestia/core"
	"%s/internal/autogen"
)

var version = "dev"

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		Version:      version,
		ProjectName:  %q,
		Modules: autogen.Modules(),
	})

	if err != nil {
		panic(err)
	}

	if err := app.Start(); err != nil {
		panic(err)
	}
	defer app.Close()

	os.Stdout.Sync()
	select {}
}
`, modulePath, modName)

		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", mainPath, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", mainPath)

		// Write minimal Makefile
		makefilePath := filepath.Join(dir, "Makefile")
		makefileContent := fmt.Sprintf(`VERSION ?= dev
PROJECT   := %s
LDFLAGS   := -ldflags '-X github.com/asaidimu/hestia/core/internal/boot.ProjectName=$(PROJECT) -X main.version=$(VERSION)'

.PHONY: build run clean

build:
	go build $(LDFLAGS) -o $(PROJECT) %s

run: build
	./$(PROJECT)

clean:
	rm -f $(PROJECT)
`, modName, initCmdMain)
		if err := os.WriteFile(makefilePath, []byte(makefileContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", makefilePath, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", makefilePath)

		// Write .env.example
		envPath := filepath.Join(dir, ".env.example")
		envContent := `# Session secret (required) — generate with: openssl rand -hex 32
SESSION_SECRET=change-me-to-a-random-secret

# Server port (default :8090)
PORT=:8090

# Cookie settings
COOKIE_SECURE=false
COOKIE_SAMESITE=lax
`
		if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", envPath, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", envPath)

		// Update .gitignore
		gitignorePath := filepath.Join(dir, ".gitignore")
		gitignoreEntry := ".env\n"
		existing, err := os.ReadFile(gitignorePath)
		if err != nil {
			// .gitignore doesn't exist — create it
			if err := os.WriteFile(gitignorePath, []byte(gitignoreEntry), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", gitignorePath, err)
				os.Exit(1)
			}
			fmt.Printf("Wrote %s\n", gitignorePath)
		} else if !strings.Contains(string(existing), ".env") {
			// .gitignore exists but missing .env entry
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", gitignorePath, err)
				os.Exit(1)
			}
			if _, err := f.WriteString(gitignoreEntry); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update %s: %v\n", gitignorePath, err)
				os.Exit(1)
			}
			f.Close()
			fmt.Printf("Updated %s\n", gitignorePath)
		} else {
			fmt.Printf("Skipped %s (already has .env entry)\n", gitignorePath)
		}

		// Run go mod tidy
		fmt.Println("Running go mod tidy...")
		tidy := exec.Command("go", "mod", "tidy")
		tidy.Stdout = os.Stdout
		tidy.Stderr = os.Stderr
		if err := tidy.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "go mod tidy failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Project initialised successfully.")
	},
}
