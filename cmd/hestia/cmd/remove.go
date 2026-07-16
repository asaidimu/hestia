package cmd

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
	Short: "Remove an external module",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		requireRoot()
		if isHestiaModule(rootDir) {
			fmt.Println("Skipping: 'remove module' is for downstream projects, not for the hestia library repo itself")
			return
		}
		modName := args[0]
		modDir := filepath.Join(rootDir, moduleTarget, modName)

		if _, err := os.Stat(modDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Module %q not found at %s\n", modName, modDir)
			os.Exit(1)
		}

		if err := os.RemoveAll(modDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", modDir, err)
			os.Exit(1)
		}
		fmt.Printf("Removed module %q\n", modName)
		genModuleRegistry()
	},
}
