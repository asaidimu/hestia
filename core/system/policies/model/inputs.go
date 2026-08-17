package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type PolicyBindingGetInput struct {
	Name string `input:"arguments.name"`
}

type PolicyRuleGetInput struct {
	Name string `input:"arguments.name"`
}

type PolicyRuleDeleteInput struct {
	Name string `input:"arguments.name"`
}

type PolicyRuleCreateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type PolicyRuleUpdateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type PolicyCreateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type PolicyUpdateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

// PolicyValidateInput carries an opaque payload: rule may be a CEL string or a
// rule object, context holds identity/resource/environment records.
type PolicyValidateInput struct {
	Payload map[string]any `input:"payload"`
}

func PolicyBindingGetInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[PolicyBindingGetInput]("input", true) }
func PolicyRuleGetInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[PolicyRuleGetInput]("input", true) }
func PolicyRuleDeleteInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[PolicyRuleDeleteInput]("input", true) }
func PolicyRuleCreateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[PolicyRuleCreateInput]("input", true) }
func PolicyRuleUpdateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[PolicyRuleUpdateInput]("input", true) }
func PolicyCreateInputSchema() *definition.Schema     { return dispatch.SchemaFromTypeWithTag[PolicyCreateInput]("input", true) }
func PolicyUpdateInputSchema() *definition.Schema     { return dispatch.SchemaFromTypeWithTag[PolicyUpdateInput]("input", true) }
func PolicyValidateInputSchema() *definition.Schema   { return dispatch.SchemaFromTypeWithTag[PolicyValidateInput]("input", true) }

type PolicyBindingListInput struct{}

type PolicyRuleListInput struct{}

type PolicyListInput struct{}

type PolicyReloadInput struct{}
