package abstract

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
)

func TestMustNewID(t *testing.T) {
	id := MustNewID()
	if id == "" {
		t.Fatal("MustNewID returned empty string")
	}
	if len(id) != 36 {
		t.Fatalf("MustNewID returned %q (len %d), want 36-char UUID", id, len(id))
	}
}

func TestNewMessage(t *testing.T) {
	ctx := context.Background()
	doc := &data.Document{}
	msg := NewMessage("test.msg", ctx, doc)
	if msg.ID() == "" {
		t.Fatal("NewMessage ID is empty")
	}
	if msg.Name() != "test.msg" {
		t.Fatalf("Name = %q, want %q", msg.Name(), "test.msg")
	}
	if msg.Context() != ctx {
		t.Fatal("Context mismatch")
	}
	if msg.Input() != doc {
		t.Fatal("Input mismatch")
	}
	if msg.InputChannel() != nil {
		t.Fatal("InputChannel should be nil")
	}
	if msg.BlobInputChannel() != nil {
		t.Fatal("BlobInputChannel should be nil")
	}
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

func TestNewDocumentResult(t *testing.T) {
	doc := &data.Document{}
	r := NewDocumentResult(doc)
	if r.Kind != ResultKindDocument {
		t.Errorf("Kind = %d, want %d", r.Kind, ResultKindDocument)
	}
	if r.Document != doc {
		t.Error("Document field mismatch")
	}
}

func TestNewDocumentsResult(t *testing.T) {
	docs := data.DocumentSet{&data.Document{}}
	r := NewDocumentsResult(docs)
	if r.Kind != ResultKindDocuments {
		t.Errorf("Kind = %d, want %d", r.Kind, ResultKindDocuments)
	}
	if len(r.Documents) != 1 {
		t.Errorf("len(Documents) = %d, want 1", len(r.Documents))
	}
}

func TestNewPageResult(t *testing.T) {
	page := &Page{Documents: data.DocumentSet{}, Pagination: nil}
	r := NewPageResult(page)
	if r.Kind != ResultKindPage {
		t.Errorf("Kind = %d, want %d", r.Kind, ResultKindPage)
	}
	if r.Page != page {
		t.Error("Page field mismatch")
	}
}

func TestNewBlobResult(t *testing.T) {
	blob := Blob{Data: []byte("hello"), ContentType: "text/plain"}
	r := NewBlobResult(blob)
	if r.Kind != ResultKindBlob {
		t.Errorf("Kind = %d, want %d", r.Kind, ResultKindBlob)
	}
	if string(r.Blob.Data) != "hello" || r.Blob.ContentType != "text/plain" {
		t.Error("Blob field mismatch")
	}
}

func TestNewDocumentChannelResult(t *testing.T) {
	ch := make(chan *data.Document)
	r := NewDocumentChannelResult(ch)
	if r.Kind != ResultKindDocumentChannel {
		t.Errorf("Kind = %d, want %d", r.Kind, ResultKindDocumentChannel)
	}
	if r.DocumentChannel != ch {
		t.Error("DocumentChannel field mismatch")
	}
}

func TestNewBlobChannelResult(t *testing.T) {
	ch := make(chan Blob)
	r := NewBlobChannelResult(ch)
	if r.Kind != ResultKindBlobChannel {
		t.Errorf("Kind = %d, want %d", r.Kind, ResultKindBlobChannel)
	}
	if r.BlobChannel != ch {
		t.Error("BlobChannel field mismatch")
	}
}

type testModule struct{}

func (m *testModule) Name() string                                                      { return "test" }
func (m *testModule) Setup(_ context.Context, _ base.Persistence) error { return nil }
func (m *testModule) Capabilities() []Capability                                         { return nil }

func TestModuleInterface(t *testing.T) {
	var _ Module = (*testModule)(nil)
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
