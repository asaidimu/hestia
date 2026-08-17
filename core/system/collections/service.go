package collections

import (
	"context"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/collections/model"
	auditmodel "github.com/asaidimu/hestia/core/system/audit/model"
	usersmodel "github.com/asaidimu/hestia/core/system/users/model"
)

// CollectionsService owns the dynamic collection CRUD and document operations,
// plus the named-collection queries (system:collections:user:query → _user_).
type CollectionsService struct {
	persist  persistence.Persistence
	logger   *zap.Logger
	policy   abstract.BindingPolicyStore
	registry abstract.Registry
}

func NewCollectionsService(rt abstract.Container) (*CollectionsService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)
	policy := abstract.MustResolve[abstract.BindingPolicyStore](rt)
	registry := abstract.MustResolve[*runtime.LocalDispatcher](rt)
	return &CollectionsService{
		persist:  persist,
		logger:   logger,
		policy:   policy,
		registry: registry,
	}, nil
}

// ListCollections lists all non-system collections.
//
// @hestia.register(
//   name="system:collections:collection:list",
//   intent="read",
//   rule="administrator",
//   description="List collections",
//   output="model.CollectionListOutput",
// )
func (s *CollectionsService) ListCollections(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	return NewCollectionListHandler(s.persist)(ctx, msg)
}

// GetCollection fetches a single collection's metadata.
//
// @hestia.register(
//   name="system:collections:collection:get",
//   intent="read",
//   rule="administrator",
//   resource_id="name",
//   description="Get collection",
//   output="model.CollectionOutput",
// )
func (s *CollectionsService) GetCollection(ctx context.Context, msg abstract.Message, input *model.CollectionGetInput) (*abstract.Result, error) {
	return NewCollectionGetHandler(s.persist)(ctx, msg)
}

// CreateCollection creates a dynamic collection from a schema definition.
//
// @hestia.register(
//   name="system:collections:collection:create",
//   intent="create",
//   rule="administrator",
//   description="Create collection via API",
//   output="model.CollectionOutput",
// )
func (s *CollectionsService) CreateCollection(ctx context.Context, msg abstract.Message, input *model.CollectionCreateInput) (*abstract.Result, error) {
	return NewCollectionCreateHandler(s.persist, s.policy, s.registry, s.logger)(ctx, msg)
}

// DeleteCollection deletes a dynamic collection and its bindings.
//
// @hestia.register(
//   name="system:collections:collection:delete",
//   intent="delete",
//   rule="administrator",
//   resource_id="name",
//   description="Delete collection via API",
// )
func (s *CollectionsService) DeleteCollection(ctx context.Context, msg abstract.Message, input *model.CollectionDeleteInput) (*abstract.Result, error) {
	return NewCollectionDeleteHandler(s.persist, s.policy, s.registry, s.logger)(ctx, msg)
}

// QueryDocuments runs a QDSL query against a collection.
//
// @hestia.register(
//   name="system:collections:document:query",
//   intent="query",
//   rule="administrator",
//   description="Query collection documents",
//   output="model.CollectionQueryOutput",
// )
func (s *CollectionsService) QueryDocuments(ctx context.Context, msg abstract.Message, input *model.CollectionDocQueryInput) (*abstract.Result, error) {
	return NewCollectionQueryHandler(s.persist)(ctx, msg)
}

// CreateDocument creates a document in a collection.
//
// @hestia.register(
//   name="system:collections:document:create",
//   intent="create",
//   rule="administrator",
//   description="Create document in collection",
//   output="model.CollectionDocumentOutput",
// )
func (s *CollectionsService) CreateDocument(ctx context.Context, msg abstract.Message, input *model.CollectionDocCreateInput) (*abstract.Result, error) {
	return NewDocumentCreateHandler(s.persist)(ctx, msg)
}

// GetDocument fetches a single document from a collection.
//
// @hestia.register(
//   name="system:collections:document:get",
//   intent="read",
//   rule="administrator",
//   resource_id="doc_id",
//   description="Get document from collection",
//   output="model.CollectionDocumentOutput",
// )
func (s *CollectionsService) GetDocument(ctx context.Context, msg abstract.Message, input *model.CollectionDocGetInput) (*abstract.Result, error) {
	return NewDocumentGetHandler(s.persist)(ctx, msg)
}

