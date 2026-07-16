package operations_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/operations"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

type testMessage struct {
	ctx context.Context
}

func (m testMessage) ID() string                            { return "" }
func (m testMessage) Name() string                          { return "test" }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() *data.Document                  { return data.MustNewDocument(nil, m.ctx) }
func (m testMessage) InputChannel() <-chan *data.Document    { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

func TestDefaultOperations(t *testing.T) {
	ops := operations.DefaultOperations()
	if len(ops) == 0 {
		t.Fatal("DefaultOperations returned empty list")
	}
	for _, op := range ops {
		if op.Name == "" {
			t.Error("DefaultOperations contains an operation with empty Name")
		}
	}
}

func TestSystemStatusHandler(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
	seed := operations.NewSeedModel(p)

	if err := seed.Set(ctx, "bootstrapped", "true"); err != nil {
		t.Fatalf("seed.Set failed: %v", err)
	}

	handler := operations.NewSystemStatusHandler(func() bool {
		val, err := seed.Get(ctx, "bootstrapped")
		if err != nil || val == "" {
			return false
		}
		return val == "true"
	})

	msg := testMessage{ctx: ctx}
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Document == nil {
		t.Fatal("result.Document is nil")
	}

	bootstrapped, _ := result.Document.GetOr("bootstrapped", nil).(bool)
	if !bootstrapped {
		t.Errorf("bootstrapped = %v, want true", bootstrapped)
	}

	ok, _ := result.Document.GetOr("ok", nil).(bool)
	if !ok {
		t.Errorf("ok = %v, want true", ok)
	}
}

func TestDocumentationHandler(t *testing.T) {
	ctx := context.Background()

	regs := []abstract.MessageRegistration{
		{
			Name:          "system:test:handler",
			Description:   "A test handler",
			Enabled:       true,
			Intent:        abstract.Read,
			BootstrapSafe: true,
		},
	}

	handler := operations.NewDocumentationHandler(&regs)
	msg := testMessage{ctx: ctx}
	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Documents) == 0 {
		t.Fatal("result.Documents is empty")
	}

	doc := result.Documents[0]
	name, _ := doc.GetOr("name", nil).(string)
	if name != "system:test:handler" {
		t.Errorf("name = %v, want system:test:handler", name)
	}
}

func TestSeedModelSetAndGet(t *testing.T) {
	ctx := context.Background()
	p := persistest.NewPersistence(t)
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
