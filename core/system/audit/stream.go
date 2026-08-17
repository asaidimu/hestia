package audit

import (
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/audit/model"
	"github.com/asaidimu/hestia/core/system/policies"
)

// StreamRegistration returns the system:audit:log:stream registration. The
// generator skips streaming annotations until dispatch.HandleInputStream
// lands, so this registration is hand-written.
func StreamRegistration(persist persistence.Persistence) []abstract.MessageRegistration {
	return []abstract.MessageRegistration{
		{Name: "system:audit:log:stream", Handler: logStreamHandler(persist), Description: "Stream audit log entries in real-time", Enabled: true, Intent: abstract.Stream, Input: runtime.Input{Schema: model.LogStreamInputSchema()}, Output: model.LogStreamOutputSchema()},
	}
}

// StreamPolicyBinding binds system:audit:log:stream to the administrator rule.
func StreamPolicyBinding() []policies.Binding {
	return []policies.Binding{
		{Name: "system:audit:log:stream", RuleKey: "administrator", Description: "Stream access logs in real-time"},
	}
}
