package audit

import "github.com/asaidimu/hestia/internal/app/policies"

func DefaultOperations() []policies.PolicyOperation {
	return []policies.PolicyOperation{
		{Name: "system:audit:log:query", RuleKey: "administrator", Description: "Query access logs"},
		{Name: "system:audit:log:export", RuleKey: "administrator", Description: "Export access logs"},
		{Name: "system:audit:log:stream", RuleKey: "administrator", Description: "Stream access logs in real-time"},
	}
}
