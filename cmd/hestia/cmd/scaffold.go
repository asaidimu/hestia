package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var AddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"scaffold"},
	Short:   "Add new project components (scaffold)",
}

func init() {
	AddCmd.AddCommand(addCmdCmd)
}

var addCmdCmd = &cobra.Command{
	Use:   "cmd <name>",
	Short: "Add a new command entry point",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		requireRoot()

		if isHestiaModule(rootDir) {
			fmt.Println("Skipping: 'add cmd' is for downstream projects, not for the hestia library repo itself")
			return
		}

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

		modName := modulePath
		if idx := strings.LastIndex(modName, "/"); idx >= 0 {
			modName = modName[idx+1:]
		}
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
		ProjectName:  %q,
		Modules: autogen.Modules(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %%v\n", err)
		os.Exit(1)
	}
}
`, modulePath+"/internal/autogen", modName)

		mainPath := filepath.Join(cmdDir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", mainPath, err)
			os.Exit(1)
		}
		fmt.Printf("Added command %q at %s\n", name, cmdDir)
	},
}