// UpdateDocument updates a document in a collection.
//
// @hestia.register(
//   name="system:collections:document:update",
//   intent="update",
//   rule="administrator",
//   resource_id="doc_id",
//   description="Update document in collection",
//   output="model.CollectionDocumentOutput",
// )
func (s *CollectionsService) UpdateDocument(ctx context.Context, msg abstract.Message, input *model.CollectionDocUpdateInput) (*abstract.Result, error) {
	return NewDocumentUpdateHandler(s.persist)(ctx, msg)
}

// DeleteDocument deletes a document from a collection.
//
// @hestia.register(
//   name="system:collections:document:delete",
//   intent="delete",
//   rule="administrator",
//   resource_id="doc_id",
//   description="Delete document from collection",
// )
func (s *CollectionsService) DeleteDocument(ctx context.Context, msg abstract.Message, input *model.CollectionDocDeleteInput) (*abstract.Result, error) {
	return NewDocumentDeleteHandler(s.persist)(ctx, msg)
}

// ReadCollection runs a QDSL query against an internal collection. The
// collection is resolved from the message's name document (dispatcher-side),
// so a single method serves every internal _*_ read.
//
// @hestia.register(
//   name="system:collections:_user:read",
//   intent="read",
//   rule="administrator",
//   internal="true",
//   description="Query users collection",
// )
// @hestia.register(
//   name="system:collections:_api_key:read",
//   intent="read",
//   rule="administrator",
//   internal="true",
//   description="Query API keys collection",
// )
// @hestia.register(
//   name="system:collections:_operation_policy:read",
//   intent="read",
//   rule="administrator",
//   internal="true",
//   description="Query policy operations",
// )
// @hestia.register(
//   name="system:collections:_iam_rule:read",
//   intent="read",
//   rule="administrator",
//   internal="true",
//   description="Query policy rules",
// )
// @hestia.register(
//   name="system:collections:_access_log:read",
//   intent="read",
//   rule="administrator",
//   internal="true",
//   description="Query access logs",
// )
func (s *CollectionsService) ReadCollection(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	return NewReadCollectionHandler(s.persist)(ctx, msg)
}

// QueryUsers runs a QDSL query against the _user_ collection, pinned by the
// named-collection query helper. This restores the system:collections:user:query
// registration previously owned by the users feature.
//
// @hestia.register(
//   name="system:collections:user:query",
//   intent="query",
//   rule="administrator",
//   description="Query users collection",
//   output="usersmodel.UserQueryOutput",
// )
func (s *CollectionsService) QueryUsers(ctx context.Context, msg abstract.Message, input *usersmodel.UserQueryInput) (*abstract.Result, error) {
	return NewNamedCollectionQueryHandler("_user_", s.persist)(ctx, msg)
}

// QueryAuditLogs runs a QDSL query against the _audit_log_ collection, pinned
// by the named-collection query helper.
//
// @hestia.register(
//   name="system:collections:audit_log:query",
//   intent="query",
//   rule="administrator",
//   description="Query audit logs",
//   output="auditmodel.LogQueryOutput",
// )
func (s *CollectionsService) QueryAuditLogs(ctx context.Context, msg abstract.Message, input *auditmodel.LogQueryInput) (*abstract.Result, error) {
	return NewNamedCollectionQueryHandler("_audit_log_", s.persist)(ctx, msg)
}

// ExportAuditLogs runs a QDSL query against the _audit_log_ collection, pinned
// by the named-collection query helper.
//
// @hestia.register(
//   name="system:audit:log:export",
//   intent="update",
//   rule="administrator",
//   description="Export audit logs",
//   output="auditmodel.LogQueryOutput",
// )
func (s *CollectionsService) ExportAuditLogs(ctx context.Context, msg abstract.Message, input *auditmodel.LogQueryInput) (*abstract.Result, error) {
	return NewNamedCollectionQueryHandler("_audit_log_", s.persist)(ctx, msg)
}