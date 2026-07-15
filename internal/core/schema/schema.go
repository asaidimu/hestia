package schema

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
)

func Load(fsys fs.FS, dir string) ([]*definition.Schema, error) {
	entries, err := fs.Glob(fsys, filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list schema files: %w", err)
	}

	schemas := make([]*definition.Schema, 0, len(entries))
	for _, entry := range entries {
		data, err := fs.ReadFile(fsys, entry)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry, err)
		}

		s, err := definition.FromJSON(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(entry), err)
		}

		schemas = append(schemas, s)
	}

	return schemas, nil
}

func MustFromJSON(data []byte) *definition.Schema {
	s, err := definition.FromJSON(data)
	if err != nil {
		panic(err)
	}
	return s
}

func LoadFromSubdirs(fsys fs.FS, pattern string) ([]*definition.Schema, error) {
	entries, err := fs.Glob(fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("list schema files: %w", err)
	}

	schemas := make([]*definition.Schema, 0, len(entries))
	for _, entry := range entries {
		data, err := fs.ReadFile(fsys, entry)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry, err)
		}

		s, err := definition.FromJSON(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(entry), err)
		}

		schemas = append(schemas, s)
	}

	return schemas, nil
}
