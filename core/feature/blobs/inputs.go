package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type NsInput struct {
	NS string `input:"arguments.ns"`
}

type NsCreateInput struct {
	Payload map[string]any `input:"payload"`
}

type BlobKeyInput struct {
	NS  string `input:"arguments.ns"`
	Key string `input:"arguments.key"`
}

type BlobListInput struct {
	NS      string         `input:"arguments.ns"`
	Payload map[string]any `input:"payload"`
}

type BlobUpdateInput struct {
	NS      string         `input:"arguments.ns"`
	Key     string         `input:"arguments.key"`
	Payload map[string]any `input:"payload"`
}

type BlobUploadInput struct {
	NS      string `input:"arguments.ns"`
	Key     string `input:"arguments.key"`
	Payload []byte `input:"payload"`
}

func NsInputSchema() *definition.Schema         { return dispatch.SchemaFromTypeWithTag[NsInput]("input", true) }
func NsCreateInputSchema() *definition.Schema   { return dispatch.SchemaFromTypeWithTag[NsCreateInput]("input", true) }
func BlobKeyInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[BlobKeyInput]("input", true) }
func BlobListInputSchema() *definition.Schema   { return dispatch.SchemaFromTypeWithTag[BlobListInput]("input", true) }
func BlobUpdateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[BlobUpdateInput]("input", true) }
func BlobUploadInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[BlobUploadInput]("input", true) }
