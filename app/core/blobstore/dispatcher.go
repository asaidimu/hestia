package blobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/identity"
)

type renamedMessage struct {
	core.Message
	name string
}

func (m *renamedMessage) Name() string { return m.name }

// NewDispatcher creates a NamespacedDispatcher that intercepts system:blobs:blob:* messages,
// rewrites the message name from system:blobs:blob:{action} to blob.{ns}.{action}
// for per-namespace policy resolution, and injects the namespace info.
func NewDispatcher(svc core.BlobStore, next core.Dispatcher) *core.NamespacedDispatcher {
	return core.NewNamespacedDispatcher("system:blobs:blob:", next, func(msg core.Message) (core.Message, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			return nil, fmt.Errorf("blob hydrator: missing namespace id in message %q", msg.Name())
		}

		// Verify namespace exists
		sysCtx := identity.SystemContext(context.Background())
		if _, err := svc.GetNamespace(sysCtx, nsID); err != nil {
			return nil, fmt.Errorf("blob hydrator: fetch namespace %q: %w", nsID, err)
		}

		// Rewrite system:blobs:blob:{action} → blob.{ns}.{action}
		newName := msg.Name()
		if strings.HasPrefix(newName, "system:blobs:blob:") {
			action := strings.TrimPrefix(newName, "system:blobs:blob:")
			newName = "blob." + nsID + "." + action
		}

		return &renamedMessage{Message: msg, name: newName}, nil
	})
}
