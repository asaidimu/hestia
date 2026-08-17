package core

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/asaidimu/go-anansi/v8/codegen/golang"
	"github.com/spf13/cobra"

	"github.com/asaidimu/hestia/cmd/hestia/core/gen"
)

// ServiceCmd scaffolds and manages services inside modules. A module is a
// collection of services: each service lives at <module-dir>/<name>/ and is
// registered by the module's generated collector (services.go).
var ServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Scaffold and manage services inside a module",
}

var serviceNewEntity string
var serviceNewModel bool
var serviceGenerateAll bool

func init() {
	serviceNewCmd.Flags().StringVar(&serviceNewEntity, "entity", "", "Singular entity name override (default: derived from <name>)")
	serviceNewCmd.Flags().BoolVar(&serviceNewModel, "model", true, "Generate a persisted model package (schema + codegen); disable for model-less services")
	serviceGenerateCmd.Flags().BoolVar(&serviceGenerateAll, "all", false, "Regenerate every module's services and their collectors")
	ServiceCmd.AddCommand(serviceNewCmd)
	ServiceCmd.AddCommand(serviceGenerateCmd)
}

// serviceModuleDir resolves the module directory a service belongs to. In the
// hestia library the module is "system" (core/system); downstream the module
// dir is located among the configured modulesDirs.
func serviceModuleDir(module string) (string, error) {
	if isHestiaModule(rootDir) {
		if module != "" && module != "system" {
			return "", fmt.Errorf("unknown module %q: the hestia library provides only the 'system' module (core/system)", module)
		}
		root := filepath.Join(rootDir, "core", "system")
		if _, err := os.Stat(root); err != nil {
			return "", err
		}
		return root, nil
	}
	if module == "" {
		return "", fmt.Errorf("missing module name: use 'hestia service new <module> <name>'")
	}
	for _, src := range modulesDirs {
		dir := filepath.Join(rootDir, src, module)
		if _, err := os.Stat(filepath.Join(dir, "module.go")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("module %q not found under %s (create it first with 'hestia add module %s')", module, strings.Join(modulesDirs, ", "), module)
}

// moduleImportPath returns the import path of a directory relative to the
// module root.
func moduleImportPath(dir string) string {
	rel, err := filepath.Rel(rootDir, dir)
	if err != nil {
		rel = dir
	}
	return modulePath + "/" + filepath.ToSlash(rel)
}

// allModuleDirs returns every module directory: the system module for the
// hestia library, or every configured module dir for downstream projects.
func allModuleDirs() []string {
	if isHestiaModule(rootDir) {
		if _, err := os.Stat(filepath.Join(rootDir, "core", "system")); err == nil {
			return []string{filepath.Join(rootDir, "core", "system")}
		}
		return nil
	}
	var dirs []string
	for _, src := range modulesDirs {
		srcDir := filepath.Join(rootDir, src)
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
				continue
			}
			if _, err := os.Stat(filepath.Join(srcDir, entry.Name(), "module.go")); err != nil {
				continue
			}
			dirs = append(dirs, filepath.Join(srcDir, entry.Name()))
		}
	}
	return dirs
}

var serviceNewCmd = &cobra.Command{
	Use:   "new <module> <name>",
	Short: "Scaffold a new service inside a module",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		requireRoot()
		module, name := args[0], args[1]

		moduleDir, err := serviceModuleDir(module)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		serviceDir := filepath.Join(moduleDir, name)
		if _, err := os.Stat(serviceDir); err == nil {
			fmt.Fprintf(os.Stderr, "Service %q already exists at %s\n", name, serviceDir)
			os.Exit(1)
		}

		entity := serviceNewEntity
		if entity == "" {
			entity = singularize(name)
		}

		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", serviceDir, err)
			os.Exit(1)
		}

		modelImport := moduleImportPath(serviceDir) + "/model"
		if serviceNewModel {
			modelDir := filepath.Join(serviceDir, "model")
			if err := os.MkdirAll(modelDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", modelDir, err)
				os.Exit(1)
			}

			schemaPath := filepath.Join(modelDir, name+".schema.json")
			writeStubSchema(schemaPath, entity)
			generateModel(modelDir, name, schemaPath)
		}

		servicePath := filepath.Join(serviceDir, "service.go")
		writeService(servicePath, name, entity, serviceNewModel, modelImport)

		if err := gen.Generate(serviceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate initial registrations for %s: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("Generated registrations.go and policies.go for %q\n", name)

		fmt.Printf("Scaffolded service %q in module %q at %s\n", name, module, serviceDir)
		refreshCollector(moduleDir)
	},
}

// singularize converts a plural service name to its singular entity form using a
// light heuristic. It is overridable via --entity when it guesses wrong.
func singularize(s string) string {
	if s == "" {
		return s
	}
	switch {
	case strings.HasSuffix(s, "ies") && len(s) > 3:
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "ss"):
		return s
	case strings.HasSuffix(s, "us"):
		return s
	case strings.HasSuffix(s, "is"):
		return s
	case strings.HasSuffix(s, "s") && len(s) > 1:
		return s[:len(s)-1]
	}
	return s
}

