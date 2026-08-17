package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the project configuration read from hestia.json. Fields:
//
//   - module:   the Go module import path (e.g. github.com/user/app).
//   - modules:  directories where modules live. Accepts a single string or a
//     list; modules are scanned from every entry, and new modules/services are
//     scaffolded into the first entry. Defaults to ["module"].
//   - autogen:  where generated code (the module registry) is written.
//     Defaults to "internal/autogen".
//   - cmds:     names of entry points created via 'add cmd' (bookkeeping).
type Config struct {
	Module  string   `json:"module,omitempty"`
	Modules Modules  `json:"modules,omitempty"`
	Autogen string   `json:"autogen,omitempty"`
	Cmds    []string `json:"cmds,omitempty"`
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

func writeConfig(root string, cfg Config) {
	path := filepath.Join(root, "hestia.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", path)
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