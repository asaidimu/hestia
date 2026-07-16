package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/app/abstract"
)

type Dependencies struct {
	BlobStore core.BlobStore
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:blobs:namespace:list", Handler: NewListNamespacesHandler(deps.BlobStore), Description: "List blob namespaces", Enabled: true, Intent: registration.Query, Output: nsListOutputSchema()},
		{Name: "system:blobs:namespace:create", Handler: NewCreateNamespaceHandler(deps.BlobStore), Description: "Create a blob namespace", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: nsCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: nsOutputSchema()},
		{Name: "system:blobs:namespace:delete", Handler: NewDeleteNamespaceHandler(deps.BlobStore), Description: "Delete a blob namespace", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: nsInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString}}},
		{Name: "system:blobs:blob:list", Handler: NewListBlobsHandler(deps.BlobStore), Description: "List blobs in a namespace", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: blobListInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString}, Payload: definition.FieldTypeRecord}, Output: blobListOutputSchema()},
		{Name: "system:blobs:blob:head", Handler: NewHeadBlobHandler(deps.BlobStore), Description: "Get blob metadata", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: blobKeyInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString, "key": definition.FieldTypeString}}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:upload", Handler: NewUploadBlobHandler(deps.BlobStore), Description: "Upload a blob", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: blobKeyInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString, "key": definition.FieldTypeString}, Payload: definition.FieldTypeBytes}, Output: blobMetaOutputSchema()},
		{Name: "system:blobs:blob:download", Handler: NewDownloadBlobHandler(deps.BlobStore), Description: "Download a blob", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: blobKeyInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString, "key": definition.FieldTypeString}}},
		{Name: "system:blobs:blob:delete", Handler: NewDeleteBlobHandler(deps.BlobStore), Description: "Delete a blob", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: blobKeyInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString, "key": definition.FieldTypeString}}},
		{Name: "system:blobs:blob:update", Handler: NewUpdateBlobHandler(deps.BlobStore), Description: "Update blob metadata", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: blobUpdateInputSchema(), Arguments: map[string]definition.FieldType{"ns": definition.FieldTypeString, "key": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: blobMetaOutputSchema()},
	}
}
