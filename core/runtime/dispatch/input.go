// @note #arch-20260821-007 issue status=open priority=P1 tags=#arch,#global-state : Global mutable state and unbounded cache in dispatch/input.go
//
// This file contains multiple pieces of global mutable state:
//
// 1. InputMetaSchemaJSON (line 12) - exported mutable byte slice
// 2. inputMetaSchema (line 106) - package-init singleton
// 3. inputSchemaValidator (line 110) - global lazy singleton via sync.Once
// 4. docValidators (line 121) - global sync.Map cache that never evicts
//
// The validatorOnce.Do callback ignores the error from definition.NewDocumentValidator
// (line 116). If the validator fails to initialize, inputSchemaValidator will be nil
// and the subsequent Validate call will panic with a nil pointer dereference.
//
// The docValidators cache grows unboundedly, which could be a memory issue in
// long-running applications with many schemas.
//
// Resolution: Store the validator error and return it from ValidateInputSchema,
// or panic immediately with a clear message. Consider adding eviction to
// docValidators or using a bounded cache.
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
					"schema": { "type": "string", "values": ["arguments", "modifiers", "headers", "payload", "_id_", "_metadata_"] }
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

// @note #review-20260821-010 issue status=open priority=P2 tags=#review,#error-handling : Swallowed error in ValidateInputSchema
// The validatorOnce.Do callback ignores the error from definition.NewDocumentValidator.
// If the validator fails to initialize, inputSchemaValidator will be nil and the
// subsequent Validate call will panic with a nil pointer dereference.
//
// Consider storing the error and returning it from ValidateInputSchema, or
// panicking immediately with a clear message if the validator cannot be created.
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

// @note #mem-20260821-001 issue status=open priority=P0 tags=#memory,#leak : Unbounded validator cache grows indefinitely
//
// docValidators (line 140) is a sync.Map that caches DocumentValidator
// instances but never evicts them. In long-running applications:
//
// - Each unique schema creates a new validator entry
// - Validators are never removed even if schemas are deleted
// - Memory grows linearly with schema count
//
// For IoT/HFT deployments:
// - IoT devices have limited memory
// - HFT systems need predictable memory usage
// - Memory leaks cause OOM kills
//
// Resolution:
// 1. Use a bounded cache with LRU eviction (e.g., sync.Pool or go-cache)
// 2. Add TTL-based eviction for unused validators
// 3. Consider caching by schema hash instead of pointer (allows GC)
// 4. Add metrics to monitor cache size
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
