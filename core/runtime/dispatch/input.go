package dispatch

import (
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/core/schema/meta"
)

var InputMetaSchemaJSON = []byte(`{
	"name": "InputMetaSchema",
	"description": "Meta schema used to validate that registration schemas are valid",
	"version": "1.0.0",
	"fields": {
		"name": { "name": "name", "description": "Schema name", "type": "string" },
		"description": { "name": "description", "description": "Schema description", "type": "string" },
		"version": { "name": "version", "description": "Schema version", "type": "string" },
		"fields": {
			"name": "fields",
			"description": "Schema fields",
			"type": "record",
			"schema": { "id": "fields" }
		},
		"schemas": {
			"name": "schemas",
			"description": "Nested schemas",
			"type": "record"
		}
	},
	"schemas": {
		"fields": {
			"name": "fields",
			"fields": {
				"name": {
					"name": "name",
					"description": "Allowed field names",
					"required": true,
					"type": "enum",
					"schema": { "type": "string", "values": ["arguments", "modifiers", "payload", "_id_", "_metadata_"] }
				},
				"type": {
					"name": "type",
					"description": "Allowed field types",
					"required": true,
					"type": "enum",
					"schema": { "type": "string", "values": ["object", "record", "string"] }
				},
				"required": {
					"name": "required",
					"description": "required flag",
					"type": "boolean"
				},
				"description": {
					"name": "description",
					"description": "Field description",
					"type": "string"
				},
				"schema": {
					"name": "schema",
					"description": "schema reference",
					"type": "record"
				}
			}
		}
	}
}`)

func MustFromJSON(data []byte) *definition.Schema {
	s, err := definition.FromJSON(data)
	if err != nil {
		panic(err)
	}
	return s
}

// SchemaFromType generates a meta-schema from any Go struct type using anansi
// struct tags, then parses it into a *definition.Schema. Panics on error.
func SchemaFromType[T any]() *definition.Schema {
	b, err := data.SchemaFrom[T]()
	if err != nil {
		panic(err)
	}
	return MustFromJSON(b)
}

// SchemaFromTypeWithTag generates a meta-schema from any Go struct type using
// anansi struct tags for field metadata and the given tag for field name/path
// resolution, then parses it into a *definition.Schema. Panics on error.
func SchemaFromTypeWithTag[T any](tag string, omitSystemFields ...bool) *definition.Schema {
	b, err := data.SchemaFromWithTag[T](tag, omitSystemFields...)
	if err != nil {
		panic(err)
	}
	return MustFromJSON(b)
}

var inputMetaSchema = MustFromJSON(InputMetaSchemaJSON)

var _ = meta.NormalizeSchema(inputMetaSchema)

var inputSchemaValidator *definition.DocumentValidator

var validatorOnce sync.Once

func ValidateInputSchema(schema *definition.Schema) ([]common.Issue, bool) {
	validatorOnce.Do(func() {
		inputSchemaValidator, _ = definition.NewDocumentValidator(inputMetaSchema, definition.PredicateMap{})
	})
	return inputSchemaValidator.Validate(schema.AsMap())
}

var docValidators sync.Map

func getDocValidator(s *definition.Schema) (*definition.DocumentValidator, error) {
	if cached, ok := docValidators.Load(s); ok {
		return cached.(*definition.DocumentValidator), nil
	}
	v, err := definition.NewDocumentValidator(s, definition.PredicateMap{})
	if err != nil {
		return nil, err
	}
	docValidators.Store(s, v)
	return v, nil
}

func ValidateInputDocument(s *definition.Schema, doc *data.Document) ([]common.Issue, bool) {
	if s == nil {
		return nil, true
	}
	v, err := getDocValidator(s)
	if err != nil {
		return nil, true
	}
	return v.ValidateLoose(doc.ToMap())
}
