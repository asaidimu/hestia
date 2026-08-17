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
	CallerID string `input:"payload.caller_id"`
}

type ElevateInput struct {
	Email    string `input:"payload.email"`
	Password string `input:"payload.password"`
}