package abstract

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	m.Run()
}

// releaseTracker wraps a data.Document and counts Release invocations so tests
// can assert that Result.Release reaches every owned document.
type releaseTracker struct {
	*data.Document
	released *int
}

func (r *releaseTracker) Release() {
	*r.released++
	r.Document.Release()
}

func TestResultRelease(t *testing.T) {
	ctx := context.Background()
	var released int

	doc := &releaseTracker{Document: data.MustNewDocument(map[string]any{"a": 1}, ctx), released: &released}
	set := data.DocumentSet{
		&releaseTracker{Document: data.MustNewDocument(map[string]any{"b": 2}, ctx), released: &released},
		nil, // nil entries must be skipped, not panic
		&releaseTracker{Document: data.MustNewDocument(map[string]any{"c": 3}, ctx), released: &released},
	}
	page := &Page{
		Documents: data.DocumentSet{
			&releaseTracker{Document: data.MustNewDocument(map[string]any{"d": 4}, ctx), released: &released},
		},
	}
	var blobFreed bool
	blob := Blob{Data: []byte("x"), Release: func() { blobFreed = true }}

	r := &Result{
		Kind:      ResultKindPage,
		Document:  doc,
		Documents: set,
		Page:      page,
		Blob:      blob,
		Metadata:  map[string]any{"k": "v"},
	}

	r.Release()

	if released != 4 {
		t.Errorf("expected 4 documents released, got %d", released)
	}
	if !blobFreed {
		t.Error("expected blob release hook to run")
	}
	if r.Document != nil || r.Documents != nil || r.Page.Documents != nil {
		t.Error("expected pooled fields to be nil after release")
	}
	if r.Metadata["k"] != "v" {
		t.Error("scalar metadata should survive release")
	}

	// Release is idempotent.
	r.Release()
	if released != 4 {
		t.Errorf("second Release should be a no-op, got %d releases", released)
	}

	// A nil result is safe to release.
	var nilR *Result
	nilR.Release()
}

func TestVerbString(t *testing.T) {
	tests := []struct {
		v    Verb
		want string
	}{
		{Create, "CREATE"},
		{Read, "READ"},
		{Update, "UPDATE"},
		{Delete, "DELETE"},
		{Query, "QUERY"},
		{Stream, "STREAM"},
		{Verb(0), ""},
		{Verb(99), ""},
	}
	for _, tc := range tests {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("Verb(%d).String() = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestVerbJSONMarshal(t *testing.T) {
	got, err := json.Marshal(Create)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `"CREATE"` {
		t.Errorf("json.Marshal(Create) = %s, want \"CREATE\"", got)
	}
}

func TestVerbJSONUnmarshal(t *testing.T) {
	var v Verb = Read
	if err := json.Unmarshal([]byte(`"CREATE"`), &v); err != nil {
		t.Fatal(err)
	}
	if v != Create {
		t.Errorf("unmarshal CREATE got %d, want %d", v, Create)
	}

	v = Read
	if err := json.Unmarshal([]byte(`"UNKNOWN"`), &v); err != nil {
		t.Fatal(err)
	}
	if v != Read {
		t.Errorf("unmarshal UNKNOWN changed value to %d, want %d", v, Read)
	}
}

func TestCapabilityStruct(t *testing.T) {
	c := Capability{
		Name:     "test-cap",
		Messages: []MessageRegistration{{Name: "hello"}},
	}
	if c.Name != "test-cap" {
		t.Errorf("Name = %q, want %q", c.Name, "test-cap")
	}
	if len(c.Messages) != 1 || c.Messages[0].Name != "hello" {
		t.Error("Messages field mismatch")
	}
}

func TestMessageRegistrationStruct(t *testing.T) {
	mr := MessageRegistration{
		Name:          "greet",
		Handler:       nil,
		Description:   "Greets the user",
		Intent:        Create,
		Enabled:       true,
		BootstrapSafe: false,
		Internal:      true,
	}
	if mr.Name != "greet" || mr.Description != "Greets the user" || mr.Intent != Create {
		t.Error("basic fields mismatch")
	}
	if !mr.Enabled {
		t.Error("Enabled should be true")
	}
	if mr.BootstrapSafe {
		t.Error("BootstrapSafe should be false")
	}
}