func writeStubSchema(path, entity string) {
	content := fmt.Sprintf(`{
  "fields": {
    "name": { "name": "name", "required": true, "type": "string" }
  },
  "name": "_%s_",
  "version": "1.0.0"
}
`, entity)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
}

// generateModel runs anansi codegen in-process against the stub schema and
// writes the generated <name>.schema.model.go beside it.
func generateModel(modelDir, name, schemaPath string) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", schemaPath, err)
		os.Exit(1)
	}

	gen := golang.NewGoGenerator(&golang.GeneratorConfig{
		TagConfig:   golang.DefaultTagConfig(),
		NameRules:   []golang.NameRule{golang.MustCompileRule("^_(.+)_$", "System")},
		PackageName: "model",
	})

	result, err := gen.Generate(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate model for %s: %v\n", schemaPath, err)
		os.Exit(1)
	}

	formatted, err := format.Source([]byte(result))
	if err != nil {
		formatted = []byte(result)
	}

	outPath := strings.TrimSuffix(schemaPath, filepath.Ext(schemaPath)) + ".model.go"
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("generated %s (package model)\n", outPath)
}

// writeService scaffolds service.go. With a model it resolves persistence from
// the abstract.Container DI container and initializes the model collection in
// the constructor; without one it is a plain struct — the feature author brings
// their own deps. modelImport is the import path of the model package.
func writeService(path, name, entity string, withModel bool, modelImport string) {
	if withModel {
		content := fmt.Sprintf(`package %s

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	%q
)

// %sService is the service for the %s domain. The model collection is
// initialized in the constructor by resolving persistence from the
// abstract.Container DI container; the struct is scaffolded once and then owned
// by the feature author.
type %sService struct {
	model *model.System%s
}

func New%sService(rt abstract.Container) (*%sService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)

	m, err := model.InitSystem%sModel(persist, logger)
	if err != nil {
		return nil, err
	}
	return &%sService{model: m}, nil
}
`, name, modelImport, title(name), name, title(name), title(entity)+"s", title(name), title(name), title(entity)+"s", title(name))
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
			os.Exit(1)
		}
		return
	}

	content := fmt.Sprintf(`package %s

import (
	"github.com/asaidimu/hestia/core/abstract"
)

// %sService is the service for the %s domain. It has no persisted model; the
// struct is scaffolded once and then owned by the feature author.
type %sService struct{}

func New%sService(rt abstract.Container) (*%sService, error) {
	return &%sService{}, nil
}
`, name, title(name), name, title(name), title(name), title(name), title(name))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
}

// serviceGenerateCmd regenerates registrations.go and policies.go for a
// service inside a module from its @hestia.register annotations, or for every
// module's services with --all.
var serviceGenerateCmd = &cobra.Command{
	Use:   "generate <module> <name>",
	Short: "Regenerate registrations.go and policies.go for a service",
	Args: func(cmd *cobra.Command, args []string) error {
		if serviceGenerateAll && len(args) != 0 {
			return fmt.Errorf("'generate --all' takes no arguments")
		}
		if !serviceGenerateAll && len(args) != 2 {
			return fmt.Errorf("requires exactly <module> <name>, or --all")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		requireRoot()

		if serviceGenerateAll {
			modules := allModuleDirs()
			if len(modules) == 0 {
				fmt.Fprintln(os.Stderr, "No modules found")
				os.Exit(1)
			}
			for _, moduleDir := range modules {
				entries, err := os.ReadDir(moduleDir)
				if err != nil {
					continue
				}
				for _, e := range entries {
					if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
						continue
					}
					if _, err := os.Stat(filepath.Join(moduleDir, e.Name(), "registrations.go")); err != nil {
						continue
					}
					dir := filepath.Join(moduleDir, e.Name())
					if err := gen.Generate(dir); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to generate %s: %v\n", e.Name(), err)
						os.Exit(1)
					}
					fmt.Printf("Generated registrations.go and policies.go for %q\n", e.Name())
				}
				refreshCollector(moduleDir)
			}
			return
		}

		module, name := args[0], args[1]
		moduleDir, err := serviceModuleDir(module)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		dir := filepath.Join(moduleDir, name)
		if _, err := os.Stat(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Service %q not found at %s\n", name, dir)
			os.Exit(1)
		}
		if err := gen.Generate(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate %s: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("Generated registrations.go and policies.go for %q\n", name)

		refreshCollector(moduleDir)
	},
}

// refreshCollector regenerates a module's service collector (services.go)
// after a service's registrations change or a service is added/removed.
func refreshCollector(moduleDir string) {
	if _, err := os.Stat(moduleDir); err != nil {
		return
	}
	if err := gen.GenerateCollector(moduleDir, modulePath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to regenerate service collector: %v\n", err)
		os.Exit(1)
	}
	rel, err := filepath.Rel(rootDir, filepath.Join(moduleDir, "services.go"))
	if err != nil {
		rel = filepath.Join(moduleDir, "services.go")
	}
	fmt.Printf("Generated %s (service collector)\n", rel)
}