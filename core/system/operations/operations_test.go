package operations_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/operations"
	"github.com/asaidimu/hestia/core/internal/testutil"
)

type testMessage struct {
	ctx   context.Context
	input data.Documenter
}

func (m testMessage) ID() string                             { return "" }
func (m testMessage) Name() string                           { return "test" }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() data.Documenter                  { return m.input }
func (m testMessage) InputChannel() <-chan abstract.StreamItem    { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m testMessage) TenantID() string                       { return "" }
func (m testMessage) TraceID() string                        { return "" }
func (m testMessage) RequestID() string                      { return "" }
func (m testMessage) SourceIP() string                       { return "" }
func (m testMessage) UserAgent() string                      { return "" }
func (m testMessage) ResourceID() string                     { return "" }
func (m testMessage) SessionID() string                      { return "" }

func TestPolicyBindings(t *testing.T) {
	bindings := operations.Policies()
	if len(bindings) == 0 {
		t.Fatal("Policies() returned empty list")
	}
	for _, b := range bindings {
		if b.Name == "" {
			t.Error("Policies() contains a binding with empty Name")
		}
	}
}

func TestSeedModelSetAndGet(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	seed := operations.NewSeedModel(p)

	if err := seed.Set(ctx, "mykey", "myvalue"); err != nil {
		t.Fatalf("seed.Set failed: %v", err)
	}

	got, err := seed.Get(ctx, "mykey")
	if err != nil {
		t.Fatalf("seed.Get failed: %v", err)
	}
	if got != "myvalue" {
		t.Errorf("seed.Get = %q, want %q", got, "myvalue")
	}

	if err := seed.Set(ctx, "mykey", "newvalue"); err != nil {
		t.Fatalf("seed.Set (overwrite) failed: %v", err)
	}
	got, err = seed.Get(ctx, "mykey")
	if err != nil {
		t.Fatalf("seed.Get after overwrite failed: %v", err)
	}
	if got != "newvalue" {
		t.Errorf("seed.Get after overwrite = %q, want %q", got, "newvalue")
	}
}
