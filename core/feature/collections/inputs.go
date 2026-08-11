package collections

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type CollectionGetInput struct {
	Name string `input:"arguments.name"`
}

type CollectionCreateInput struct {
	Payload map[string]any `input:"payload"`
}

type CollectionDeleteInput struct {
	Name string `input:"arguments.name"`
}

type CollectionDocQueryInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type CollectionDocCreateInput struct {
	Name    string         `input:"arguments.name"`
	Payload map[string]any `input:"payload"`
}

type CollectionDocGetInput struct {
	Name  string `input:"arguments.name"`
	DocID string `input:"arguments.doc_id"`
}

type CollectionDocUpdateInput struct {
	Name    string         `input:"arguments.name"`
	DocID   string         `input:"arguments.doc_id"`
	Payload map[string]any `input:"payload"`
}

type CollectionDocDeleteInput struct {
	Name  string `input:"arguments.name"`
	DocID string `input:"arguments.doc_id"`
}

func CollectionGetInputSchema() *definition.Schema       { return dispatch.SchemaFromTypeWithTag[CollectionGetInput]("input", true) }
func CollectionCreateInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[CollectionCreateInput]("input", true) }
func CollectionDeleteInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[CollectionDeleteInput]("input", true) }
func CollectionDocQueryInputSchema() *definition.Schema  { return dispatch.SchemaFromTypeWithTag[CollectionDocQueryInput]("input", true) }
func CollectionDocCreateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[CollectionDocCreateInput]("input", true) }
func CollectionDocGetInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[CollectionDocGetInput]("input", true) }
func CollectionDocUpdateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[CollectionDocUpdateInput]("input", true) }
func CollectionDocDeleteInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[CollectionDocDeleteInput]("input", true) }
