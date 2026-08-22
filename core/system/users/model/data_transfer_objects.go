package model

type UserGetInput struct {
	UserID string `input:"arguments.user_id"`
}

type UserChangePasswordInput struct {
	UserID  string `input:"arguments.user_id"`
	Current string `input:"payload.current"`
	New     string `input:"payload.new"`
}

type UserDeleteInput struct {
	UserID string `input:"arguments.user_id"`
}

type UserUpdateInput struct {
	UserUpdate
}

type UserRegisterInput struct {
	UserRegister
}

type UserQueryInput struct {
	Username string `input:"payload.username,omitempty"`
	Limit    int    `input:"payload.limit,omitempty"`
	Cursor   string `input:"payload.cursor,omitempty"`
}

type UserOutput struct {
	Document UserPublic `anansi:"document"`
}

type UserQueryOutput struct {
	Documents []UserPublic `anansi:"page.documents"`
	Total     int          `anansi:"page.pagination.total"`
	Cursor    string       `anansi:"page.pagination.cursor"`
	Limit     int          `anansi:"page.pagination.limit"`
}

type MessageOutput struct {
	Message string `anansi:"message"`
}
