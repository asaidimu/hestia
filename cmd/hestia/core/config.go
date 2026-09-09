package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the project configuration read from hestia.json. Fields:
//
//   - module:      the Go module import path (e.g. github.com/user/app).
//   - core_dir:    directory holding shared services, always loaded.
//     Defaults to "core".
//   - modules_dir: directory holding feature modules, selected per command.
//     Defaults to "modules".
//   - autogen:     where generated code (the module registry) is written.
//     Defaults to "internal/autogen".
//   - commands:    named entry points. Each command declares the feature
//     modules it uses; core modules are always loaded implicitly.
//   - modules:     LEGACY — directories where modules live. Accepts a single
//     string or a list. Honoured when core_dir/modules_dir are unset so old
//     projects keep working. New projects use core_dir + modules_dir.
//   - cmds:        LEGACY — names of entry points created via 'add cmd'.
type Config struct {
	Module     string                   `json:"module,omitempty"`
	CoreDir    string                   `json:"core_dir,omitempty"`
	ModulesDir string                   `json:"modules_dir,omitempty"`
	Autogen    string                   `json:"autogen,omitempty"`
	Commands   map[string]CommandConfig `json:"commands,omitempty"`
	Modules    Modules                  `json:"modules,omitempty"`
	Cmds       []string                 `json:"cmds,omitempty"`
}

// CommandConfig describes a single binary entry point: where its main.go
// lives and which feature modules it loads.
type CommandConfig struct {
	Entry string   `json:"entry"`
	Uses  []string `json:"uses,omitempty"`
}

// CoreDirResolved returns the configured core directory or the default.
func (c Config) CoreDirResolved() string {
	if c.CoreDir != "" {
		return c.CoreDir
	}
	return "core"
}

// ModulesDirResolved returns the configured feature-module directory or the
// default. When only the legacy "modules" list is set, its first entry is
// used so old projects keep working.
func (c Config) ModulesDirResolved() string {
	if c.ModulesDir != "" {
		return c.ModulesDir
	}
	if len(c.Modules) > 0 {
		return c.Modules[0]
	}
	return "modules"
}

// AutogenResolved returns the configured autogen directory or the default.
func (c Config) AutogenResolved() string {
	if c.Autogen != "" {
		return c.Autogen
	}
	return "internal/autogen"
}

// UsesFor returns the feature modules a command declares.
func (c Config) UsesFor(cmd string) []string {
	if c.Commands == nil {
		return nil
	}
	return c.Commands[cmd].Uses
}

// effectiveModulesDirs returns the directories to scan for modules. New-style
// configs (core_dir/modules_dir/commands set) scan both core and modules
// dirs; legacy configs keep scanning the old "modules" list.
func effectiveModulesDirs(cfg Config) []string {
	if cfg.CoreDir != "" || cfg.ModulesDir != "" || len(cfg.Commands) > 0 {
		return []string{cfg.CoreDirResolved(), cfg.ModulesDirResolved()}
	}
	if len(cfg.Modules) > 0 {
		return []string(cfg.Modules)
	}
	return []string{"module"}
}

// Modules is a list of directories that hold modules. It unmarshals from
// either a single string or a JSON array; the first entry is the target used
// when scaffolding new modules/services.
type Modules []string

func (m *Modules) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*m = Modules{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*m = Modules(list)
	return nil
}

func readConfig(root string) Config {
	path := filepath.Join(root, "hestia.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

func writeConfig(root string, cfg Config) error {
	path := filepath.Join(root, "hestia.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("Wrote %s\n", path)
	return nil
}

func detectModulePath(root string) string {
	// Walk up from root looking for go.mod
	dir := root
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			// Read the first line: "module <path>"
			var line string
			for i := 0; i < len(data); i++ {
				if data[i] == '\n' {
					line = string(data[:i])
					break
				}
			}
			if len(line) > 7 && line[:7] == "module " {
				modPath := line[7:]
				// If root is a subdirectory of where go.mod is, append the relative path
				rel, err := filepath.Rel(dir, root)
				if err == nil && rel != "." {
					return modPath + "/" + filepath.ToSlash(rel)
				}
				return modPath
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}