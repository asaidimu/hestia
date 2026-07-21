package runtime

import (
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/common"
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
