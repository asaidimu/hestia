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
	root.AddCommand(cmd.InitCmd)
	root.AddCommand(cmd.GenerateCmd)
	root.AddCommand(cmd.AddCmd)
	root.AddCommand(cmd.RemoveCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
