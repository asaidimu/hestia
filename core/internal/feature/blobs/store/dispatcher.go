package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

type renamedMessage struct {
	abstract.Message
	name string
}

func (m *renamedMessage) Name() string { return m.name }

// NewDispatcher creates a NamespacedDispatcher that intercepts system:blobs:blob:* messages,
// rewrites the message name from system:blobs:blob:{action} to system:blobs:{ns}:{action}
// for per-namespace policy resolution, and injects the namespace info.
func NewDispatcher(svc BlobStore, next abstract.Dispatcher) *runtime.NamespacedDispatcher {
	return runtime.NewNamespacedDispatcher("system:blobs:blob:", next, blobHydrator(svc))
}

// NewDispatcherLink returns the same NamespacedDispatcher without binding a next
// dispatcher, so it can be used in a DispatcherChain.
func NewDispatcherLink(svc BlobStore) *runtime.NamespacedDispatcher {
	return runtime.NewNamespacedDispatcher("system:blobs:blob:", nil, blobHydrator(svc))
}

func blobHydrator(svc BlobStore) func(abstract.Message) (abstract.Message, error) {
	return func(msg abstract.Message) (abstract.Message, error) {
		nsID, _ := msg.Input().GetOr("arguments.ns", "").(string)
		if nsID == "" {
			return nil, fmt.Errorf("blob hydrator: missing namespace id in message %q", msg.Name())
		}

		// Verify namespace exists
		sysCtx := runtimecontext.SystemContext(context.Background())
		if _, err := svc.GetNamespace(sysCtx, nsID); err != nil {
			return nil, fmt.Errorf("blob hydrator: fetch namespace %q: %w", nsID, err)
		}

		// Rewrite system:blobs:blob:{action} → system:blobs:{ns}:{action}
		newName := msg.Name()
		if strings.HasPrefix(newName, "system:blobs:blob:") {
			action := strings.TrimPrefix(newName, "system:blobs:blob:")
			newName = "system:blobs:" + nsID + ":" + action
		}

		return &renamedMessage{Message: msg, name: newName}, nil
	}
}
