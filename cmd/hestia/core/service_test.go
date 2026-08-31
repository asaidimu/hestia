package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asaidimu/hestia/cmd/hestia/core/gen"
)

func TestSingularize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"users", "user"},
		{"widgets", "widget"},
		{"policies", "policy"},
		{"categories", "category"},
		{"status", "status"},
		{"access", "access"},
		{"orders", "order"},
		{"api", "api"},
		{"", ""},
		{"notifications", "notification"},
	}
	for _, c := range cases {
		if got := singularize(c.in); got != c.want {
			t.Errorf("singularize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestModulesConfig verifies that the Modules config field accepts both a
// single string and a list, defaulting the target to the first entry.
func TestModulesConfig(t *testing.T) {
	var single Config
	writeFile(filepath.Join(t.TempDir(), "hestia.json"), `{"module":"m","modules":"mods"}`)
	// UnmarshalJSON is exercised via readConfig below.
	dir := t.TempDir()
	writeFile(filepath.Join(dir, "hestia.json"), `{"module":"github.com/example/bench","modules":"mods"}`)
	cfg := readConfig(dir)
	if len(cfg.Modules) != 1 || cfg.Modules[0] != "mods" {
		t.Errorf("single-string modules = %v, want [mods]", cfg.Modules)
	}

	dir2 := t.TempDir()
	writeFile(filepath.Join(dir2, "hestia.json"), `{"module":"github.com/example/bench","modules":["mods","vendor"]}`)
	cfg2 := readConfig(dir2)
	if len(cfg2.Modules) != 2 || cfg2.Modules[0] != "mods" {
		t.Errorf("list modules = %v, want [mods vendor]", cfg2.Modules)
	}

	dir3 := t.TempDir()
	writeFile(filepath.Join(dir3, "hestia.json"), `{"module":"github.com/example/bench"}`)
	cfg3 := readConfig(dir3)
	if len(cfg3.Modules) != 0 {
		t.Errorf("absent modules = %v, want empty", cfg3.Modules)
	}
	_ = single
}

// TestScaffoldService runs the service-new flow against a temporary module and
// asserts the expected tree plus a compile-check of the generated package.
func TestScaffoldService(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	writeFile(filepath.Join(dir, "go.mod"), `module github.com/example/bench

go 1.27rc1

require (
	github.com/asaidimu/go-anansi/v8 v8.5.7
	github.com/asaidimu/hestia v0.0.0
	go.uber.org/zap v1.28.0
)

replace github.com/asaidimu/hestia => `+repoRoot+`
`)
	writeFile(filepath.Join(dir, "hestia.json"), `{
  "module": "github.com/example/bench",
  "modules": ["module", "vendor"]
}`)

	// Copy the module graph from the hestia repo (test runs inside its module)
	// so the temp module can resolve go-anansi offline.
	if data, err := os.ReadFile(filepath.Join(repoRoot, "go.sum")); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "go.sum"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	prevRoot, prevModule, prevDirs, prevAutogen := rootDir, modulePath, modulesDirs, autogenDir
	t.Cleanup(func() {
		rootDir, modulePath, modulesDirs, autogenDir = prevRoot, prevModule, prevDirs, prevAutogen
	})
	rootDir, modulePath, modulesDirs, autogenDir = dir, "github.com/example/bench", []string{"module", "vendor"}, "internal/autogen"

	// service-new expects the process cwd to contain hestia.json. Temporarily
	// chdir so requireRoot() resolves against the fixture, not the hestia repo.
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	// The module must exist before a service can be added to it.
	modDir := filepath.Join(dir, "module", "billing")
	if err := os.MkdirAll(filepath.Join(modDir, "billing"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(filepath.Join(modDir, "module.go"), scaffoldModule("billing"))

	entity := "user"
	serviceDir := filepath.Join(modDir, "users")

	rootDir, modulePath = "", ""
	requireRoot()
	if len(modulesDirs) != 2 || modulesDirs[0] != "module" {
		t.Fatalf("requireRoot modulesDirs = %v", modulesDirs)
	}
	if _, err := os.Stat(serviceDir); err == nil {
		t.Fatalf("service dir should not exist yet: %s", serviceDir)
	}
	modelDir := filepath.Join(serviceDir, "model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(modelDir, "users.schema.json")
	writeStubSchema(schemaPath, entity)
	generateModel(modelDir, "users", schemaPath)
	modelImport := moduleImportPath(serviceDir) + "/model"
	writeService(filepath.Join(serviceDir, "service.go"), "users", entity, true, modelImport)
	if err := gen.Generate(serviceDir); err != nil {
		t.Fatalf("gen.Generate: %v", err)
	}
	seeded, err := gen.SeedSanitization(serviceDir)
	if err != nil || !seeded {
		t.Fatalf("gen.SeedSanitization: seeded=%v err=%v", seeded, err)
	}
	refreshCollector(modDir)

	for _, p := range []string{
		"module/billing/users/service.go",
		"module/billing/users/sanitization.go",
		"module/billing/users/model/users.schema.json",
		"module/billing/users/model/users.schema.model.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("missing scaffolded file %s: %v", p, err)
		}
	}

	svc, err := os.ReadFile(filepath.Join(serviceDir, "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svc), "type UsersService struct") {
		t.Errorf("service.go missing UsersService struct:\n%s", svc)
	}
	if !strings.Contains(string(svc), "model *model.SystemUsers") {
		t.Errorf("service.go missing model field:\n%s", svc)
	}
	if !strings.Contains(string(svc), `"github.com/example/bench/module/billing/users/model"`) {
		t.Errorf("service.go missing module-scoped model import:\n%s", svc)
	}

	modelSrc, err := os.ReadFile(filepath.Join(serviceDir, "model", "users.schema.model.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type SystemUser struct", "type SystemUsers struct", "package model"} {
		if !strings.Contains(string(modelSrc), want) {
			t.Errorf("generated model missing %q", want)
		}
	}

	// Compile proof: the scaffolded package must build against the real
	// go-anansi dependency. Resolve the temp module's graph (deps are in the
	// local module cache, so tidy stays offline), then build.
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy in temp module failed: %v\n%s", err, out)
	}
	build := exec.Command("go", "build", "./module/...")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build of scaffolded service failed: %v\n%s", err, out)
	}
}

// TestScaffoldModule runs the add-module flow against a temporary module and
// asserts the expected tree (module.go + annotation-style feature + generated
// registrations and module collector) plus a compile-check of the packages.
func TestScaffoldModule(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	writeFile(filepath.Join(dir, "go.mod"), `module github.com/example/bench

go 1.27rc1

require (
	github.com/asaidimu/go-anansi/v8 v8.5.7
	github.com/asaidimu/hestia v0.0.0
	go.uber.org/zap v1.28.0
)

replace github.com/asaidimu/hestia => `+repoRoot+`
`)
	writeFile(filepath.Join(dir, "hestia.json"), `{
  "module": "github.com/example/bench",
  "modules": "module",
  "autogen": "internal/autogen"
}`)
	if data, err := os.ReadFile(filepath.Join(repoRoot, "go.sum")); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "go.sum"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	prevRoot, prevModule, prevDirs, prevAutogen := rootDir, modulePath, modulesDirs, autogenDir
	t.Cleanup(func() {
		rootDir, modulePath, modulesDirs, autogenDir = prevRoot, prevModule, prevDirs, prevAutogen
	})
	rootDir, modulePath, modulesDirs, autogenDir = dir, "github.com/example/bench", []string{"module"}, "internal/autogen"

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	rootDir, modulePath = "", ""
	requireRoot()

	modName, featureName := "billing", "billing"
	modDir := filepath.Join(dir, "module", modName)
	featureDir := filepath.Join(modDir, featureName)
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(filepath.Join(modDir, "module.go"), scaffoldModule(modName))
	writeFile(filepath.Join(featureDir, "handler.go"), scaffoldHandler(modName, featureName))
	writeFile(filepath.Join(featureDir, "model.go"), scaffoldModel(featureName))
	if err := gen.Generate(featureDir); err != nil {
		t.Fatalf("gen.Generate: %v", err)
	}
	refreshCollector(modDir)

	for _, p := range []string{
		"module/billing/module.go",
		"module/billing/services.go",
		"module/billing/billing/handler.go",
		"module/billing/billing/model.go",
		"module/billing/billing/registrations.go",
		"module/billing/billing/policies.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("missing scaffolded file %s: %v", p, err)
		}
	}

	modSrc, err := os.ReadFile(filepath.Join(modDir, "module.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"module.BaseModule",
		"Setup(ctx context.Context, rt abstract.Container) error",
		"Capabilities(rt abstract.Container) ([]abstract.Capability, error)",
		"RegisterServices(rt)",
		"CollectServiceRegistrations(rt)",
	} {
		if !strings.Contains(string(modSrc), want) {
			t.Errorf("module.go missing %q:\n%s", want, modSrc)
		}
	}
	regSrc, err := os.ReadFile(filepath.Join(featureDir, "registrations.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(regSrc), `Name:        "billing:billing:ping:check"`) {
		t.Errorf("registrations.go missing 4-segment message name:\n%s", regSrc)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy in temp module failed: %v\n%s", err, out)
	}
	build := exec.Command("go", "build", "./module/...")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build of scaffolded module failed: %v\n%s", err, out)
	}
}

// TestScaffoldServiceNoModel verifies --model=false skips the model package
// entirely and emits a plain service struct inside a module.
func TestScaffoldServiceNoModel(t *testing.T) {
	dir := t.TempDir()
	writeFile(filepath.Join(dir, "go.mod"), "module github.com/example/bench\n\ngo 1.27rc1\n")
	writeFile(filepath.Join(dir, "hestia.json"), `{
  "module": "github.com/example/bench",
  "modules": "module"
}`)

	prevRoot, prevModule, prevDirs := rootDir, modulePath, modulesDirs
	t.Cleanup(func() {
		rootDir, modulePath, modulesDirs = prevRoot, prevModule, prevDirs
	})
	rootDir, modulePath, modulesDirs = dir, "github.com/example/bench", []string{"module"}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	rootDir, modulePath = "", ""
	requireRoot()

	modDir := filepath.Join(dir, "module", "billing")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(filepath.Join(modDir, "module.go"), scaffoldModule("billing"))

	serviceDir := filepath.Join(modDir, "events")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeService(filepath.Join(serviceDir, "service.go"), "events", "event", false, "")

	// No model package should exist.
	if _, err := os.Stat(filepath.Join(serviceDir, "model")); !os.IsNotExist(err) {
		t.Errorf("model package should not exist for --model=false")
	}

	svc, err := os.ReadFile(filepath.Join(serviceDir, "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svc), `"github.com/asaidimu/hestia/core/abstract"`) {
		t.Errorf("model-less service should import abstract for the DI constructor:\n%s", svc)
	}
	if !strings.Contains(string(svc), "type EventsService struct{}") {
		t.Errorf("service.go missing plain struct:\n%s", svc)
	}
	if !strings.Contains(string(svc), "func NewEventsService(rt abstract.Container) (*EventsService, error)") {
		t.Errorf("service.go missing DI-shaped constructor:\n%s", svc)
	}
}