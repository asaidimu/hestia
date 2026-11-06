package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// @note #tool-20261029-004 todo P2 #tooling,#scaffold : init does not make the project anansi-ready
// @author claude
// @see #tool-20261029-001
// @see #tool-20261029-006
//
// init writes main.go, the Makefile, .env, .gitignore and hestia.json, but no anansi.json, schemas directory or lockfile. A fresh project cannot run migrate or codegen until the user runs `anansi scaffold --existing` separately, and anansi's default glob would not match hestia's core_dir/modules_dir layout.
// Fix: once scaffold is importable (#tool-20261029-001), have init create anansi.json non-destructively, with the glob derived from core_dir and modules_dir, and add sync/check targets to the generated Makefile. Config overlap is #tool-20261029-006.

var InitCmd = &cobra.Command{
	Use:   "init [module-path] [root]",
	Short: "Initialise a new hestia project",
	Long: `Creates a new hestia project: go.mod (if missing), hestia.json,
core/ and modules/ directories, a command entry point, the generated
module registry, and runs go mod tidy.

Examples:
  hestia init github.com/user/myapp
  hestia init github.com/user/myapp ./src`,
	Args: cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			dir = "."
		}
		if len(args) >= 2 {
			dir = args[1]
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("create %s: %w", dir, err)
			}
		}

		if _, err := os.Stat(filepath.Join(dir, "hestia.json")); err == nil {
			fmt.Println("hestia.json already exists")
			return nil
		}

		modPath := ""
		if len(args) >= 1 {
			modPath = args[0]
		} else {
			modPath = detectModulePath(dir)
		}
		if modPath == "" {
			return fmt.Errorf("cannot detect module path: pass it explicitly (hestia init <module-path>) or ensure go.mod exists")
		}

		// go.mod first so detectModulePath and go mod tidy work.
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); os.IsNotExist(err) {
			fmt.Println("Running go mod init...")
			initCmd := exec.Command("go", "mod", "init", modPath)
			initCmd.Dir = dir
			initCmd.Stdout = os.Stdout
			initCmd.Stderr = os.Stderr
			if err := initCmd.Run(); err != nil {
				return fmt.Errorf("go mod init failed: %w", err)
			}
		}

		basename := modPath
		if idx := strings.LastIndex(basename, "/"); idx >= 0 {
			basename = basename[idx+1:]
		}

		cfg := Config{
			Module:     modPath,
			CoreDir:    "core",
			ModulesDir: "modules",
			Autogen:    "internal/autogen",
			Commands: map[string]CommandConfig{
				basename: {
					Entry: filepath.ToSlash(filepath.Join("cmd", basename, "main.go")),
					Uses:  []string{},
				},
			},
		}
		if err := writeConfig(dir, cfg); err != nil {
			return err
		}

		p, err := newProject(dir, false)
		if err != nil {
			return err
		}

		// Create module source directories
		for _, src := range []string{p.coreDir(), p.modulesDir()} {
			srcDir := filepath.Join(dir, src)
			os.MkdirAll(srcDir, 0755)
		}

		// Generate stub module registry
		if err := p.genModuleRegistry(); err != nil {
			return err
		}

		// Write entry point
		cmdDir := filepath.Join(dir, "cmd", basename)
		os.MkdirAll(cmdDir, 0755)

		funcName := commandFuncName(basename)
		mainContent := commandMainContent(p.modPath, p.autoDir, basename, funcName)
		mainPath := filepath.Join(cmdDir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			return fmt.Errorf("write %s: %w", mainPath, err)
		}
		fmt.Printf("Wrote %s\n", mainPath)

		// Generate per-command stub
		if err := writeCommandStub(dir, cfg, basename); err != nil {
			return fmt.Errorf("write command stub: %w", err)
		}

		// Write minimal Makefile
		makefilePath := filepath.Join(dir, "Makefile")
		makefileContent := fmt.Sprintf(`VERSION ?= dev
PROJECT   := %s
LDFLAGS   := -ldflags '-X github.com/asaidimu/hestia/core/internal/boot.ProjectName=$(PROJECT) -X main.version=$(VERSION)'

.PHONY: build run clean

build:
	go build $(LDFLAGS) -o $(PROJECT) ./cmd/%s

run: build
	./$(PROJECT)

clean:
	rm -f $(PROJECT)
`, basename, basename)
		if err := os.WriteFile(makefilePath, []byte(makefileContent), 0644); err != nil {
			return fmt.Errorf("write %s: %w", makefilePath, err)
		}
		fmt.Printf("Wrote %s\n", makefilePath)

		// Write .env.example
		envPath := filepath.Join(dir, ".env.example")
		envContent := `# Session secret (required) — generate with: openssl rand -hex 32
SESSION_SECRET=change-me-to-a-random-secret

# Server port (default 8090)
PORT=8090

# Cookie settings
COOKIE_SECURE=false
COOKIE_SAMESITE=lax
`
		if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
			return fmt.Errorf("write %s: %w", envPath, err)
		}
		fmt.Printf("Wrote %s\n", envPath)

		// Update .gitignore
		gitignorePath := filepath.Join(dir, ".gitignore")
		gitignoreEntry := ".env\n"
		existing, err := os.ReadFile(gitignorePath)
		if err != nil {
			if err := os.WriteFile(gitignorePath, []byte(gitignoreEntry), 0644); err != nil {
				return fmt.Errorf("write %s: %w", gitignorePath, err)
			}
			fmt.Printf("Wrote %s\n", gitignorePath)
		} else if !strings.Contains(string(existing), ".env") {
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("open %s: %w", gitignorePath, err)
			}
			if _, err := f.WriteString(gitignoreEntry); err != nil {
				f.Close()
				return fmt.Errorf("update %s: %w", gitignorePath, err)
			}
			f.Close()
			fmt.Printf("Updated %s\n", gitignorePath)
		} else {
			fmt.Printf("Skipped %s (already has .env entry)\n", gitignorePath)
		}

		// Run go mod tidy
		fmt.Println("Running go mod tidy...")
		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = dir
		tidy.Stdout = os.Stdout
		tidy.Stderr = os.Stderr
		if err := tidy.Run(); err != nil {
			return fmt.Errorf("go mod tidy failed: %w", err)
		}

		fmt.Println("Project initialised successfully.")
		return nil
	},
}
