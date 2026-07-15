package blobs

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/identity"
)

type hydratedMessage struct {
	core.Message
	nsInfo *core.BlobNamespaceInfo
	blobKey string
}

func (m *hydratedMessage) ResourceContext() any {
	return map[string]any{
		"namespace_id": m.nsInfo.ID,
		"public":       m.nsInfo.Public,
		"blob_key":     m.blobKey,
	}
}

// NewDispatcher creates a NamespacedDispatcher that intercepts blobs:* messages,
// fetches blob namespace metadata using a system identity, and enriches the
// message with a ResourceContextExtractor for centralized auth evaluation.
func NewDispatcher(svc core.BlobStore, next core.Dispatcher) *core.NamespacedDispatcher {
	return core.NewNamespacedDispatcher("blobs:", next, func(msg core.Message) (core.Message, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			return nil, fmt.Errorf("blob hydrator: missing namespace id in message %q", msg.Name())
		}

		sysCtx := identity.SystemContext(context.Background())

		nsInfo, err := svc.GetNamespace(sysCtx, nsID)
		if err != nil {
			return nil, fmt.Errorf("blob hydrator: fetch namespace %q: %w", nsID, err)
		}

		key, _ := msg.Input().GetOr("arguments.key", "").(string)

		return &hydratedMessage{Message: msg, nsInfo: nsInfo, blobKey: key}, nil
	})
}

var _ core.ResourceContextExtractor = (*hydratedMessage)(nil)
