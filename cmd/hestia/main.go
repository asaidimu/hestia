package main

import (
	"os"

	"github.com/spf13/cobra"

	cmd "github.com/asaidimu/hestia/cmd/hestia/core"
)

// @note #tool-20261029-001 todo : hestia cannot embed the anansi CLI because its logic is unimportable
// @priority high
// @tags #tooling #architecture #anansi
// @author claude
// @see #tool-20261029-002
// @see #tool-20261029-003
// @see #tool-20261029-004
//
// anansi's commands live in go-anansi cmd/anansi/main.go (package main) and cmd/anansi/internal (schema new/normalize, migrate generate/squash, codegen golang/typescript/faker, scaffold, agents). Go forbids importing either from another module, so hestia can only call the public libraries it already uses (codegen/golang). Every other step is a separate binary: README and skills/SKILL.md tell users to run `anansi migrate generate --dry-run`, `anansi migrate generate` and `anansi codegen golang` by hand, then hestia's own generators (#tool-20261029-002).
// Version drift: go.mod pins go-anansi v8.10.1 (latest tag is v8.10.2) while the CLI is installed independently, and the migration and model output depends on the CLI version that happened to run.
// End state: in go-anansi, move command construction out of main/internal into an importable package (for example .../cli returning the schema, migrate, codegen and agents cobra commands, taking the config location and writers), keep cmd/anansi as a thin main, and mount them in hestia (hestia schema|migrate|codegen|agents) so the binary is version-locked to go.mod. The logic already depends only on public packages (core/schema/definition, core/data, codegen/*, skills/anansi).
// Interim, no upstream change (not exercised): add `tool github.com/asaidimu/go-anansi/v8/cmd/anansi` to go.mod (this module is on Go 1.27) and make a hestia passthrough run `go tool anansi`. Do not exec whatever `anansi` is on PATH.
// Skills: `hestia agents` should install both the hestia and anansi skills; skills/SKILL.md currently says "see the anansi skill". Follow-ups: #tool-20261029-003 and #tool-20261029-004.

func main() {
	root := &cobra.Command{
		Use:   "hestia",
		Short: "Hestia Platform CLI — code generation and scaffolding",
	}
	root.AddCommand(cmd.InitCmd)
	root.AddCommand(cmd.GenerateCmd)
	root.AddCommand(cmd.AddCmd)
	root.AddCommand(cmd.RemoveCmd)
	root.AddCommand(cmd.ServiceCmd)
	root.AddCommand(cmd.UseCmd)
	root.AddCommand(cmd.UnuseCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
