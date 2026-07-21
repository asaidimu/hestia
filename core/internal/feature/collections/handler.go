package collections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/core/schema/meta"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/registration"
)

type OperationPolicyStore interface {
	EnsureOperation(ctx context.Context, name, ruleKey, intentType, description string) error
	DeleteOperation(ctx context.Context, name string) error
	ForceDeleteOperation(ctx context.Context, name string) error
	EnsureRule(ctx context.Context, name, expr, description string) error
	DeleteRule(ctx context.Context, name string) error
	ForceDeleteRule(ctx context.Context, name string) error
	ReloadPolicies(ctx context.Context) error
}

func wrapErr(err error, code, msg string) *common.SystemError {
	if sysErr, ok := errors.AsType[*common.SystemError](err); ok {
		return common.NewSystemError(code, fmt.Sprintf("%s: %s", msg, sysErr.Error())).
			WithCause(sysErr)
	}
	return common.NewSystemError(code, fmt.Sprintf("%s: %s", msg, err.Error()))
}

func NewCollectionCreateHandler(persist persistence.Persistence, policyOp OperationPolicyStore, registry runtime.Registry, logger *zap.Logger) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		bodyRaw := doc.GetOr("payload", nil)
		var schemaBytes []byte
		if bodyRaw != nil {
			schemaBytes, _ = json.Marshal(bodyRaw)
		}

		if len(schemaBytes) == 0 {
			return nil, common.NewSystemError("SCHEMA_REQUIRED", "request body must be a valid Anansi schema definition")
		}

		s, err := definition.FromJSON(schemaBytes)
		if err != nil {
			return nil, wrapErr(err, "PARSE_SCHEMA", "invalid schema JSON")
		}

		name := s.Name
		if name == "" {
			return nil, common.NewSystemError("SCHEMA_MISSING_NAME", "schema must have a 'name' field")
		}

		exists, err := persist.HasCollection(ctx, name)
		if err != nil {
			return nil, wrapErr(err, "COLLECTION_CHECK_FAILED", fmt.Sprintf("failed to check if collection %q exists", name))
		}

		if exists {
			return nil, common.NewSystemError("COLLECTION_EXISTS", fmt.Sprintf("collection %q already exists", name))
		}

		if IsSystemCollection(name) {
			return nil, common.NewSystemError("RESERVED_NAME", fmt.Sprintf("collection name %q is reserved for system use", name))
		}

		meta.NormalizeSchema(s)
		if issues, err := schema.ValidateSchema(s); err != nil || len(issues) > 0 {
			return nil, common.NewSystemError("INVALID_SCHEMA").WithIssues(issues)
		}

		created, err := persist.CreateCollection(ctx, s)
		if err != nil {
			return nil, wrapErr(err, "CREATE_COLLECTION", fmt.Sprintf("failed to create collection %q", name))
		}

		type opDef struct {
			suffix, ruleKey, intent, desc string
		}
		ops := []opDef{
			{"read", "authenticated", "QUERY", "Query " + name + " collection"},
			{"write", "authenticated", "COMMAND", "Write to " + name + " collection"},
			{"delete", "authenticated", "COMMAND", "Delete from " + name + " collection"},
			{"document.read", "authenticated", "QUERY", "Get a document from " + name},
			{"document.create", "administrator", "COMMAND", "Create a document in " + name},
			{"document.update", "administrator", "COMMAND", "Update a document in " + name},
			{"document.delete", "administrator", "COMMAND", "Delete a document from " + name},
		}
		for _, op := range ops {
			opName := "collection." + name + "." + op.suffix
			if err := policyOp.EnsureOperation(ctx, opName, op.ruleKey, op.intent, op.desc); err != nil {
				return nil, wrapErr(err, "ENSURE_OPERATION", fmt.Sprintf("register operation %s", opName))
			}
		}

		if err := policyOp.ReloadPolicies(ctx); err != nil {
			logger.Warn("Policy reload after collection create failed", zap.String("collection", name), zap.Error(err))
		}

		if err := RegisterDocumentHandlers(registry, persist, name); err != nil {
			return nil, wrapErr(err, "REGISTER_DOCUMENT_HANDLERS", fmt.Sprintf("register document handlers for collection %q", name))
		}

		metadata := created.Metadata(context.Background(), nil, false)
		now := time.Now().UTC().Format(time.RFC3339)
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{
				"name":    name,
				"schema":  metadata.Schema,
				"created": now,
				"updated": now,
			}, ctx),
		}, nil
	}
}

func NewCollectionDeleteHandler(persist persistence.Persistence, policyOp OperationPolicyStore, registry runtime.Registry, logger *zap.Logger) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if IsSystemCollection(name) {
			return nil, fmt.Errorf("collection %q is a system collection and cannot be deleted", name)
		}

		docSuffixes := []string{"document.create", "document.read", "document.update", "document.delete"}
		for _, s := range docSuffixes {
			registry.DeleteHandler("collection." + name + "." + s)
		}

		if _, err := persist.Delete(ctx, name); err != nil {
			return nil, fmt.Errorf("delete collection %q: %w", name, err)
		}

		suffixes := []string{"read", "write", "delete", "document.create", "document.read", "document.update", "document.delete"}
		for _, s := range suffixes {
			opName := "collection." + name + "." + s
			if err := policyOp.ForceDeleteOperation(ctx, opName); err != nil {
				logger.Warn("Failed to delete operation", zap.String("operation", opName), zap.Error(err))
			}
		}

		if err := policyOp.ReloadPolicies(ctx); err != nil {
			logger.Warn("Policy reload after collection delete failed", zap.String("collection", name), zap.Error(err))
		}

		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"name": name}, ctx),
		}, nil
	}
}

