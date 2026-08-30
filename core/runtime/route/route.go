// Package route holds transport-neutral route derivation for message
// registrations (audit A-3). The intent-to-method/path math describes the
// OPERATION, not the HTTP transport — the HTTP transport, the desktop
// (wails) adapter, the docs listing, and the route codegen all need the
// same triplet. It used to live in core/interface/http, which forced every
// non-HTTP consumer to import the HTTP package's guts.
package route

import (
	"fmt"
	"strings"

	"github.com/asaidimu/hestia/core/abstract"
)

// DeriveRoute converts a message name (system:blobs:ns:download) into a
// URL path with argument placeholders (/system/blobs/ns/download/{ns}).
func DeriveRoute(name string, arguments []abstract.ArgumentDefinition) string {
	parts := strings.Split(name, ":")
	path := "/" + strings.Join(parts, "/")
	for _, arg := range arguments {
		path += fmt.Sprintf("/{%s}", arg.Name)
	}
	return path
}

// IntentToHTTPMethod maps an operation intent to its HTTP verb.
func IntentToHTTPMethod(verb abstract.Verb) string {
	switch verb {
	case abstract.Create:
		return "POST"
	case abstract.Read:
		return "GET"
	case abstract.Update:
		return "PATCH"
	case abstract.Delete:
		return "DELETE"
	case abstract.Query:
		return "POST"
	case abstract.Stream:
		return "GET"
	case abstract.Check:
		return "POST"
	}
	return "GET"
}

// IntentToHTTPPath maps an intent to its path form (identity for now; kept
// so the intent participates in route derivation at one seam).
func IntentToHTTPPath(verb abstract.Verb, path string) string {
	return path
}
