package runtime

import (
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/core/schema/meta"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/schema"
)

type Input = abstract.Input

var inputMetaSchema = schema.MustFromJSON(schema.InputMetaSchemaJSON)

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
