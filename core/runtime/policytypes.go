// @note #arch-20260821-013 issue status=open priority=P2 tags=#arch,#placement : Domain types in leaf package
//
// runtime is a leaf package but contains domain types that other packages depend on:
//
// - Policy types (RateLimitPolicy, ThrottlePolicy, ThrottleActionPolicy) - stored
//   in the _operation_policy_ collection alongside each operation
// - Rate limit types (RateLimitError, RateLimitStore, MatchIdentity, RateLimitLookup)
// - Capability types (CapabilityItem, CapabilitiesDocument)
// - Permission types (PermissionManager, ReloadablePermissionManager, CapabilityMetadata)
// - Metadata types (RateLimitMeta)
// - Audit context keys (AuditActorIDKey, etc.)
//
// These types are used extensively throughout the codebase (52+ files import runtime).
// This creates a tight coupling between the runtime package and all other packages.
//
// Resolution: Move domain types to proper packages:
// - RateLimitPolicy/ThrottlePolicy -> operations/policy or similar model package
// - PermissionManager -> permissions package
// - RateLimitStore -> ratestore package
// - CapabilityItem/CapabilitiesDocument -> capabilities package
package runtime

// Stored in the _operation_policy_ collection alongside each operation.

type RateLimitPolicy struct {
	Enabled  bool   `json:"enabled"`
	Identity string `json:"identity"`
	Capacity int64  `json:"capacity"`
	Refill   int64  `json:"refill"`
	Period   int64  `json:"period"` // seconds
}

type ThrottleActionPolicy struct {
	Message string         `json:"message"`
	Input   map[string]any `json:"input"`
}

type ThrottlePolicy struct {
	Limit  int64                 `json:"limit"`
	Window int64                 `json:"window"` // seconds
	Action *ThrottleActionPolicy `json:"action"`
}
