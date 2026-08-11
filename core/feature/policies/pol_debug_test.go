package policies_test

import (
	"fmt"
	"testing"

	"github.com/asaidimu/hestia/core/feature/policies"
	"github.com/asaidimu/hestia/core/internal/testutil"
)

func TestDebugRefCase(t *testing.T) {
	payload := map[string]any{
		"rule": map[string]any{"type": "ref", "name": "administrator"},
		"context": map[string]any{
			"identity":    map[string]any{"permissions": []string{"administrator"}},
			"resource":    map[string]any{},
			"environment": map[string]any{},
		},
	}
	doc := testutil.InputDoc(t, policies.PolicyValidateInputSchema(), `{"payload":{"rule":{"type":"ref","name":"administrator"},"context":{"identity":{"permissions":["administrator"]},"resource":{},"environment":{}}}}`)
	p := doc.GetOr("payload", nil)
	fmt.Printf("payload type=%T\n", p)
	if m, ok := p.(map[string]any); ok {
		fmt.Printf("rule=%#v (%T)\n", m["rule"], m["rule"])
		fmt.Printf("context=%#v (%T)\n", m["context"], m["context"])
		if c, ok := m["context"].(map[string]any); ok {
			fmt.Printf("identity=%#v\n", c["identity"])
		}
	}
	_ = payload
}
