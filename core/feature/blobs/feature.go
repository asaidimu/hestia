package blobs

import (
	"github.com/asaidimu/blobs/staging"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
	blobutil "github.com/asaidimu/hestia/core/feature/blobs/store"
	"github.com/asaidimu/hestia/core/runtime"
)

type Dependencies struct {
	BlobStore    blobutil.BlobStore
	Staging      *staging.Manager
	PolicyBridge abstract.BindingPolicyStore
	Registry     abstract.Registry
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:blobs:namespace:list", Handler: NewListNamespacesHandler(deps.BlobStore), Description: "List blob namespaces", Enabled: true, Intent: abstract.Query, Output: nsListOutputSchema()},
		{Name: "system:blobs:namespace:create", Handler: NewCreateNamespaceHandler(deps.BlobStore, deps.Staging, deps.PolicyBridge, deps.Registry), Description: "Create a blob namespace", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: NsCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: nsOutputSchema()},
		{Name: "system:blobs:namespace:delete", Handler: NewDeleteNamespaceHandler(deps.BlobStore, deps.PolicyBridge, deps.Registry), Description: "Delete a blob namespace", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Schema: NsInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, ResourceIDField: "ns"}},
		{Name: "system:blobs:blob:list", Handler: NewListBlobsHandler(deps.BlobStore), Description: "List blobs in a namespace", Enabled: true, Intent: abstract.Query, Input: runtime.Input{Schema: BlobListInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, Payload: definition.FieldTypeRecord}, Output: blobListOutputSchema()},
		{Name: "system:blobs:blob:head", Handler: NewHeadBlobHandler(deps.BlobStore), Description: "Get blob metadata", Enabled: true, Intent: abstract.Query, Input: runtime.Input{Schema: BlobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:upload", Handler: NewUploadBlobHandler(deps.BlobStore), Description: "Upload a blob", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: BlobUploadInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, Modifiers: map[string]definition.FieldType{"overwrite": definition.FieldTypeString}, HeaderFields: map[string]string{"Content-Type": "content_type"}, ResourceIDField: "key", Payload: definition.FieldTypeBytes}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:download", Handler: NewDownloadBlobHandler(deps.BlobStore), Description: "Download a blob", Enabled: true, Intent: abstract.Read, Input: runtime.Input{Schema: BlobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}},
		{Name: "system:blobs:blob:delete", Handler: NewDeleteBlobHandler(deps.BlobStore), Description: "Delete a blob", Enabled: true, Intent: abstract.Delete, Input: runtime.Input{Schema: BlobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}},
		{Name: "system:blobs:blob:update", Handler: NewUpdateBlobHandler(deps.BlobStore), Description: "Update blob metadata", Enabled: true, Intent: abstract.Update, Input: runtime.Input{Schema: BlobUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key", Payload: definition.FieldTypeObject}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:begin", Handler: NewBeginUploadHandler(deps.BlobStore, deps.Staging), Description: "Begin a resumable blob upload", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: BlobBeginInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, Modifiers: map[string]definition.FieldType{"overwrite": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: uploadBeginOutputSchema()},
		{Name: "system:blobs:blob:chunk", Handler: NewUploadChunkHandler(deps.Staging), Description: "Upload a chunk of a resumable blob upload", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: BlobChunkInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, HeaderFields: map[string]string{"X-Session-ID": "session_id", "X-Offset": "offset", "X-Chunk-SHA256": "sha256"}, Payload: definition.FieldTypeBytes}, Output: uploadChunkOutputSchema()},
		{Name: "system:blobs:blob:complete", Handler: NewCompleteUploadHandler(deps.BlobStore, deps.Staging), Description: "Complete a resumable blob upload", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: BlobCompleteInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, Modifiers: map[string]definition.FieldType{"overwrite": definition.FieldTypeString}, HeaderFields: map[string]string{"X-Session-ID": "session_id"}}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:progress", Handler: NewProgressUploadHandler(deps.Staging), Description: "Report progress of a resumable blob upload", Enabled: true, Intent: abstract.Query, Input: runtime.Input{Schema: BlobProgressInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, Modifiers: map[string]definition.FieldType{"session_id": definition.FieldTypeString}}, Output: uploadProgressOutputSchema()},
		{Name: "system:blobs:blob:abort", Handler: NewAbortUploadHandler(deps.Staging), Description: "Abort a resumable blob upload", Enabled: true, Intent: abstract.Create, Input: runtime.Input{Schema: BlobAbortInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, HeaderFields: map[string]string{"X-Session-ID": "session_id"}}},
	}
}
