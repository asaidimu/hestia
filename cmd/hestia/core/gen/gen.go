// Package gen renders a service's registrations.go and policies.go from the
// annotations parsed by the annotate package. Emission targets:
//
//   - Registrations() []abstract.MessageRegistration, each with a
//     dispatch.Handle*/HandleEmpty/HandleInputStream adapter bound to the
//     service method. Streaming signatures (<-chan dispatch.Item[TIn] params)
//     emit HandleInputStream and Input.Streaming = true.
//   - Policies() []policies.Binding keyed by the annotation's rule — streaming
//     operations included; a stream op gets the same policy gate as any other.
//
// Deprecated fields (Arguments/Modifiers/Payload) are never emitted.
package gen

import (
        "bytes"
        "fmt"
        "go/format"
        "os"
        "path/filepath"
        "sort"
        "strings"

        "github.com/asaidimu/hestia/cmd/hestia/core/annotate"
)

// Generate writes registrations.go and policies.go into dir based on the
// annotations found there. Files that already exist are overwritten; this is
// the codegen contract (they are DO NOT EDIT outputs).
func Generate(dir string) error {
        anns, err := annotate.Parse(dir)
        if err != nil {
                return err
        }
        pkg, err := packageName(dir)
        if err != nil {
                return err
        }

        regSrc, err := renderRegistrations(pkg, anns)
        if err != nil {
                return err
        }
        if err := writeFormatted(filepath.Join(dir, "registrations.go"), regSrc); err != nil {
                return err
        }

        polSrc, err := renderPolicies(pkg, anns)
        if err != nil {
                return err
        }
        if err := writeFormatted(filepath.Join(dir, "policies.go"), polSrc); err != nil {
                return err
        }

        return nil
}

// GenerateCollector scans serviceRoot for services (dirs exposing
// RegisterService/Registrations) and writes a single collector file at
// serviceRoot/services.go exposing RegisterServices(rt) and
// CollectServiceRegistrations(rt). The collector registers every service
// provider in the runtime DI container; boot seals the container, then
// CollectServiceRegistrations resolves each service's registrations — so
// services can depend on one another. modulePath is the Go module's import
// path.
func GenerateCollector(serviceRoot, modulePath string) error {
        services, err := discoverServices(serviceRoot, modulePath)
        if err != nil {
                return err
        }
        pkg, err := packageName(serviceRoot)
        if err != nil {
                pkg = filepath.Base(serviceRoot)
        }
        src, err := renderCollector(pkg, services)
        if err != nil {
                return err
        }
        return writeFormatted(filepath.Join(serviceRoot, "services.go"), src)
}

// SeedSanitization writes a scaffolding sanitization.go into dir when the
// service does not have one yet. Unlike registrations.go and policies.go the
// sanitization file is a SEED, not a regenerated artifact: it is written once
// so feature authors (human or agent) have a documented, in-place place to
// declare mask policies for sensitive properties, and it is never overwritten
// afterwards. Reports whether the seed was written.
func SeedSanitization(dir string) (bool, error) {
        path := filepath.Join(dir, "sanitization.go")
        if _, err := os.Stat(path); err == nil {
                return false, nil
        } else if !os.IsNotExist(err) {
                return false, err
        }
        pkg, err := packageName(dir)
        if err != nil {
                return false, err
        }
        if err := writeFormatted(path, renderSanitizationSeed(pkg)); err != nil {
                return false, err
        }
        return true, nil
}

// GenerateSanitizationCollector scans serviceRoot for services declaring
// SanitizationRules() (a sanitization.go containing the function) and writes
// the module's sanitization collector — gen_sanitization.go — exposing
// allSanitizationRules: the scope-keyed map the system module feeds into
// sanitize.Registry() during Setup. Unlike the per-service sanitization file,
// the collector is a DO NOT EDIT output and is regenerated on every service
// add/generate. modulePath is the Go module's import path.
func GenerateSanitizationCollector(serviceRoot, modulePath string) error {
        services, err := discoverSanitizedServices(serviceRoot, modulePath)
        if err != nil {
                return err
        }
        pkg, err := packageName(serviceRoot)
        if err != nil {
                pkg = filepath.Base(serviceRoot)
        }
        src, err := renderSanitizationCollector(pkg, services)
        if err != nil {
                return err
        }
        return writeFormatted(filepath.Join(serviceRoot, "gen_sanitization.go"), src)
}

