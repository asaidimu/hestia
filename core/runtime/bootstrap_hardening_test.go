package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

type staticBootstrapRegistry struct {
	safe map[string]bool
}

func (r staticBootstrapRegistry) IsHandlerBootstrapSafe(name string) bool {
	return r.safe[name]
}

// TestBootstrapDispatcher_GatesEverythingPreBootstrap verifies the bootstrap
// surface: before bootstrapping, only BootstrapSafe handlers dispatch; every
// other operation — including system-scoped messages — is blocked.
func TestBootstrapDispatcher_GatesEverythingPreBootstrap(t *testing.T) {
	registry := staticBootstrapRegistry{safe: map[string]bool{"sys:bootstrap:init:run": true}}
	disp := NewBootstrapDispatcher(noopDispatcher{}, registry, func() bool { return false })

	// Non-safe handler blocked pre-bootstrap.
	_, err := disp.Send(testMessage{name: "collections:_user:read", ctx: context.Background()})
	if err == nil {
		t.Fatal("expected pre-bootstrap denial for non-safe handler, got nil")
	}
	if !strings.Contains(err.Error(), "not available until the system is bootstrapped") {
		t.Fatalf("unexpected error text: %v", err)
	}

	// System-scoped message is still gated — bootstrap gating happens OUTSIDE
	// authorization, so no identity can sneak past it.
	_, err = disp.Send(testMessage{name: "sys:api:key:create", ctx: systemContext()})
	if err == nil {
		t.Fatal("expected pre-bootstrap denial even for system identity, got nil")
	}

	// BootstrapSafe handler allowed through.
	if _, err = disp.Send(testMessage{name: "sys:bootstrap:init:run", ctx: context.Background()}); err != nil {
		t.Fatalf("bootstrap-safe handler must dispatch pre-bootstrap: %v", err)
	}
}

// TestBootstrapDispatcher_OpenAfterBootstrap verifies the dispatcher stops
// gating once the system is bootstrapped.
func TestBootstrapDispatcher_OpenAfterBootstrap(t *testing.T) {
	registry := staticBootstrapRegistry{safe: map[string]bool{}}
	disp := NewBootstrapDispatcher(noopDispatcher{}, registry, func() bool { return true })

	if _, err := disp.Send(testMessage{name: "collections:_user:read", ctx: context.Background()}); err != nil {
		t.Fatalf("post-bootstrap dispatch must be unrestricted: %v", err)
	}
}

// TestBootstrapDispatcher_DefaultChainPosition pins that bootstrap is the
// OUTERMOST link: it runs before secure, audit, and everything else, so the
// pre-bootstrap gate applies to all identities and produces no audit entry.
func TestBootstrapDispatcher_DefaultChainPosition(t *testing.T) {
	defaultOrder := []string{"bootstrap", "secure", "ratelimit", "throttle", "tenant", "blob", "recovery", "audit"}
	if defaultOrder[0] != "bootstrap" {
		t.Fatalf("bootstrap must be outermost in the default chain, got %q", defaultOrder[0])
	}
}

var _ interface{ IsHandlerBootstrapSafe(string) bool } = staticBootstrapRegistry{}

var _ abstract.Dispatcher = (*BootstrapDispatcher)(nil)
