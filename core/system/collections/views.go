package collections

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/core/abstract"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// View payload keys for system:collections:view:create.
const (
	viewPayloadQuery        = "query"
	viewPayloadMaterialized = "materialized"
)

// parseViewQuery extracts the stored QDSL query from a view:create payload.
func parseViewQuery(payload map[string]any) (*query.Query, error) {
	raw, ok := payload[viewPayloadQuery]
	if !ok || raw == nil {
		return nil, common.NewSystemError("VIEW_QUERY_REQUIRED", "request body must contain a 'query' object with the stored view definition")
	}
	q, err := parseCollectionQuery(raw)
	if err != nil {
		return nil, common.NewSystemError("VIEW_QUERY_INVALID", fmt.Sprintf("invalid view query definition: %s", err.Error())).WithCause(err)
	}
	if q == nil {
		return nil, common.NewSystemError("VIEW_QUERY_INVALID", "request body 'query' must be a valid JSON query definition")
	}
	return q, nil
}

func viewTargetName(q *query.Query) string {
	if q == nil || q.Target == nil {
		return ""
	}
	return q.Target.Name
}

// resolveViewTarget attaches the underlying collection's schema to the
// stored query's target. Query JSON round-trips drop the schema, but
// CreateView needs it to derive the view's result schema.
func resolveViewTarget(ctx context.Context, persist persistence.Persistence, q *query.Query) error {
	if q == nil || q.Target == nil || q.Target.Name == "" {
		return common.NewSystemError("VIEW_TARGET_REQUIRED", "view query must specify a target collection")
	}
	if q.Target.Schema != nil {
		return nil
	}
	s, err := persist.Schema(ctx, q.Target.Name)
	if err != nil {
		return wrapErr(err, "VIEW_TARGET_UNKNOWN", fmt.Sprintf("view target collection %q does not exist", q.Target.Name))
	}
	if s == nil {
		return common.NewSystemError("VIEW_TARGET_UNKNOWN", fmt.Sprintf("view target collection %q does not exist", q.Target.Name))
	}
	q.Target.Schema = s
	return nil
}

// NewViewCreateHandler registers a read-only view collection backed by a
// stored query. Virtual views (default) re-run the stored query on every
// read; materialized views snapshot into a physical table refreshed via
// view:refresh. Views are readable through the regular document:query
// message and reject writes with ERR_PERSISTENCE_READ_ONLY.
func NewViewCreateHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		if name == "" {
			return nil, common.NewSystemError("VIEW_NAME_REQUIRED", "view name is required")
		}
		if IsSystemCollection(name) {
			return nil, common.NewSystemError("RESERVED_NAME", fmt.Sprintf("collection name %q is reserved for system use", name))
		}

		var payload map[string]any
		if raw := doc.GetOr("payload", nil); raw != nil {
			payload, _ = raw.(map[string]any)
		}
		if len(payload) == 0 {
			return nil, common.NewSystemError("VIEW_QUERY_REQUIRED", "request body must contain a 'query' object with the stored view definition")
		}

		q, err := parseViewQuery(payload)
		if err != nil {
			return nil, err
		}
		if err := resolveViewTarget(ctx, persist, q); err != nil {
			return nil, err
		}
		materialized, _ := payload[viewPayloadMaterialized].(bool)

		exists, err := persist.HasCollection(ctx, name)
		if err != nil {
			return nil, wrapErr(err, "VIEW_CHECK_FAILED", fmt.Sprintf("failed to check if collection %q exists", name))
		}
		if exists {
			return nil, common.NewSystemError("COLLECTION_EXISTS", fmt.Sprintf("collection %q already exists", name))
		}

		if _, err := persist.CreateView(ctx, name, q, materialized); err != nil {
			return nil, wrapErr(err, "VIEW_CREATE_FAILED", fmt.Sprintf("failed to create view %q", name))
		}

		return dispatch.NewDocumentResultFrom(&CollectionViewView{
			Name:         name,
			Materialized: materialized,
			Target:       viewTargetName(q),
		})
	}
}

// NewViewRefreshHandler re-populates a materialized view's snapshot from its
// stored query. Virtual views (and non-views) fail with
// ERR_PERSISTENCE_NOT_MATERIALIZED.
func NewViewRefreshHandler(persist persistence.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		if name == "" {
			return nil, common.NewSystemError("VIEW_NAME_REQUIRED", "view name is required")
		}

		if err := persist.RefreshView(ctx, name); err != nil {
			return nil, wrapErr(err, "VIEW_REFRESH_FAILED", fmt.Sprintf("failed to refresh view %q", name))
		}

		return dispatch.NewDocumentResultFrom(&CollectionViewView{
			Name:         name,
			Materialized: true,
		})
	}
}
