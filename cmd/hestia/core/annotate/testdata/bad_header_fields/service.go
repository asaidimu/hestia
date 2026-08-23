package users

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
)

// CallbackInput declares a transport-context field: the HTTP interface lifts
// the X-Session-Id header into it.
type CallbackInput struct {
	SessionID string `input:"context.session_id"`
}

// UsersService is the service under test.
type UsersService struct{}

// AckCallback uses a deprecated header_fields attribute; codegen must fail
// loudly instead of silently dropping the binding.
//
// @hestia.register(
//   name="system:users:callback:ack",
//   intent="create",
//   rule="authenticated",
//   header_fields="X-Session-ID=session_id",
// )
func (s *UsersService) AckCallback(ctx context.Context, msg abstract.Message, input *CallbackInput) error {
	return nil
}
