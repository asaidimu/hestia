package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type featureInfo struct {
	PkgName string
	DirName string
	Fields  []depField
}

type depField struct {
	Name string
	Type string
}

var (
	rootDir       string
	modulePath    string
	moduleSources []string
	moduleTarget  string
	autogenTarget string
	selfMode      bool
	forceMode     bool
)

func requireRoot() {
	if rootDir != "" {
		return
	}
	if _, err := os.Stat("hestia.json"); err != nil {
		fmt.Fprintln(os.Stderr, "hestia.json not found in current directory")
		os.Exit(1)
	}
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	rootDir = dir
	cfg := readConfig(rootDir)
	if cfg.Module != "" {
		modulePath = cfg.Module
	} else {
		modulePath = detectModulePath(rootDir)
	}
	moduleSources = cfg.ModuleSources
	if len(moduleSources) == 0 {
		moduleSources = []string{"module"}
	}
	moduleTarget = cfg.ModuleTarget
	if moduleTarget == "" {
		moduleTarget = "module"
	}
	autogenTarget = cfg.AutogenTarget
	if autogenTarget == "" {
		autogenTarget = "internal/autogen"
	}
	if modulePath == "" {
		fmt.Fprintln(os.Stderr, "Cannot detect module path: set \"module\" in hestia.json or ensure go.mod is present")
		os.Exit(1)
	}
}

var GenerateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate feature wiring and module registry code",
}

var GenerateFeaturesCmd = &cobra.Command{
	Use:   "features",
	Short: "[LIBRARY DEV ONLY] Regenerate internal feature wiring inside hestia's own repo",
	Run: func(cmd *cobra.Command, args []string) {
		requireRoot()
		if !isHestiaModule(rootDir) {
			fmt.Println("Skipping: 'generate features' is only for use inside the hestia library repo")
			return
		}
		features := scanFeatures()
		genFeatures(features)
	},
}

var GenerateModulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Generate internal/autogen/modules.go for the current project",
	Run: func(cmd *cobra.Command, args []string) {
		requireRoot()
		genModuleRegistry()
	},
}

func init() {
	GenerateCmd.AddCommand(GenerateFeaturesCmd)
	GenerateCmd.AddCommand(GenerateModulesCmd)
	GenerateCmd.Flags().BoolVar(&forceMode, "force", false, "Force module registry generation even when running inside the hestia library repo")
	// Keep --self as a deprecated top-level flag for backward compat
	GenerateCmd.Flags().BoolVar(&selfMode, "self", false, "[DEPRECATED] Use 'generate features' instead")
	GenerateCmd.Flags().MarkDeprecated("self", "use 'hestia generate features' instead")
}

func scanFeatures() []featureInfo {
	featureDir := filepath.Join(rootDir, "internal", "app")
	entries, err := os.ReadDir(featureDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read feature directory %s: %v\n", featureDir, err)
		os.Exit(1)
	}

	var features []featureInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}

		regPath := filepath.Join(featureDir, name, "feature.go")
		if _, err := os.Stat(regPath); os.IsNotExist(err) {
			continue
		}

		pkgName := extractPackageName(regPath)
		if pkgName == "" {
			continue
		}

		fields := extractFeatureDeps(regPath)

		features = append(features, featureInfo{
			PkgName: pkgName,
			DirName: name,
			Fields:  fields,
		})
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].DirName < features[j].DirName
	})
	return features
}

func extractPackageName(path string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
	if err != nil {
		return ""
	}
	return f.Name.Name
}

func extractFeatureDeps(path string) []depField {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Dependencies" {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			var fields []depField
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				typeStr := exprToString(field.Type)
				fields = append(fields, depField{
					Name: field.Names[0].Name,
					Type: typeStr,
				})
			}
			return fields
		}
	}
	return nil
}

func isAllUpper(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			return false
		}
	}
	return true
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "any"
	default:
		return "unknown"
	}
}

func fieldToExpr(f depField) string {
	t := f.Type
	switch {
	case t == "*zap.Logger":
		return "m.opts.Logger"
	case t == "*UserModel":
		return "m.userModel"
	case t == "*APIKeyModel":
		return "m.apiKeyModel"
	case t == "*PolicyModel":
		return "m.policyModel"
	case t == "*SeedModel":
		return "m.seedModel"
	case t == "*audit.AccessLogModel":
		return "m.accessLogModel"
	case t == "core.Dispatcher" || t == "*corepkg.LocalDispatcher" || t == "*core.LocalDispatcher":
		return "m.disp"
	case t == "persistence.Persistence" || t == "base.Persistence":
		return "m.persist"
	case t == "core.Registry" || t == "corepkg.Registry":
		return "m.disp"
	case t == "core.ReloadablePermissionManager" || t == "corepkg.ReloadablePermissionManager":
		return "m.permMgr"
	case t == "iam.AccessController":
		return "m.ac"
	case strings.Contains(t, "BlobStore"):
		return "m.blobSvc"
	case strings.Contains(t, "PermissionManager"):
		return "m.permMgr"
	case strings.Contains(t, "PolicyStore") || strings.Contains(t, "PolicyOperation"):
		return "m.policyBridge"
	case strings.HasPrefix(t, "func("):
		switch f.Name {
		case "Bootstrapped":
			return "func() bool { return m.bootstrapped }"
		case "OnBootstrap":
			return "func() {\n\t\t\tm.bootstrapped = true\n\t\t\tif m.opts.OnBootstrapped != nil {\n\t\t\t\tm.opts.OnBootstrapped()\n\t\t\t}\n\t\t}"
		case "OnReset":
			return "func() {\n\t\t\tif m.opts.OnReset != nil {\n\t\t\t\tm.opts.OnReset()\n\t\t\t}\n\t\t}"
		case "CompileRules":
			return "policies.CompileRules"
		default:
			return "func() {}"
		}
	case f.Name == "AdminUserID":
		return `m.adminUserID`
	case t == "*APIKeyAuthenticator":
		return "apiKeyAuth"
	case strings.Contains(t, "*[]abstract.MessageRegistration"):
		return "&allRegs"
	default:
		name := f.Name
		first := strings.ToLower(name[:1])
		rest := name[1:]
		if len(name) > 2 && isAllUpper(name[:2]) {
			first = strings.ToLower(name[:2])
			rest = name[2:]
		}
		return fmt.Sprintf("m.%s%s", first, rest)
	}
}

