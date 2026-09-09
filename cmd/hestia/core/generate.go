package core

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

var forceMode bool

var GenerateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate feature wiring and module registry code",
}

var GenerateFeaturesCmd = &cobra.Command{
	Use:   "features",
	Short: "[LIBRARY DEV ONLY] Regenerate internal feature wiring inside hestia's own repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", forceMode)
		if err != nil {
			return err
		}
		if !p.isHestia() {
			return nil
		}
		features, err := p.scanFeatures()
		if err != nil {
			return err
		}
		return p.genFeatures(features)
	},
}

var GenerateModulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Generate internal/autogen/modules.go for the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := newProject(".", forceMode)
		if err != nil {
			return err
		}
		return p.genModuleRegistry()
	},
}

func init() {
	GenerateCmd.AddCommand(GenerateFeaturesCmd)
	GenerateCmd.AddCommand(GenerateModulesCmd)
	GenerateCmd.Flags().BoolVar(&forceMode, "force", false, "Force module registry generation even when running inside the hestia library repo")
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
	case strings.Contains(t, "PolicyStore") || strings.Contains(t, "BindingPolicyStore"):
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

func sortFeatures(features []featureInfo) {
	sort.Slice(features, func(i, j int) bool {
		return features[i].DirName < features[j].DirName
	})
}
