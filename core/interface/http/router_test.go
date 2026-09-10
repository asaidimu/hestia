package http

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

func TestPathTrie_StaticRoutes(t *testing.T) {
	trie := newPathTrie()
	trie.insert("GET", "/health", routeEntry{handler: handler()})
	trie.insert("POST", "/api/users", routeEntry{handler: handler()})
	trie.insert("DELETE", "/api/users/{id}", routeEntry{handler: handler()})

	_, _, ok := trie.lookup("GET", "/health")
	if !ok {
		t.Error("expected /health to match")
	}
	_, _, ok = trie.lookup("POST", "/api/users")
	if !ok {
		t.Error("expected POST /api/users to match")
	}
	_, _, ok = trie.lookup("DELETE", "/api/users/abc-123")
	if !ok {
		t.Error("expected DELETE /api/users/abc-123 to match")
	}
}

func TestPathTrie_MethodMismatch(t *testing.T) {
	trie := newPathTrie()
	trie.insert("POST", "/api/users", routeEntry{handler: handler()})

	_, _, ok := trie.lookup("GET", "/api/users")
	if ok {
		t.Error("expected method mismatch to not match")
	}
}

func TestPathTrie_PathParams(t *testing.T) {
	trie := newPathTrie()
	trie.insert("GET", "/users/{id}", routeEntry{handler: handler()})
	trie.insert("GET", "/{ns}/items/{key}", routeEntry{handler: handler()})

	_, params, ok := trie.lookup("GET", "/users/42")
	if !ok {
		t.Fatal("expected /users/42 to match")
	}
	if params["id"] != "42" {
		t.Errorf("params[id] = %q, want 42", params["id"])
	}

	_, params, ok = trie.lookup("GET", "/blobs/items/file.txt")
	if !ok {
		t.Fatal("expected /blobs/items/file.txt to match")
	}
	if params["ns"] != "blobs" {
		t.Errorf("params[ns] = %q, want blobs", params["ns"])
	}
	if params["key"] != "file.txt" {
		t.Errorf("params[key] = %q, want file.txt", params["key"])
	}
}

func TestPathTrie_NotFound(t *testing.T) {
	trie := newPathTrie()
	trie.insert("GET", "/api/users", routeEntry{handler: handler()})

	_, _, ok := trie.lookup("GET", "/api/items")
	if ok {
		t.Error("expected unknown route to not match")
	}

	_, _, ok = trie.lookup("GET", "/api/users/extra")
	if ok {
		t.Error("expected shorter pattern with longer path to not match")
	}
}

func TestPathTrie_RootPath(t *testing.T) {
	trie := newPathTrie()
	trie.insert("GET", "/", routeEntry{handler: handler()})

	_, _, ok := trie.lookup("GET", "/")
	if !ok {
		t.Error("expected root path to match")
	}
}

func TestPathTrie_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate route")
		}
	}()
	trie := newPathTrie()
	trie.insert("GET", "/api/users", routeEntry{handler: handler()})
	trie.insert("GET", "/api/users", routeEntry{handler: handler()})
}

func TestPathTrie_StaticBeforeParam(t *testing.T) {
	trie := newPathTrie()
	trie.insert("GET", "/users/{id}", routeEntry{handler: handler()})
	trie.insert("GET", "/users/me", routeEntry{handler: handler()})

	e, params, ok := trie.lookup("GET", "/users/me")
	if !ok {
		t.Fatal("expected /users/me to match")
	}
	if e.handler == nil {
		t.Error("expected non-nil handler")
	}
	if len(params) != 0 {
		t.Errorf("expected empty params, got %v", params)
	}

	e, params, ok = trie.lookup("GET", "/users/42")
	if !ok {
		t.Fatal("expected /users/42 to match")
	}
	if e.handler == nil {
		t.Error("expected non-nil handler")
	}
	if params["id"] != "42" {
		t.Errorf("params[id] = %q, want 42", params["id"])
	}
}

func TestPathTrie_Operation(t *testing.T) {
	trie := newPathTrie()
	trie.insert("GET", "/api/system/health", routeEntry{handler: handler()})

	e, params, ok := trie.lookup("GET", "/api/system/health")
	if !ok {
		t.Fatal("expected /api/system/health to match")
	}
	if e.handler == nil {
		t.Error("expected non-nil handler")
	}
	if len(params) != 0 {
		t.Errorf("expected no params, got %v", params)
	}
}

func TestPathTrie_StreamingBodyFlag(t *testing.T) {
	trie := newPathTrie()
	trie.insert("POST", "/import", routeEntry{handler: handler(), streamingBody: true})
	trie.insert("POST", "/normal", routeEntry{handler: handler()})

	e, _, ok := trie.lookup("POST", "/import")
	if !ok || !e.streamingBody {
		t.Error("expected /import to match with streamingBody=true")
	}
	e, _, ok = trie.lookup("POST", "/normal")
	if !ok || e.streamingBody {
		t.Error("expected /normal to match with streamingBody=false")
	}
}

func TestPathTrie_GreedyParam(t *testing.T) {
	trie := newPathTrie()
	trie.insert("POST", "/api/blobs/blob/upload/{ns}/{key*}", routeEntry{handler: handler()})
	trie.insert("DELETE", "/api/blobs/blob/delete/{ns}/{key*}", routeEntry{handler: handler()})

	// Single segment still matches.
	_, params, ok := trie.lookup("POST", "/api/blobs/blob/upload/admissions/photo.jpg")
	if !ok {
		t.Fatal("expected single-segment key to match")
	}
	if params["ns"] != "admissions" || params["key"] != "photo.jpg" {
		t.Errorf("params = %v, want ns=admissions key=photo.jpg", params)
	}

	// Slashes in the key are consumed greedily.
	_, params, ok = trie.lookup("DELETE", "/api/blobs/blob/delete/admissions/such/keys/are/allowed.txt")
	if !ok {
		t.Fatal("expected slashed key to match")
	}
	if params["key"] != "such/keys/are/allowed.txt" {
		t.Errorf("params[key] = %q, want such/keys/are/allowed.txt", params["key"])
	}

	// Wrong method on a greedy route does not match.
	if _, _, ok = trie.lookup("GET", "/api/blobs/blob/upload/admissions/a/b"); ok {
		t.Error("expected method mismatch to not match")
	}
}

func TestPathTrie_GreedyMustBeTerminal(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on non-terminal greedy param")
		}
	}()
	trie := newPathTrie()
	trie.insert("GET", "/api/{key*}/suffix", routeEntry{handler: handler()})
}

func handler() Handler {
	return func(ctx context.Context, req abstract.Request) (abstract.Response, error) {
		return abstract.Response{Status: 200}, nil
	}
}