var iamProtectedCollections = map[string]struct{}{
	"_iam_rule_":        {},
	"_operation_policy_": {},
}

func isIAMProtectedCollection(name string) bool {
	_, ok := iamProtectedCollections[name]
	return ok
}

func NewDocumentCreateHandler(persist persistence.Persistence) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if isIAMProtectedCollection(name) {
			return nil, common.NewSystemError("PROTECTED_COLLECTION", fmt.Sprintf("direct writes to %q are not allowed; use the dedicated policy API", name))
		}

		bodyRaw := doc.GetOr("payload", nil)

		var body map[string]any
		if bodyRaw != nil {
			body, _ = bodyRaw.(map[string]any)
		}
		if len(body) == 0 {
			return nil, common.NewSystemError("DOCUMENT_REQUIRED", "request body must be a valid JSON document")
		}

		col, err := persist.Collection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("access collection %q: %w", name, err)
		}

		createdDoc := data.MustNewDocument(body)
		created, err := col.CreateOne(ctx, createdDoc)
		if err != nil {
			return nil, fmt.Errorf("create document in %q: %w", name, err)
		}

		return &registration.Result{Document: created.Data}, nil
	}
}

func NewDocumentDeleteHandler(persist persistence.Persistence) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if isIAMProtectedCollection(name) {
			return nil, common.NewSystemError("PROTECTED_COLLECTION", fmt.Sprintf("direct writes to %q are not allowed; use the dedicated policy API", name))
		}

		documentID, _ := doc.GetOr("arguments.doc_id", "").(string)

		col, err := persist.Collection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("access collection %q: %w", name, err)
		}

		filter := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(documentID).Build().Filters
		deleted, err := col.Delete(ctx, filter, false)
		if err != nil {
			return nil, fmt.Errorf("delete document %q from %q: %w", documentID, name, err)
		}
		if deleted == 0 {
			return nil, fmt.Errorf("document %q not found in %q", documentID, name)
		}

		return &registration.Result{}, nil
	}
}

func NewDocumentUpdateHandler(persist persistence.Persistence) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)

		if isIAMProtectedCollection(name) {
			return nil, common.NewSystemError("PROTECTED_COLLECTION", fmt.Sprintf("direct writes to %q are not allowed; use the dedicated policy API", name))
		}

		documentID, _ := doc.GetOr("arguments.doc_id", "").(string)
		bodyRaw := doc.GetOr("payload", nil)

		var body map[string]any
		if bodyRaw != nil {
			body, _ = bodyRaw.(map[string]any)
		}
		if len(body) == 0 {
			return nil, common.NewSystemError("DOCUMENT_REQUIRED", "request body must be a valid JSON document")
		}

		col, err := persist.Collection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("access collection %q: %w", name, err)
		}

		filter := query.NewQueryBuilder().Where(data.DocumentIDField).Eq(documentID).Build().Filters
		setDoc := data.Patch(body).Document(ctx)
		result, err := col.Update(ctx, &persistence.CollectionUpdate{
			Set:            setDoc,
			Filter:         filter,
			ReturnDocument: true,
		})
		if err != nil {
			return nil, fmt.Errorf("update document %q in %q: %w", documentID, name, err)
		}

		if len(result.Data) == 0 {
			return nil, common.NewSystemError("DOCUMENT_NOT_FOUND", fmt.Sprintf("document %q not found in %q", documentID, name))
		}

		return &registration.Result{Document: result.Data[0]}, nil
	}
}

func RegisterDocumentHandlers(r runtime.Registry, persist persistence.Persistence, name string) error {
	createHandler := NewDocumentCreateHandler(persist)
	getHandler := NewDocumentGetHandler(persist)
	updateHandler := NewDocumentUpdateHandler(persist)
	deleteHandler := NewDocumentDeleteHandler(persist)

	docCmdInfo := runtime.HandlerInfo{Description: fmt.Sprintf("Create a document in %q", name), Enabled: true}
	docQryInfo := runtime.HandlerInfo{Description: fmt.Sprintf("Get a document from %q", name), Enabled: true}
	docUpdInfo := runtime.HandlerInfo{Description: fmt.Sprintf("Update a document in %q", name), Enabled: true}
	docDelInfo := runtime.HandlerInfo{Description: fmt.Sprintf("Delete a document from %q", name), Enabled: true}

	if err := r.RegisterHandler("collection."+name+".document.create", createHandler, docCmdInfo); err != nil {
		return err
	}
	if err := r.RegisterHandler("collection."+name+".document.read", getHandler, docQryInfo); err != nil {
		return err
	}
	if err := r.RegisterHandler("collection."+name+".document.update", updateHandler, docUpdInfo); err != nil {
		return err
	}
	if err := r.RegisterHandler("collection."+name+".document.delete", deleteHandler, docDelInfo); err != nil {
		return err
	}

	return nil
}
