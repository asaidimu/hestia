package abstract

import "context"

// BindingPolicyStore manages IAM policies for CRUD operations on dynamic
// resources such as blob namespaces and document collections.
type BindingPolicyStore interface {
	EnsureBinding(ctx context.Context, name, ruleKey string) error
	DeleteBinding(ctx context.Context, name string) error
	ReloadPolicies(ctx context.Context) error
}
