package boot

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func newFirstRunConfig(t *testing.T) *runtime.Config {
	t.Helper()
	dir := t.TempDir()
	cfg := runtime.DefaultConfig()
	cfg.DataDir = dir
	cfg.DBPath = filepath.Join(dir, "test.db")
	cfg.LogPath = filepath.Join(dir, "server.log")
	cfg.BlobsDir = filepath.Join(dir, "blobs")
	return cfg
}

// captureStdout swaps os.Stdout for the duration of fn and returns everything
// written to it. UserOutput captures the os.Stdout variable at construction,
// so the swap must happen before boot.New.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}

func TestFirstRunPrintsEphemeralKey(t *testing.T) {
	out := captureStdout(t, func() {
		cfg := newFirstRunConfig(t)
		app := New(cfg)
		if err := app.Boot(context.Background(), dispatch.SystemOptions{}); err != nil {
			t.Fatalf("boot: %v", err)
		}
		app.Close()
	})

	if !strings.Contains(out, "X-API-Key:") {
		t.Fatalf("expected ephemeral API key on first-run boot output, got:\n%s", out)
	}
	if !strings.Contains(out, "not bootstrapped") {
		t.Fatalf("expected 'not bootstrapped' hint on first-run boot output, got:\n%s", out)
	}
}

func TestFirstRunSuppressesKeyWhenBootstrapped(t *testing.T) {
	out := captureStdout(t, func() {
		cfg := newFirstRunConfig(t)
		app := New(cfg)
		if err := app.Boot(context.Background(), dispatch.SystemOptions{ForceBootstrapped: true}); err != nil {
			t.Fatalf("boot: %v", err)
		}
		app.Close()
	})

	if strings.Contains(out, "X-API-Key:") {
		t.Fatalf("expected no ephemeral API key once bootstrapped, got:\n%s", out)
	}
}