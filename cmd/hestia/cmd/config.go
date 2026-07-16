package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Module        string   `json:"module,omitempty"`
	ModuleSources []string `json:"module_sources,omitempty"`
	ModuleTarget  string   `json:"module_target,omitempty"`
	Cmds          []string `json:"cmds,omitempty"`
	AutogenTarget string   `json:"autogen_target,omitempty"`
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
