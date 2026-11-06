package boot

import (
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type fakeModule struct {
	name     string
	deps     []string
	startErr error
	starts   atomic.Int32
	stops    atomic.Int32
}

func (m *fakeModule) Name() string { return m.name }

func (m *fakeModule) Setup(context.Context, abstract.Container) error { return nil }

func (m *fakeModule) Capabilities(abstract.Container) ([]abstract.Capability, error) {
	return nil, nil
}

func (m *fakeModule) Dependencies() []string { return m.deps }

func (m *fakeModule) Start(context.Context) error {
	m.starts.Add(1)
	return m.startErr
}

func (m *fakeModule) Stop(context.Context) error {
	m.stops.Add(1)
	return nil
}

func (m *fakeModule) Health(context.Context) any { return nil }

func newLifecycleApp(t *testing.T, modules ...abstract.Module) *Application {
	t.Helper()
	cfg := &runtime.Config{
		LogPath:       filepath.Join(t.TempDir(), "hestia-lifecycle.log"),
		LogMaxSize:    100,
		LogMaxAge:     1,
		LogMaxBackups: 1,
	}
	return &Application{
		Modules: modules,
		Loggers: NewLoggers(cfg),
	}
}

func TestTopoSortRejectsMissingDependency(t *testing.T) {
	_, err := topoSort([]abstract.Module{
		&fakeModule{name: "a"},
		&fakeModule{name: "b", deps: []string{"ghost"}},
	})
	if err == nil {
		t.Fatal("expected an error for a missing dependency")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("error should name the missing dependency, got: %v", err)
	}
}

func TestTopoSortOrdersDependencies(t *testing.T) {
	a := &fakeModule{name: "a"}
	b := &fakeModule{name: "b", deps: []string{"a"}}
	c := &fakeModule{name: "c", deps: []string{"b"}}

	sorted, err := topoSort([]abstract.Module{c, a, b})
	if err != nil {
		t.Fatalf("topoSort: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("got %d modules, want 3", len(sorted))
	}
	want := []string{"a", "b", "c"}
	for i, m := range sorted {
		if m.Name() != want[i] {
			t.Fatalf("position %d = %q, want %q", i, m.Name(), want[i])
		}
	}
}

func TestStartPropagatesModuleErrors(t *testing.T) {
	ok := &fakeModule{name: "ok"}
	bad := &fakeModule{name: "bad", startErr: context.Canceled}
	app := newLifecycleApp(t, ok, bad)
	defer app.Close()

	err := app.Start()
	if err == nil {
		t.Fatal("expected Start to return the module failure")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Fatalf("error should name the failing module, got: %v", err)
	}
	if ok.starts.Load() != 1 || bad.starts.Load() != 1 {
		t.Fatalf("expected both modules to start, got ok=%d bad=%d", ok.starts.Load(), bad.starts.Load())
	}
}

func TestCloseRunsShutdownAfterStart(t *testing.T) {
	m := &fakeModule{name: "m"}
	app := newLifecycleApp(t, m)

	if err := app.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	app.Close()

	if m.stops.Load() != 1 {
		t.Fatalf("Close should stop the module once, got %d", m.stops.Load())
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	m := &fakeModule{name: "m"}
	app := newLifecycleApp(t, m)

	if err := app.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	app.Close()
	app.Close()

	if m.stops.Load() != 1 {
		t.Fatalf("Close called twice should stop the module once, got %d", m.stops.Load())
	}
}

func TestCloseBeforeStartDoesNotStop(t *testing.T) {
	m := &fakeModule{name: "m"}
	app := newLifecycleApp(t, m)

	app.Close()

	if m.stops.Load() != 0 {
		t.Fatalf("Close before Start should not stop modules, got %d", m.stops.Load())
	}
}

func TestCloseAfterShutdownStopsOnce(t *testing.T) {
	m := &fakeModule{name: "m"}
	app := newLifecycleApp(t, m)

	if err := app.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	app.Close()

	if m.stops.Load() != 1 {
		t.Fatalf("Shutdown followed by Close should stop the module once, got %d", m.stops.Load())
	}
}
