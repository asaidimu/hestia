package model

import (
	"encoding/json"
	"testing"
)

func TestProbeSchemas(t *testing.T) {
	tests := map[string]func() interface{ Fields() interface{} }{
		"UserOutput": func() interface{ Fields() interface{} } { return nil },
	}
	_ = tests
	for _, s := range []*struct{ name string }{} {
		_ = s
	}
	b, err := json.MarshalIndent(UserOutputSchema(), "", "  ")
	t.Logf("marshal err=%v\n%s", err, b)
}
