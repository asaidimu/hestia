package dispatch

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/hestia/core/abstract"
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

func TestNewDocumentResult(t *testing.T) {
	doc := &data.Document{}
	r := NewDocumentResult(doc)
	if r.Kind != abstract.ResultKindDocument {
		t.Errorf("Kind = %d, want %d", r.Kind, abstract.ResultKindDocument)
	}
	if r.Document != doc {
		t.Error("Document field mismatch")
	}
}

func TestNewDocumentsResult(t *testing.T) {
	docs := data.DocumentSet{&data.Document{}}
	r := NewDocumentsResult(docs)
	if r.Kind != abstract.ResultKindDocuments {
		t.Errorf("Kind = %d, want %d", r.Kind, abstract.ResultKindDocuments)
	}
	if len(r.Documents) != 1 {
		t.Errorf("len(Documents) = %d, want 1", len(r.Documents))
	}
}

func TestNewPageResult(t *testing.T) {
	page := &abstract.Page{Documents: data.DocumentSet{}, Pagination: nil}
	r := NewPageResult(page)
	if r.Kind != abstract.ResultKindPage {
		t.Errorf("Kind = %d, want %d", r.Kind, abstract.ResultKindPage)
	}
	if r.Page != page {
		t.Error("Page field mismatch")
	}
}

func TestNewBlobResult(t *testing.T) {
	blob := abstract.Blob{Data: []byte("hello"), ContentType: "text/plain"}
	r := NewBlobResult(blob)
	if r.Kind != abstract.ResultKindBlob {
		t.Errorf("Kind = %d, want %d", r.Kind, abstract.ResultKindBlob)
	}
	if string(r.Blob.Data) != "hello" || r.Blob.ContentType != "text/plain" {
		t.Error("Blob field mismatch")
	}
}

func TestNewDocumentChannelResult(t *testing.T) {
	ch := make(chan data.Documenter)
	r := NewDocumentChannelResult(ch)
	if r.Kind != abstract.ResultKindDocumentChannel {
		t.Errorf("Kind = %d, want %d", r.Kind, abstract.ResultKindDocumentChannel)
	}
	if r.DocumentChannel != ch {
		t.Error("DocumentChannel field mismatch")
	}
}

func TestNewBlobChannelResult(t *testing.T) {
	ch := make(chan abstract.Blob)
	r := NewBlobChannelResult(ch)
	if r.Kind != abstract.ResultKindBlobChannel {
		t.Errorf("Kind = %d, want %d", r.Kind, abstract.ResultKindBlobChannel)
	}
	if r.BlobChannel != ch {
		t.Error("BlobChannel field mismatch")
	}
}
