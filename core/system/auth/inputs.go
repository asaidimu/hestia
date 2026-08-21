// @note #cruft-20260821-001 issue status=open priority=P1 tags=#cruft,#duplicate : Duplicate input types in auth/inputs.go and auth/model/inputs.go
// @see #8uuufn
//
// This file defines LoginInput, DeleteSessionInput, PasswordResetInput,
// PasswordConfirmInput, BootstrapPasswordInput, and ElevateInput — all of
// which are duplicated verbatim in auth/model/inputs.go. The generated
// registrations in auth/registrations.go use the model/ versions.
//
// Additionally, the schema functions (LoginInputSchema, etc.) at the bottom
// are dead code — the registrations use dispatch.SchemaFromTypeWithTag directly.
//
// Resolution: delete this file entirely; all consumers should import from
// auth/model. The schema functions are unnecessary.
package auth

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

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

func LoginInputSchema() *definition.Schema             { return dispatch.SchemaFromTypeWithTag[LoginInput]("input", true) }
func DeleteSessionInputSchema() *definition.Schema     { return dispatch.SchemaFromTypeWithTag[DeleteSessionInput]("input", true) }
func PasswordResetInputSchema() *definition.Schema     { return dispatch.SchemaFromTypeWithTag[PasswordResetInput]("input", true) }
func PasswordConfirmInputSchema() *definition.Schema   { return dispatch.SchemaFromTypeWithTag[PasswordConfirmInput]("input", true) }
func BootstrapPasswordInputSchema() *definition.Schema { return dispatch.SchemaFromTypeWithTag[BootstrapPasswordInput]("input", true) }
func ElevateInputSchema() *definition.Schema           { return dispatch.SchemaFromTypeWithTag[ElevateInput]("input", true) }
