package runtime

import (
	"testing"

	"github.com/asaidimu/hestia/core/runtime/di"
)

func TestRuntimeResolvesDeps(t *testing.T) {
	rt := NewRuntime()

	rt.RegisterInstance[string]("hello")
	if err := rt.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := rt.MustResolve[string](); got != "hello" {
		t.Fatalf("MustResolve[string] = %q, want %q", got, "hello")
	}
}

func TestRuntimePromotesContainer(t *testing.T) {
	rt := NewRuntime()
	if rt.Container == nil {
		t.Fatal("Runtime should embed a non-nil di.Container")
	}
	var _ *di.Container = rt.Container
}
