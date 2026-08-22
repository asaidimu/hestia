package schedules

type ScheduleCreateInput struct {
	UserID   string         `input:"payload.user_id"`
	Message  string         `input:"payload.message"`
	Input    map[string]any `input:"payload.input"`
	Cron     string         `input:"payload.cron"`
	Disabled bool           `input:"payload.disabled"`
}

type ScheduleGetInput struct {
	ID string `input:"arguments.id"`
}

type ScheduleUpdateInput struct {
	ID       string         `input:"arguments.id"`
	Message  string         `input:"payload.message"`
	Input    map[string]any `input:"payload.input"`
	Cron     string         `input:"payload.cron"`
	Disabled bool           `input:"payload.disabled"`
}

type ScheduleDeleteInput struct {
	ID string `input:"arguments.id"`
}