func genFeatures(features []featureInfo) {
	var b strings.Builder

	b.WriteString("// Code generated by cmd/hestia generate; DO NOT EDIT.\n")
	b.WriteString("\n// Package app contains generated feature wiring.\n")
	b.WriteString("package app\n\n")
	b.WriteString("import (\n")
	b.WriteString(fmt.Sprintf("\t%q\n", modulePath+"/internal/abstract"))
	for _, f := range features {
		b.WriteString(fmt.Sprintf("\t%q\n", fmt.Sprintf("%s/internal/app/%s", modulePath, f.DirName)))
	}
	b.WriteString(")\n\n")

	b.WriteString("var allDefaultOperations = func() []policies.PolicyOperation {\n")
	b.WriteString("\tvar all []policies.PolicyOperation\n")
	for _, f := range features {
		b.WriteString(fmt.Sprintf("\tall = append(all, %s.DefaultOperations()...)\n", f.PkgName))
	}
	b.WriteString("\treturn all\n")
	b.WriteString("}()\n\n")

	b.WriteString("func collectFeatureRegistrations(m *SystemModule, apiKeyAuth *auth.APIKeyAuthenticator) []abstract.MessageRegistration {\n")
	b.WriteString("\tvar all []abstract.MessageRegistration\n")
	b.WriteString("\tvar allRegs []abstract.MessageRegistration\n\n")

	for _, f := range features {
		b.WriteString(fmt.Sprintf("\t%sDeps := %s.Dependencies{\n", f.PkgName, f.PkgName))
		for _, field := range f.Fields {
			expr := fieldToExpr(field)
			b.WriteString(fmt.Sprintf("\t\t%s: %s,\n", field.Name, expr))
		}
		b.WriteString("\t}\n")
		b.WriteString(fmt.Sprintf("\tall = append(all, %s.Registrations(%sDeps)...)\n", f.PkgName, f.PkgName))
	}

	b.WriteString("\n\tallRegs = all\n")
	b.WriteString("\treturn all\n")
	b.WriteString("}\n")

	outDir := filepath.Join(rootDir, "internal", "app")
	os.MkdirAll(outDir, 0755)
	outPath := filepath.Join(outDir, "gen_features.go")
	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write generated file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s with %d features\n", outPath, len(features))
}

func isHestiaModule(root string) bool {
	// Walk up from root looking for go.mod with "module github.com/asaidimu/hestia"
	dir := root
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			for i := 0; i < len(data); i++ {
				if data[i] == '\n' || data[i] == '\r' || i == len(data)-1 {
					end := i
					if i == len(data)-1 {
						end = i + 1
					}
					line := string(data[:end])
					if strings.HasPrefix(line, "module ") {
						return strings.TrimSpace(line[7:]) == "github.com/asaidimu/hestia"
					}
					return false
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func genModuleRegistry() {
	if isHestiaModule(rootDir) && !forceMode {
		fmt.Println("Skipping module registry generation: running inside the hestia library (use --force to override)")
		return
	}
	var modules []string
	for _, src := range moduleSources {
		srcDir := filepath.Join(rootDir, src)
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				continue
			}
			modPath := filepath.Join(srcDir, name, "module.go")
			if _, err := os.Stat(modPath); os.IsNotExist(err) {
				continue
			}
			pkgName := extractPackageName(modPath)
			if pkgName == "" {
				continue
			}
			modules = append(modules, pkgName)
		}
	}

	if len(modules) == 0 {
		outDir := filepath.Join(rootDir, autogenTarget)
		os.MkdirAll(outDir, 0755)
		outPath := filepath.Join(outDir, "modules.go")
		stub := "// Code generated by cmd/hestia generate; DO NOT EDIT.\n\n// Package autogen is the generated module registry.\npackage autogen\n\nimport \"github.com/asaidimu/hestia\"\n\nfunc Modules() []hestia.Module {\n\treturn nil\n}\n"
		if err := os.WriteFile(outPath, []byte(stub), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s (no modules found)\n", outPath)
		return
	}

	sort.Strings(modules)

	var b strings.Builder
	b.WriteString("// Code generated by cmd/hestia generate; DO NOT EDIT.\n")
	b.WriteString("\n// Package autogen is the generated module registry.\n")
	b.WriteString("package autogen\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"github.com/asaidimu/hestia\"\n")
	for _, m := range modules {
		b.WriteString(fmt.Sprintf("\t%q\n", fmt.Sprintf("%s/%s/%s", modulePath, moduleTarget, m)))
	}
	b.WriteString(")\n\n")
	b.WriteString("func Modules() []hestia.Module {\n")
	b.WriteString("\treturn []hestia.Module{\n")
	for _, m := range modules {
		b.WriteString(fmt.Sprintf("\t\t%s.New(),\n", m))
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")

	outDir := filepath.Join(rootDir, autogenTarget)
	os.MkdirAll(outDir, 0755)
	outPath := filepath.Join(outDir, "modules.go")
	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s with %d modules\n", outPath, len(modules))
}
