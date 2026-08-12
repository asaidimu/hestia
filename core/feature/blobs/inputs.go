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
	NS        string `input:"arguments.ns"`
	Key       string `input:"arguments.key"`
	ContentType string `input:"headers.content_type"`
	Overwrite string `input:"modifiers.overwrite"`
	Payload   []byte `input:"payload"`
}

type BlobBeginInput struct {
	NS        string         `input:"arguments.ns"`
	Overwrite string         `input:"modifiers.overwrite"`
	Payload   map[string]any `input:"payload"`
}

type BlobChunkInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"headers.session_id"`
	Offset    string `input:"headers.offset"`
	SHA256    string `input:"headers.sha256"`
	Payload   []byte `input:"payload"`
}

type BlobCompleteInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"headers.session_id"`
	Overwrite string `input:"modifiers.overwrite"`
}

type BlobAbortInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"headers.session_id"`
}

type BlobProgressInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"modifiers.session_id"`
}

func NsInputSchema() *definition.Schema         { return dispatch.SchemaFromTypeWithTag[NsInput]("input", true) }
func NsCreateInputSchema() *definition.Schema   { return dispatch.SchemaFromTypeWithTag[NsCreateInput]("input", true) }
func BlobKeyInputSchema() *definition.Schema    { return dispatch.SchemaFromTypeWithTag[BlobKeyInput]("input", true) }
func BlobListInputSchema() *definition.Schema   { return dispatch.SchemaFromTypeWithTag[BlobListInput]("input", true) }
func BlobUpdateInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[BlobUpdateInput]("input", true) }
func BlobUploadInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[BlobUploadInput]("input", true) }
func BlobBeginInputSchema() *definition.Schema  { return dispatch.SchemaFromTypeWithTag[BlobBeginInput]("input", true) }
func BlobChunkInputSchema() *definition.Schema  { return dispatch.SchemaFromTypeWithTag[BlobChunkInput]("input", true) }
func BlobCompleteInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[BlobCompleteInput]("input", true)
}
func BlobAbortInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[BlobAbortInput]("input", true) }
func BlobProgressInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[BlobProgressInput]("input", true)
}
