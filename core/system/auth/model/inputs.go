// @note #cruft-20260821-013 observation resolved priority=P3 tags=#cruft,#note : Canonical input types for auth
// @see #cruft-20260821-001
// Duplicate auth/inputs.go deleted. auth/model/inputs.go is now the sole source.
//
// This is the canonical location for auth input types. The duplicate
// definitions in auth/inputs.go should be deleted (see #cruft-20260821-001).
// The generated registrations and service methods import from this package.
package model

type LoginInput struct {
	Email    string `input:"payload.email"`
	Password string `input:"payload.password"`
}

type DeleteSessionInput struct{}

type PasswordResetInput struct {
	Email string `input:"payload.email"`
}

type PasswordConfirmInput struct {
	Token    string `input:"payload.token"`
	Password string `input:"payload.password"`
}

type BootstrapPasswordInput struct {
	Email    string `input:"payload.email"`
	Password string `input:"payload.password"`
}

type ElevateInput struct {
	Email    string `input:"payload.email"`
	Password string `input:"payload.password"`
}
