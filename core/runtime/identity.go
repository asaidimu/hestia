package runtime

import (
	"context"

	"github.com/asaidimu/go-iam/v2/iam"
)

// GetIdentityProperty extracts a typed property from the IAM identity in the
// context. Returns the value and true if the identity exists and the property
// has the expected type, zero value and false otherwise.
func GetIdentityProperty[T any](ctx context.Context, key string) (T, bool) {
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		var zero T
		return zero, false
	}
	props, ok := ident.Properties.(map[string]any)
	if !ok {
		var zero T
		return zero, false
	}
	v, ok := props[key].(T)
	return v, ok
}

// GetIdentityProperties returns the raw properties map from the IAM identity
// in the context, or nil if no identity is present.
func GetIdentityProperties(ctx context.Context) map[string]any {
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		return nil
	}
	props, _ := ident.Properties.(map[string]any)
	return props
}

// GetUserID is a convenience wrapper for extracting user_id from the identity.
func GetUserID(ctx context.Context) string {
	uid, _ := GetIdentityProperty[string](ctx, "user_id")
	return uid
}

// GetTokenID is a convenience wrapper for extracting token_id from the identity.
func GetTokenID(ctx context.Context) string {
	tid, _ := GetIdentityProperty[string](ctx, "token_id")
	return tid
}

// IsAnonymous reports whether the context carries no identity or has an empty
// user_id, indicating an unauthenticated request.
func IsAnonymous(ctx context.Context) bool {
	return GetUserID(ctx) == ""
}
