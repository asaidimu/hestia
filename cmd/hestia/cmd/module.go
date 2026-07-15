package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var ModuleCmd = &cobra.Command{
	Use:   "module",
	Short: "Manage external modules",
}

func init() {
	ModuleCmd.AddCommand(scaffoldCmd)
}

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <module-name> [feature-name]",
	Short: "Scaffold a new external module",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		requireRoot()
		modName := args[0]
		featureName := modName
		if len(args) >= 2 {
			featureName = args[1]
		}

		modDir := filepath.Join(rootDir, moduleTarget, modName)
		featureDir := filepath.Join(modDir, featureName)

		if _, err := os.Stat(modDir); err == nil {
			fmt.Fprintf(os.Stderr, "Module %q already exists at %s\n", modName, modDir)
			os.Exit(1)
		}

		os.MkdirAll(featureDir, 0755)

		writeFile(filepath.Join(modDir, "module.go"), scaffoldModule(modName, featureName))
		writeFile(filepath.Join(featureDir, "feature.go"), scaffoldRegister(modName, featureName))
		writeFile(filepath.Join(featureDir, "handler.go"), scaffoldHandler(featureName))
		writeFile(filepath.Join(featureDir, "model.go"), scaffoldModel(featureName))
		writeFile(filepath.Join(featureDir, "defaults.go"), scaffoldDefaults(featureName))

		fmt.Printf("Scaffolded module %q at %s\n", modName, modDir)
	},
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content+"\n"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
}

func scaffoldModule(modName, featureName string) string {
	importPath := modulePath + "/" + moduleTarget + "/" + modName + "/" + featureName
	return fmt.Sprintf(`package %s

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia"
	%q
)

type Module struct {
	store    *%s.%sStore
	messages []hestia.MessageRegistration
}

func New() *Module {
	return &Module{
		store: %s.New%sStore(),
	}
}

func (m *Module) Name() string { return %q }

func (m *Module) Setup(_ context.Context, _ base.Persistence) error {
	m.messages = %s.Registrations(%s.Dependencies{Store: m.store})
	return nil
}

func (m *Module) Capabilities() []hestia.Capability {
	return []hestia.Capability{
		{Name: %q, Messages: m.messages},
	}
}
`, modName, importPath, featureName, title(featureName), featureName, title(featureName), modName, featureName, featureName, featureName)
}

func scaffoldRegister(modName, featureName string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/asaidimu/hestia"
)

type Dependencies struct {
	Store *%sStore
}

func Registrations(deps Dependencies) []hestia.MessageRegistration {
	return []hestia.MessageRegistration{
		{Name: %q, Handler: NewPingHandler(deps.Store), Description: "Ping", Enabled: true, Intent: hestia.Read, Output: outputSchema()},
	}
}
`, featureName, title(featureName), fmt.Sprintf("%s:%s:ping", modName, featureName))
}

func scaffoldHandler(featureName string) string {
	return fmt.Sprintf(`package %s

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia"
)

func NewPingHandler(store *%sStore) hestia.MessageHandler {
	return func(ctx context.Context, msg hestia.Message) (*hestia.Result, error) {
		return &hestia.Result{
			Document: data.MustNewDocument(map[string]any{"pong": true}, ctx),
		}, nil
	}
}
`, featureName, title(featureName))
}

func scaffoldModel(featureName string) string {
	return fmt.Sprintf(`package %s

type %sStore struct{}

func New%sStore() *%sStore {
	return &%sStore{}
}
`, featureName, title(featureName), title(featureName), title(featureName), title(featureName))
}

func scaffoldDefaults(featureName string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia"
)

func outputSchema() *definition.Schema {
	return hestia.MustFromJSON([]byte(`+"`"+`{
		"name": "%sOutput",
		"version": "1.0.0",
		"fields": {
			"pong": { "name": "pong", "type": "boolean" }
		}
	}`+"`"+`))
}
`, featureName, title(featureName))
}

func title(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
