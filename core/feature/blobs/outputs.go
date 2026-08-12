package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// NamespaceView is the wire shape of a blob namespace.
type NamespaceView struct {
	ID          string `anansi:"id"`
	DisplayName string `anansi:"display_name"`
	Public      bool   `anansi:"public"`
}

// NamespaceListDocument is the body of a namespace list response.
type NamespaceListDocument struct {
	Namespaces []NamespaceView `anansi:"namespaces"`
}

// NamespaceListOutput is the envelope declaring the namespace list schema.
type NamespaceListOutput struct {
	Document NamespaceListDocument `anansi:"document"`
}

// NamespaceOutput is the envelope declaring the single-namespace schema.
type NamespaceOutput struct {
	Document NamespaceView `anansi:"document"`
}

// BlobMetaView is the wire shape of blob metadata. Timestamps are RFC3339
// strings to keep the derived schema's "string" field types stable.
type BlobMetaView struct {
	Key         string            `anansi:"key"`
	NamespaceID string            `anansi:"namespace_id"`
	ContentType string            `anansi:"content_type"`
	Size        int64             `anansi:"size"`
	CreatedAt   string            `anansi:"created_at"`
	UpdatedAt   *string           `anansi:"updated_at,omitempty"`
	Custom      map[string]string `anansi:"custom"`
}

// BlobListDocument is the body of a blob list response.
type BlobListDocument struct {
	Blobs []BlobMetaView `anansi:"blobs"`
}

// BlobListOutput is the envelope declaring the blob list schema.
type BlobListOutput struct {
	Document BlobListDocument `anansi:"document"`
}

// BlobMetaOutput is the envelope declaring the blob metadata schema.
type BlobMetaOutput struct {
	Document BlobMetaView `anansi:"document"`
}

// UploadBeginDocument is the body of a begin-upload response.
type UploadBeginDocument struct {
	SessionID string `anansi:"session_id"`
	Key       string `anansi:"key"`
	Offset    int64  `anansi:"offset"`
	BlockSize int64  `anansi:"block_size"`
}

// UploadBeginOutput is the envelope declaring the begin-upload schema.
type UploadBeginOutput struct {
	Document UploadBeginDocument `anansi:"document"`
}

// UploadChunkDocument is the body of a chunk-write response.
type UploadChunkDocument struct {
	Total int64 `anansi:"total"`
}

// UploadChunkOutput is the envelope declaring the chunk-write schema.
type UploadChunkOutput struct {
	Document UploadChunkDocument `anansi:"document"`
}

// ByteRange is a half-open [Start, End) byte range.
type ByteRange struct {
	Start int64 `anansi:"start"`
	End   int64 `anansi:"end"`
}

// UploadProgressDocument is the body of a progress response.
type UploadProgressDocument struct {
	Total        int64       `anansi:"total"`
	Ranges       []ByteRange `anansi:"ranges"`
	BlockSize    int64       `anansi:"block_size"`
	ExpectedSize int64       `anansi:"expected_size"`
}

// UploadProgressOutput is the envelope declaring the progress schema.
type UploadProgressOutput struct {
	Document UploadProgressDocument `anansi:"document"`
}

func nsListOutputSchema() *definition.Schema { return dispatch.SchemaFromType[NamespaceListOutput]() }
func nsOutputSchema() *definition.Schema     { return dispatch.SchemaFromType[NamespaceOutput]() }
func blobListOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[BlobListOutput]()
}
func blobMetaOutputSchema() *definition.Schema { return dispatch.SchemaFromType[BlobMetaOutput]() }
func uploadBeginOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[UploadBeginOutput]()
}
func uploadChunkOutputSchema() *definition.Schema { return dispatch.SchemaFromType[UploadChunkOutput]() }
func uploadProgressOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[UploadProgressOutput]()
}
