package core

import (
	"fmt"
	"os"
	"path/filepath"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		p, err := newProject(".", false)
		if err != nil {
			return err
		}
		if p.isHestia() {
			return nil
		}

		if p.Cfg.Commands == nil {
			p.Cfg.Commands = map[string]CommandConfig{}
		}

		if _, exists := p.Cfg.Commands[name]; !exists {
			p.Cfg.Commands[name] = CommandConfig{
				Entry: filepath.ToSlash(filepath.Join("cmd", name, "main.go")),
				Uses:  []string{},
			}
		}
		if err := writeConfig(p.Root, p.Cfg); err != nil {
			return err
		}

		cmdDir := filepath.Join(p.Root, "cmd", name)
		if _, err := os.Stat(cmdDir); err == nil {
			return fmt.Errorf("command %q already exists at %s", name, cmdDir)
		}
		os.MkdirAll(cmdDir, 0755)

		funcName := commandFuncName(name)
		mainContent := commandMainContent(p.modPath, p.autoDir, name, funcName)

		mainPath := filepath.Join(cmdDir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			return fmt.Errorf("write %s: %w", mainPath, err)
		}
		fmt.Printf("Added command %q at %s\n", name, cmdDir)

		if err := writeCommandStub(p.Root, p.Cfg, name); err != nil {
			return fmt.Errorf("write command stub: %w", err)
		}
		return nil
	},
}
