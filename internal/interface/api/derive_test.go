package api

import (
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/app/core/registration"
)

func TestDeriveRoute(t *testing.T) {
	got := DeriveRoute("system:auth:session:create", nil)
	want := "/system/auth/session"
	if got != want {
		t.Fatalf("DeriveRoute() = %q, want %q", got, want)
	}
}

func TestIntentToHTTPMethod(t *testing.T) {
	tests := []struct {
		verb registration.Verb
		want string
	}{
		{registration.Create, "POST"},
		{registration.Read, "GET"},
		{registration.Update, "PATCH"},
		{registration.Delete, "DELETE"},
		{registration.Query, "POST"},
		{registration.Stream, "GET"},
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
	got := DeriveRoute("system:auth:session:create", []abstract.ArgDef{{Name: "id", Type: definition.FieldTypeString}})
	want := "/system/auth/session/{id}"
	if got != want {
		t.Fatalf("DeriveRoute() = %q, want %q", got, want)
	}
}
