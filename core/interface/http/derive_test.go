package http

import (
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/abstract"
)

func TestDeriveRoute(t *testing.T) {
	got := DeriveRoute("system:auth:session:create", nil)
	want := "/system/auth/session/create"
	if got != want {
		t.Fatalf("DeriveRoute() = %q, want %q", got, want)
	}
}

func TestIntentToHTTPMethod(t *testing.T) {
	tests := []struct {
		verb abstract.Verb
		want string
	}{
		{abstract.Create, "POST"},
		{abstract.Read, "GET"},
		{abstract.Update, "PATCH"},
		{abstract.Delete, "DELETE"},
		{abstract.Query, "POST"},
		{abstract.Stream, "GET"},
		{abstract.Check, "POST"},
	}
	for _, tt := range tests {
		t.Run(tt.verb.String(), func(t *testing.T) {
			got := IntentToHTTPMethod(tt.verb)
			if got != tt.want {
				t.Errorf("IntentToHTTPMethod(%v) = %q, want %q", tt.verb, got, tt.want)
			}
		})
	}
}

func TestDeriveRouteWithArgs(t *testing.T) {
	got := DeriveRoute("system:auth:session:create", []abstract.ArgumentDefinition{{Name: "id", Type: definition.FieldTypeString}})
	want := "/system/auth/session/create/{id}"
	if got != want {
		t.Fatalf("DeriveRoute() = %q, want %q", got, want)
	}
}
