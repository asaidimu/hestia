package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/asaidimu/hestia/cmd/hestia/cmd"
)

func main() {
	root := &cobra.Command{
		Use:   "hestia",
		Short: "Hestia Platform CLI — code generation and scaffolding",
	}
	root.AddCommand(cmd.GenerateCmd)
	root.AddCommand(cmd.ModuleCmd)
	root.AddCommand(cmd.ScaffoldCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
