package schema_test

import (
	"testing"

	"github.com/asaidimu/hestia/app/core/schema"
)

func TestInputMetaSchemaJSON(t *testing.T) {
	if len(schema.InputMetaSchemaJSON) == 0 {
		t.Fatal("InputMetaSchemaJSON must not be empty")
	}
}

func TestMustFromJSON(t *testing.T) {
	s := schema.MustFromJSON(schema.InputMetaSchemaJSON)
	if s == nil {
		t.Fatal("MustFromJSON returned nil")
	}
	if s.Name != "InputMetaSchema" {
		t.Fatalf("expected Name %q, got %q", "InputMetaSchema", s.Name)
	}
	if s.Description == "" {
		t.Fatal("expected non-empty Description")
	}
	if len(s.Fields) == 0 {
		t.Fatal("expected at least one Field")
	}
	if s.Version == nil {
		t.Fatal("expected non-nil Version")
	}
}

func TestMustFromJSONPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid JSON")
		}
	}()
	schema.MustFromJSON([]byte(`{invalid}`))
	_ = 0
}
