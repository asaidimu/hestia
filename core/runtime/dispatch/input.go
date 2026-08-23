package dispatch

import (
	"fmt"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/cache"
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
					"schema": { "type": "string", "values": ["arguments", "modifiers", "context", "payload", "_id_", "_metadata_"] }
				},
				"type": {
					"name": "type",
					"description": "Allowed field types",
					"required": true,
					"type": "enum",
					"schema": { "type": "string", "values": ["object", "record", "string", "bytes"] }
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
var validatorInitErr error

var validatorOnce sync.Once

func ValidateInputSchema(schema *definition.Schema) ([]common.Issue, bool) {
	validatorOnce.Do(func() {
		inputSchemaValidator, validatorInitErr = definition.NewDocumentValidator(inputMetaSchema, definition.PredicateMap{})
	})
	if validatorInitErr != nil {
		return []common.Issue{{Code: "INTERNAL_ERROR", Message: "input schema validator init failed: " + validatorInitErr.Error()}}, false
	}
	return inputSchemaValidator.Validate(schema.AsMap())
}

var docValidatorCache = cache.NewManagedCache[*definition.DocumentValidator](cache.CacheConfig{
	MaxEntries:  1024,
	ShardCount:  4,
	PositiveTTL: 0, // validators are process-lifetime; no TTL eviction
}, nil)

func getDocValidator(s *definition.Schema) (*definition.DocumentValidator, error) {
	key := fmt.Sprintf("%p", s)
	if v, ok := docValidatorCache.Get(key); ok {
		return v, nil
	}
	v, err := definition.NewDocumentValidator(s, definition.PredicateMap{})
	if err != nil {
		return nil, err
	}
	docValidatorCache.Set(key, v)
	return v, nil
}

func ValidateInputDocument(s *definition.Schema, doc data.Documenter) ([]common.Issue, bool) {
	if s == nil {
		return nil, true
	}
	v, err := getDocValidator(s)
	if err != nil {
		return nil, true
	}
	return v.ValidateLoose(doc.ToMap())
}
