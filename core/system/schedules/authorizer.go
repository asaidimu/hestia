package schedules

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/users"
)

// ScheduleAuthorizer enforces the S-1 authorization model for schedules.
//
// Before the fix, schedule creation was open to any authenticated user while
// the fire path dispatched with a SYSTEM identity through the raw terminal
// dispatcher — no policy evaluation, no audit, no rate limiting. Any user
// could schedule any operation (user creation, policy edits, settings
// writes) and it executed as the system with zero trail.
//
// The model has two halves:
//
//  1. AuthorizeTarget runs at create/update time and rejects schedules whose
//     target operation the caller could not invoke directly under the
//     current policy set.
//
//  2. FireClaims resolves the creator's CURRENT claims at fire time —
//     scopes and tenant are re-read from the user claims cache, the same
//     source session validation and API-key authentication use. Deleted or
//     disabled creators resolve to nil and the fire is skipped (fail
//     closed), and permission changes take effect without rotating the
//     schedule. The fire itself is dispatched through the full middleware
//     chain, so the secure dispatcher re-evaluates the target policy
//     against this identity and audit/rate-limit/tenant all apply.
type ScheduleAuthorizer struct {
	permMgr    runtime.PermissionManager
	accessCtrl iam.AccessController

	// liveUsers is resolved lazily: the user claims cache is initialized
	// after the schedules module during boot.
	liveUsers func() collection.LiveCollection[*users.UserClaims]
}

// NewScheduleAuthorizer wires the S-1 checks. A nil liveUsers provider is
// valid (unit tests); fire-time claims then degrade to a scope-less creator
// identity so "authenticated"-rule targets still fire.
func NewScheduleAuthorizer(permMgr runtime.PermissionManager, accessCtrl iam.AccessController, liveUsers func() collection.LiveCollection[*users.UserClaims]) *ScheduleAuthorizer {
	return &ScheduleAuthorizer{permMgr: permMgr, accessCtrl: accessCtrl, liveUsers: liveUsers}
}

// AuthorizerFromContainer resolves the policy machinery from the DI
// container, returning nil when it is absent (unit tests, minimal boots).
func AuthorizerFromContainer(rt abstract.Container) *ScheduleAuthorizer {
	permMgr, err := abstract.Resolve[runtime.ReloadablePermissionManager](rt)
	if err != nil {
		return nil
	}
	ac, err := abstract.Resolve[iam.AccessController](rt)
	if err != nil {
		return nil
	}
	return NewScheduleAuthorizer(permMgr, ac, func() collection.LiveCollection[*users.UserClaims] {
		lu, _ := abstract.Resolve[collection.LiveCollection[*users.UserClaims]](rt)
		return lu
	})
}

// AuthorizeTarget rejects schedules targeting operations the caller could
// not invoke directly. It resolves the target's policy binding with the
// caller's context (so tenant-scoped policies apply) and evaluates the rule
// against the caller's identity — the same PermMgr/AccessController pair
// the secure dispatcher uses, so the decision matches a direct invocation.
func (a *ScheduleAuthorizer) AuthorizeTarget(ctx context.Context, target string) error {
	if a == nil || a.permMgr == nil || a.accessCtrl == nil {
		// Policy machinery not wired (unit tests) — schema validation in
		// validateScheduleTarget still applies.
		return nil
	}

	msg := dispatch.NewMessage(target, ctx, nil)
	ruleKey, enabled, err := a.permMgr.Resolve(msg)
	if err != nil {
		// Every registered operation carries a policy binding; a resolve
		// failure here means the target is not dispatchable. Fail closed.
		return common.NewSystemError("SCHEDULE_TARGET_FORBIDDEN", fmt.Sprintf("message %q has no resolvable policy binding", target))
	}
	if !enabled {
		return common.NewSystemError("SCHEDULE_TARGET_FORBIDDEN", fmt.Sprintf("message %q is disabled by policy", target))
	}
	if !a.accessCtrl.Can(ctx, ruleKey, nil, nil) {
		return common.NewSystemError("SCHEDULE_TARGET_FORBIDDEN", fmt.Sprintf("schedules may only target operations the caller may invoke directly (rule %q denied)", ruleKey))
	}
	return nil
}

// FireClaims resolves the creator's current identity for a schedule fire.
// It returns nil when the creator cannot be resolved to an active account —
// the caller must then skip the fire (fail closed).
func (a *ScheduleAuthorizer) FireClaims(creatorID, storedTenantID string) *abstract.Claims {
	if creatorID == "" {
		return nil
	}
	if a == nil || a.liveUsers == nil {
		// No claims cache wired (unit tests): restricted creator identity
		// without scopes. "authenticated"-rule targets still fire; scoped
		// rules deny.
		return &abstract.Claims{UserID: creatorID, TenantID: storedTenantID}
	}
	cache := a.liveUsers()
	if cache == nil {
		return nil
	}
	uc, ok := cache.Get(creatorID)
	if !ok || uc == nil {
		// Creator deleted or disabled — the claims cache QueryFunc only
		// returns active users (disabled == -1).
		return nil
	}
	tenantID := uc.TenantID
	if tenantID == "" {
		tenantID = storedTenantID
	}
	return &abstract.Claims{
		UserID:   creatorID,
		Email:    uc.Email,
		Scopes:   uc.Permissions,
		TenantID: tenantID,
	}
}
