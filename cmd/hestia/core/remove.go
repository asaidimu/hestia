package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var RemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove project components",
}

func init() {
	RemoveCmd.AddCommand(removeModuleCmd)
}

var removeModuleCmd = &cobra.Command{
	Use:   "module <module-name>",
	Short: "Remove a module",
	Long: `Deletes the module directory (from core_dir or modules_dir),
removes it from every command's uses list, and regenerates the
module registry and command modules.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", false)
		if err != nil {
			return err
		}
		if p.isHestia() {
			return nil
		}
		modName := args[0]

		removed := false
		for _, dir := range []string{p.coreDir(), p.modulesDir()} {
			modDir := filepath.Join(p.Root, dir, modName)
			if _, err := os.Stat(modDir); os.IsNotExist(err) {
				continue
			}
			if err := os.RemoveAll(modDir); err != nil {
				return fmt.Errorf("remove %s: %w", modDir, err)
			}
			fmt.Printf("Removed module %q from %s\n", modName, dir)
			removed = true
		}
		if !removed {
			return fmt.Errorf("module %q not found under %s or %s", modName, p.coreDir(), p.modulesDir())
		}

		// Strip from every command's uses list.
		for cmdName, cc := range p.Cfg.Commands {
			var kept []string
			for _, u := range cc.Uses {
				if u != modName {
					kept = append(kept, u)
				}
			}
			cc.Uses = kept
			p.Cfg.Commands[cmdName] = cc
		}
		if err := writeConfig(p.Root, p.Cfg); err != nil {
			return err
		}

		if err := p.genModuleRegistry(); err != nil {
			return err
		}
		if err := p.regenCommandModules(); err != nil {
			return fmt.Errorf("regenerate command modules: %w", err)
		}
		return nil
	},
}
