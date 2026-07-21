package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
	"github.com/asaidimu/hestia/core/abstract"
)

type Dependencies struct {
	BlobStore    runtime.BlobStore
	PolicyBridge OperationPolicyStore
	Registry     runtime.Registry
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:blobs:namespace:list", Handler: NewListNamespacesHandler(deps.BlobStore), Description: "List blob namespaces", Enabled: true, Intent: registration.Query, Output: nsListOutputSchema()},
		{Name: "system:blobs:namespace:create", Handler: NewCreateNamespaceHandler(deps.BlobStore, deps.PolicyBridge, deps.Registry), Description: "Create a blob namespace", Enabled: true, Intent: registration.Create, Input: runtime.Input{Schema: nsCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: nsOutputSchema()},
		{Name: "system:blobs:namespace:delete", Handler: NewDeleteNamespaceHandler(deps.BlobStore, deps.PolicyBridge, deps.Registry), Description: "Delete a blob namespace", Enabled: true, Intent: registration.Delete, Input: runtime.Input{Schema: nsInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, ResourceIDField: "ns"}},
		{Name: "system:blobs:blob:list", Handler: NewListBlobsHandler(deps.BlobStore), Description: "List blobs in a namespace", Enabled: true, Intent: registration.Query, Input: runtime.Input{Schema: blobListInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}}, Payload: definition.FieldTypeRecord}, Output: blobListOutputSchema()},
		{Name: "system:blobs:blob:head", Handler: NewHeadBlobHandler(deps.BlobStore), Description: "Get blob metadata", Enabled: true, Intent: registration.Query, Input: runtime.Input{Schema: blobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:upload", Handler: NewUploadBlobHandler(deps.BlobStore), Description: "Upload a blob", Enabled: true, Intent: registration.Create, Input: runtime.Input{Schema: blobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key", Payload: definition.FieldTypeBytes}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:download", Handler: NewDownloadBlobHandler(deps.BlobStore), Description: "Download a blob", Enabled: true, Intent: registration.Read, Input: runtime.Input{Schema: blobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}},
		{Name: "system:blobs:blob:delete", Handler: NewDeleteBlobHandler(deps.BlobStore), Description: "Delete a blob", Enabled: true, Intent: registration.Delete, Input: runtime.Input{Schema: blobKeyInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key"}},
		{Name: "system:blobs:blob:update", Handler: NewUpdateBlobHandler(deps.BlobStore), Description: "Update blob metadata", Enabled: true, Intent: registration.Update, Input: runtime.Input{Schema: blobUpdateInputSchema(), Arguments: []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}}, ResourceIDField: "key", Payload: definition.FieldTypeObject}, Output: blobMetaOutputSchema()},
	}
}
