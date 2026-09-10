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

// UserQueryInput is the input for system:collections:user:query. The
// payload is an opaque QDSL document passed through to the collection query
// handler verbatim (same shape as the audit LogQueryInput). Typed filter
// fields must NOT be declared here: the input pool strips unknown fields,
// so anything but a free-form map silently drops filters/pagination.
type UserQueryInput struct {
	Payload map[string]any `input:"payload"`
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
