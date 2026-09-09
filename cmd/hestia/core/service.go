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

// ServiceCmd scaffolds and manages services inside modules.
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

var serviceNewCmd = &cobra.Command{
	Use:   "new <module> <name>",
	Short: "Scaffold a new service inside a module",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", false)
		if err != nil {
			return err
		}
		module, name := args[0], args[1]

		moduleDir, err := p.serviceModuleDir(module)
		if err != nil {
			return err
		}

		serviceDir := filepath.Join(moduleDir, name)
		if _, err := os.Stat(serviceDir); err == nil {
			return fmt.Errorf("service %q already exists at %s", name, serviceDir)
		}

		entity := serviceNewEntity
		if entity == "" {
			entity = singularize(name)
		}

		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", serviceDir, err)
		}

		modelImport := p.moduleImport(serviceDir) + "/model"
		if serviceNewModel {
			modelDir := filepath.Join(serviceDir, "model")
			if err := os.MkdirAll(modelDir, 0755); err != nil {
				return fmt.Errorf("create %s: %w", modelDir, err)
			}
			migrationsDir := filepath.Join(modelDir, "migrations")
			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				return fmt.Errorf("create %s: %w", migrationsDir, err)
			}

			schemaPath := filepath.Join(modelDir, name+".schema.json")
			if err := writeStubSchema(schemaPath, entity); err != nil {
				return err
			}
			if err := generateModel(modelDir, name, schemaPath); err != nil {
				return err
			}
		}

		servicePath := filepath.Join(serviceDir, "service.go")
		if err := writeService(servicePath, name, entity, serviceNewModel, modelImport); err != nil {
			return err
		}

		if err := gen.Generate(serviceDir); err != nil {
			return fmt.Errorf("generate initial registrations for %s: %w", name, err)
		}
		fmt.Printf("Generated registrations.go and policies.go for %q\n", name)
		if seeded, err := gen.SeedSanitization(serviceDir); err != nil {
			return fmt.Errorf("seed sanitization.go for %s: %w", name, err)
		} else if seeded {
			fmt.Printf("Generated sanitization.go (seed) for %q\n", name)
		}

		fmt.Printf("Scaffolded service %q in module %q at %s\n", name, module, serviceDir)
		return p.refreshCollector(moduleDir)
	},
}

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
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", false)
		if err != nil {
			return err
		}

		if serviceGenerateAll {
			modules := p.allModuleDirs()
			if len(modules) == 0 {
				return fmt.Errorf("no modules found")
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
						return fmt.Errorf("generate %s: %w", e.Name(), err)
					}
					fmt.Printf("Generated registrations.go and policies.go for %q\n", e.Name())
					if seeded, err := gen.SeedSanitization(dir); err != nil {
						return fmt.Errorf("seed sanitization.go for %s: %w", e.Name(), err)
					} else if seeded {
						fmt.Printf("Generated sanitization.go (seed) for %q\n", e.Name())
					}
				}
				if err := p.refreshCollector(moduleDir); err != nil {
					return err
				}
			}
			return nil
		}

		module, name := args[0], args[1]
		moduleDir, err := p.serviceModuleDir(module)
		if err != nil {
			return err
		}
		dir := filepath.Join(moduleDir, name)
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("service %q not found at %s", name, dir)
		}
		if err := gen.Generate(dir); err != nil {
			return fmt.Errorf("generate %s: %w", name, err)
		}
		fmt.Printf("Generated registrations.go and policies.go for %q\n", name)
		if seeded, err := gen.SeedSanitization(dir); err != nil {
			return fmt.Errorf("seed sanitization.go for %s: %w", name, err)
		} else if seeded {
			fmt.Printf("Generated sanitization.go (seed) for %q\n", name)
		}

		return p.refreshCollector(moduleDir)
	},
}

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

func writeStubSchema(path, entity string) error {
	content := fmt.Sprintf(`{
  "fields": {
    "name": { "name": "name", "required": true, "type": "string" }
  },
  "name": "_%s_",
  "version": "1.0.0"
}
`, entity)
	return os.WriteFile(path, []byte(content), 0644)
}

func generateModel(modelDir, name, schemaPath string) error {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", schemaPath, err)
	}

	gen := golang.NewGoGenerator(&golang.GeneratorConfig{
		TagConfig:   golang.DefaultTagConfig(),
		NameRules:   []golang.NameRule{golang.MustCompileRule("^_(.+)_$", "System")},
		PackageName: "model",
	})

	result, err := gen.Generate(raw)
	if err != nil {
		return fmt.Errorf("generate model for %s: %w", schemaPath, err)
	}

	formatted, err := format.Source([]byte(result))
	if err != nil {
		formatted = []byte(result)
	}

	outPath := strings.TrimSuffix(schemaPath, filepath.Ext(schemaPath)) + ".model.go"
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Printf("generated %s (package model)\n", outPath)
	return nil
}

func writeService(path, name, entity string, withModel bool, modelImport string) error {
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
`, name, modelImport, gen.Title(name), name, gen.Title(name), gen.Title(entity)+"s", gen.Title(name), gen.Title(name), gen.Title(entity)+"s", gen.Title(name))
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		return nil
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
`, name, gen.Title(name), name, gen.Title(name), gen.Title(name), gen.Title(name), gen.Title(name))
	return os.WriteFile(path, []byte(content), 0644)
}
