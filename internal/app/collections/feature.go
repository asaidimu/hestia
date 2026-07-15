package collections

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Dependencies struct {
	Persist      persistence.Persistence
	Registry     core.Registry
	Logger       *zap.Logger
	PolicyBridge PolicyOperationStore
}

func Registrations(deps Dependencies) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:collections:collection:list", Handler: NewCollectionListHandler(deps.Persist), Description: "List collections", Enabled: true, Intent: registration.Read, Output: collectionListOutputSchema()},
		{Name: "system:collections:collection:get", Handler: NewCollectionGetHandler(deps.Persist), Description: "Get collection", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: collectionGetInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}}, Output: collectionOutputSchema()},
		{Name: "system:collections:collection:create", Handler: NewCollectionCreateHandler(deps.Persist, deps.PolicyBridge, deps.Registry, deps.Logger), Description: "Create collection via API", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: collectionCreateInputSchema(), Payload: definition.FieldTypeObject}, Output: collectionOutputSchema()},
		{Name: "system:collections:collection:delete", Handler: NewCollectionDeleteHandler(deps.Persist, deps.PolicyBridge, deps.Registry, deps.Logger), Description: "Delete collection via API", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: collectionDeleteInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}}},
		{Name: "system:collections:document:query", Handler: NewCollectionQueryHandler(deps.Persist), Description: "Query collection documents", Enabled: true, Intent: registration.Query, Input: core.Input{Schema: collectionDocQueryInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}, Payload: definition.FieldTypeRecord}, Output: collectionQueryOutputSchema()},
		{Name: "system:collections:document:create", Handler: NewDocumentCreateHandler(deps.Persist), Description: "Create document in collection", Enabled: true, Intent: registration.Create, Input: core.Input{Schema: collectionDocCreateInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: collectionDocumentOutputSchema()},
		{Name: "system:collections:document:get", Handler: NewDocumentGetHandler(deps.Persist), Description: "Get document from collection", Enabled: true, Intent: registration.Read, Input: core.Input{Schema: collectionDocGetInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString, "doc_id": definition.FieldTypeString}}, Output: collectionDocumentOutputSchema()},
		{Name: "system:collections:document:update", Handler: NewDocumentUpdateHandler(deps.Persist), Description: "Update document in collection", Enabled: true, Intent: registration.Update, Input: core.Input{Schema: collectionDocUpdateInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString, "doc_id": definition.FieldTypeString}, Payload: definition.FieldTypeObject}, Output: collectionDocumentOutputSchema()},
		{Name: "system:collections:document:delete", Handler: NewDocumentDeleteHandler(deps.Persist), Description: "Delete document from collection", Enabled: true, Intent: registration.Delete, Input: core.Input{Schema: collectionDocDeleteInputSchema(), Arguments: map[string]definition.FieldType{"name": definition.FieldTypeString, "doc_id": definition.FieldTypeString}}},
		{Name: "system:collections:_user:read", Handler: NewReadCollectionHandler(deps.Persist), Description: "Query users collection", Enabled: true, Internal: true, Intent: registration.Read},
		{Name: "system:collections:_api_key:read", Handler: NewReadCollectionHandler(deps.Persist), Description: "Query API keys collection", Enabled: true, Internal: true, Intent: registration.Read},
		{Name: "system:collections:_policy_operation:read", Handler: NewReadCollectionHandler(deps.Persist), Description: "Query policy operations", Enabled: true, Internal: true, Intent: registration.Read},
		{Name: "system:collections:_policy_rule:read", Handler: NewReadCollectionHandler(deps.Persist), Description: "Query policy rules", Enabled: true, Internal: true, Intent: registration.Read},
		{Name: "system:collections:_access_log:read", Handler: NewReadCollectionHandler(deps.Persist), Description: "Query access logs", Enabled: true, Internal: true, Intent: registration.Read},
	}
}
