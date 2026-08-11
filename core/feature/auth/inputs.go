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
	CallerID string `input:"payload.caller_id"`
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
