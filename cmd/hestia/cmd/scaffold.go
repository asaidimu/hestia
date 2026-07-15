package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ScaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Scaffold new project components",
}

func init() {
	ScaffoldCmd.AddCommand(scaffoldCmdCmd)
}

var scaffoldCmdCmd = &cobra.Command{
	Use:   "cmd <name>",
	Short: "Scaffold a new command entry point",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		requireRoot()

		cfg := readConfig(rootDir)
		if cfg.Module == "" {
			cfg.Module = detectModulePath(rootDir)
		}
		if len(cfg.ModuleSources) == 0 {
			cfg.ModuleSources = []string{"module"}
		}
		if cfg.ModuleTarget == "" {
			cfg.ModuleTarget = "module"
		}

		alreadyRegistered := false
		for _, c := range cfg.Cmds {
			if c == name {
				alreadyRegistered = true
				break
			}
		}
		if !alreadyRegistered {
			cfg.Cmds = append(cfg.Cmds, name)
		}
		writeConfig(rootDir, cfg)

		cmdDir := filepath.Join(rootDir, "cmd", name)
		if _, err := os.Stat(cmdDir); err == nil {
			fmt.Fprintf(os.Stderr, "Command %q already exists at %s\n", name, cmdDir)
			os.Exit(1)
		}
		os.MkdirAll(cmdDir, 0755)

		mainContent := fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/asaidimu/hestia"
	%q
)

var version = "dev"

func main() {
	if err := hestia.Run(hestia.SetupConfig{
		Version:      version,
		Modules: autogen.Modules(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %%v\n", err)
		os.Exit(1)
	}
}
`, modulePath+"/internal/autogen")

		mainPath := filepath.Join(cmdDir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", mainPath, err)
			os.Exit(1)
		}
		fmt.Printf("Scaffolded command %q at %s\n", name, cmdDir)
	},
}
