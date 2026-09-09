package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var UseCmd = &cobra.Command{
	Use:   "use <command> <module...>",
	Short: "Add feature modules to a command's uses list",
	Long: `Declares that a command loads the given feature modules.
Updates hestia.json and regenerates the command's <name>_modules.go.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", false)
		if err != nil {
			return err
		}
		cmdName := args[0]
		want := args[1:]

		cc, ok := p.Cfg.Commands[cmdName]
		if !ok {
			return fmt.Errorf("unknown command %q (create it first with 'hestia add cmd %s')", cmdName, cmdName)
		}

		_, coreImports := p.discoverModules(p.coreDir())
		_, featImports := p.discoverModules(p.modulesDir())
		var toAdd []string
		for _, m := range want {
			if _, isCore := coreImports[m]; isCore {
				fmt.Fprintf(os.Stderr, "Warning: %q is a core module, already loaded by every command — skipping\n", m)
				continue
			}
			if _, ok := featImports[m]; !ok {
				return fmt.Errorf("unknown module %q under %s", m, filepath.Join(p.modulesDir()))
			}
			already := false
			for _, u := range cc.Uses {
				if u == m {
					already = true
					break
				}
			}
			if !already {
				toAdd = append(toAdd, m)
			}
		}

		cc.Uses = append(cc.Uses, toAdd...)
		p.Cfg.Commands[cmdName] = cc
		if err := writeConfig(p.Root, p.Cfg); err != nil {
			return err
		}

		if err := p.regenCommandModules(); err != nil {
			return fmt.Errorf("regenerate command modules: %w", err)
		}
		fmt.Printf("Command %q now uses %v\n", cmdName, cc.Uses)
		return nil
	},
}

var UnuseCmd = &cobra.Command{
	Use:   "unuse <command> <module...>",
	Short: "Remove feature modules from a command's uses list",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", false)
		if err != nil {
			return err
		}
		cmdName := args[0]
		drop := map[string]bool{}
		for _, m := range args[1:] {
			drop[m] = true
		}

		cc, ok := p.Cfg.Commands[cmdName]
		if !ok {
			return fmt.Errorf("unknown command %q", cmdName)
		}

		var kept []string
		for _, u := range cc.Uses {
			if !drop[u] {
				kept = append(kept, u)
			}
		}
		cc.Uses = kept
		p.Cfg.Commands[cmdName] = cc
		if err := writeConfig(p.Root, p.Cfg); err != nil {
			return err
		}

		if err := p.regenCommandModules(); err != nil {
			return fmt.Errorf("regenerate command modules: %w", err)
		}
		fmt.Printf("Command %q now uses %v\n", cmdName, cc.Uses)
		return nil
	},
}
