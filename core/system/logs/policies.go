package logs

import "github.com/asaidimu/hestia/core/system/policies"

// Policies returns the binding policies for the logs feature.
func Policies() []policies.Binding {
	return []policies.Binding{
		{Name: "system:logs:list", RuleKey: "administrator", Description: "Query application logs"},
		{Name: "system:logs:stream", RuleKey: "administrator", Description: "Stream live log entries"},
	}
}