// discoverSanitizedServices lists direct subdirectories of serviceRoot whose
// sanitization.go declares SanitizationRules(), sorted by package name for
// deterministic output. Dirs without a sanitization file (model-only or
// sanitize-free services) are skipped — the runtime simply has no scope to
// register for them.
func discoverSanitizedServices(serviceRoot, modulePath string) ([]Service, error) {
        entries, err := os.ReadDir(serviceRoot)
        if err != nil {
                return nil, err
        }
        var services []Service
        for _, e := range entries {
                if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
                        continue
                }
                src, err := os.ReadFile(filepath.Join(serviceRoot, e.Name(), "sanitization.go"))
                if err != nil || !bytes.Contains(src, []byte("func SanitizationRules()")) {
                        continue
                }
                pkg, err := packageName(filepath.Join(serviceRoot, e.Name()))
                if err != nil {
                        continue
                }
                services = append(services, Service{
                        Package:    pkg,
                        ImportPath: serviceImportPath(serviceRoot, modulePath, e.Name()),
                })
        }
        sort.Slice(services, func(i, j int) bool { return services[i].Package < services[j].Package })
        return services, nil
}

// Service identifies one discovered service package.
type Service struct {
        // Package is the Go package name (e.g. "users").
        Package string
        // ImportPath is the full import path (e.g.
        // github.com/asaidimu/hestia/core/system/users).
        ImportPath string
}

// discoverServices lists direct subdirectories of serviceRoot that expose a
// generated registrations.go (i.e. are services), returning them sorted by
// package name for deterministic output.
func discoverServices(serviceRoot, modulePath string) ([]Service, error) {
        entries, err := os.ReadDir(serviceRoot)
        if err != nil {
                return nil, err
        }
        var services []Service
        for _, e := range entries {
                if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
                        continue
                }
                if _, err := os.Stat(filepath.Join(serviceRoot, e.Name(), "registrations.go")); err != nil {
                        continue
                }
                pkg, err := packageName(filepath.Join(serviceRoot, e.Name()))
                if err != nil {
                        continue
                }
                services = append(services, Service{
                        Package:    pkg,
                        ImportPath: serviceImportPath(serviceRoot, modulePath, e.Name()),
                })
        }
        sort.Slice(services, func(i, j int) bool { return services[i].Package < services[j].Package })
        return services, nil
}

// serviceImportPath computes the import path of a service package under
// serviceRoot. When serviceRoot is absolute the module root is located by
// walking up to go.mod; otherwise serviceRoot is treated as relative to the
// module root.
func serviceImportPath(serviceRoot, modulePath, name string) string {
        root := serviceRoot
        if filepath.IsAbs(root) {
                for {
                        if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
                                break
                        }
                        parent := filepath.Dir(root)
                        if parent == root {
                                return modulePath + "/" + name
                        }
                        root = parent
                }
                rel, err := filepath.Rel(root, filepath.Join(serviceRoot, name))
                if err != nil {
                        return modulePath + "/" + name
                }
                return modulePath + "/" + filepath.ToSlash(rel)
        }
        return modulePath + "/" + filepath.ToSlash(filepath.Join(serviceRoot, name))
}

// packageName returns the Go package name declared by the first .go file in dir.
func packageName(dir string) (string, error) {
        entries, err := os.ReadDir(dir)
        if err != nil {
                return "", err
        }
        for _, e := range entries {
                if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
                        continue
                }
                src, err := os.ReadFile(filepath.Join(dir, e.Name()))
                if err != nil {
                        return "", err
                }
                // naive: first line matching "package X"
                for _, line := range bytes.Split(src, []byte("\n")) {
                        line = bytes.TrimSpace(line)
                        if bytes.HasPrefix(line, []byte("package ")) {
                                return string(bytes.TrimPrefix(line, []byte("package "))), nil
                        }
                }
        }
        return "", fmt.Errorf("no .go files with a package clause in %s", dir)
}

func writeFormatted(path, src string) error {
        formatted, err := format.Source([]byte(src))
        if err != nil {
                return fmt.Errorf("format %s: %v", path, err)
        }
        return os.WriteFile(path, formatted, 0644)
}

// verbConst maps an annotation verb to the abstract.Verb constant name.
func verbConst(v annotate.Verb) string {
        switch v {
        case annotate.VerbCreate:
                return "abstract.Create"
        case annotate.VerbRead:
                return "abstract.Read"
        case annotate.VerbUpdate:
                return "abstract.Update"
        case annotate.VerbDelete:
                return "abstract.Delete"
        case annotate.VerbQuery:
                return "abstract.Query"
        case annotate.VerbStream:
                return "abstract.Stream"
        case annotate.VerbCheck:
                return "abstract.Check"
        }
        return "0"
}

// qual returns the qualified reference for a type plus the import path needed
// to reference it ("" for same-package types).
func qual(t annotate.TypeRef) (ref, importPath string) {
        if t.ImportPath == "" {
                return t.Name, ""
        }
        return t.Package + "." + t.Name, t.ImportPath
}
