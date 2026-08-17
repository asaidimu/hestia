package runtime

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
)

// TestSecureDispatcher_NoPolicyIsDeniedByDefault verifies secure-by-default:
// a message with no registered policy/binding is denied, not silently allowed.
func TestSecureDispatcher_NoPolicyIsDeniedByDefault(t *testing.T) {
	permMgr := NewMapPermissionManager()
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: adminContext(), name: "unknown:no:policy:msg"})
	if err == nil {
		t.Fatal("expected denial for message with no registered policy, got nil")
	}
	if !strings.Contains(err.Error(), "PERMISSION_NOT_REGISTERED") {
		t.Fatalf("expected permission-not-registered error, got: %v", err)
	}
}

// TestSecureDispatcher_PolicyDisabledDeniesNonAnonymous documents that a
// disabled policy is enforced as a denial (and as auth-required for anonymous).
func TestSecureDispatcher_PolicyDisabledDenies(t *testing.T) {
	// A "disabled" policy surfaces through Resolve's enabled=false.
	permMgr := disabledScopeManager{name: "sys:svc:op:run"}
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: adminContext(), name: "sys:svc:op:run"})
	if err == nil {
		t.Fatal("expected denial for disabled policy, got nil")
	}
	if !strings.Contains(err.Error(), "ERR_ACCESS_DENIED") {
		t.Fatalf("expected access-denied error for authenticated caller, got: %v", err)
	}

	_, err = disp.Send(testMessage{ctx: anonymousContext(), name: "sys:svc:op:run"})
	if err == nil {
		t.Fatal("expected error for anonymous, got nil")
	}
	if !strings.Contains(err.Error(), "AUTH_REQUIRED") {
		t.Fatalf("expected auth-required error for anonymous, got: %v", err)
	}
}

// TestSecureDispatcher_SystemIdentityBypassesIsRoot documents the root
// equivalence of the system scope: any identity carrying a "system:*"
// permission skips SecureDispatcher entirely. Never grant it casually.
func TestSecureDispatcher_SystemIdentityBypassesIsRoot(t *testing.T) {
	permMgr := NewMapPermissionManager()
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{})

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	// No rules, no registered scope, yet system identity sails through.
	_, err := disp.Send(testMessage{ctx: systemContext(), name: "anything:at:all:goes"})
	if err != nil {
		t.Fatalf("system identity must bypass authorization, got: %v", err)
	}
}

// disabledScopeManager is a PermissionManager that reports a message as
// registered but disabled.
type disabledScopeManager struct {
	name string
}

func (m disabledScopeManager) Resolve(msg abstract.Message) (string, bool, error) {
	if msg.Name() == m.name {
		return "administrator", false, nil
	}
	return "", false, ErrPermissionNotRegistered
}

func (m disabledScopeManager) ListCapabilities() []CapabilityMetadata { return nil }