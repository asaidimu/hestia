package auth_test

import (
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/auth"
)

// TestSessionGetIsBootstrapSafe pins that the session probe is reachable
// before the system is bootstrapped. The UI calls it on every fresh tab to
// decide whether a session already exists; when it is not bootstrap-safe the
// pre-bootstrap gate answers 403 ("handler not available until bootstrapped")
// instead of the expected 401, and clients that distinguish "not logged in"
// from "could not check" treat the auth page as fatal.
func TestSessionGetIsBootstrapSafe(t *testing.T) {
	p := testutil.NewPersistence(t)
	svc := newTestAuthService(t, p)

	rt := runtime.NewRuntime()
	if err := abstract.RegisterInstance[*auth.AuthService](rt, svc); err != nil {
		t.Fatalf("register auth service: %v", err)
	}

	regs, err := auth.Registrations(rt)
	if err != nil {
		t.Fatalf("Registrations: %v", err)
	}

	var found bool
	for _, reg := range regs {
		if reg.Name != "system:auth:session:get" {
			continue
		}
		found = true
		if !reg.BootstrapSafe {
			t.Fatal("system:auth:session:get must be BootstrapSafe so a pre-bootstrap probe returns 401, not 403")
		}
	}
	if !found {
		t.Fatal("system:auth:session:get registration not found")
	}
}
