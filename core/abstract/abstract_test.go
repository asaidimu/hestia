package abstract

import (
	"encoding/json"
	"testing"
)

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
