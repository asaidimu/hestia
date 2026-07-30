package notifications

import "github.com/asaidimu/hestia/core/internal/feature/policies"

func PolicyBindings() []policies.Binding {
	return []policies.Binding{
		{Name: "system:notifications:notification:list", RuleKey: "authenticated", Description: "List notifications"},
		{Name: "system:notifications:notification:read", RuleKey: "authenticated", Description: "Mark notification as read"},
		{Name: "system:notifications:read:all", RuleKey: "authenticated", Description: "Mark all notifications as read"},
		{Name: "system:notifications:unread:count", RuleKey: "authenticated", Description: "Count unread notifications"},
	}
}
