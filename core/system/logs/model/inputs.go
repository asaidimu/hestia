package model

// LogQueryInput is the input for querying historical logs.
type LogQueryInput struct {
	Level  string `input:"payload.level"`
	From   string `input:"payload.from"`
	To     string `input:"payload.to"`
	Search string `input:"payload.search"`
	Limit  int    `input:"payload.limit"`
	Offset int    `input:"payload.offset"`
}

// LogStreamInput is the input for streaming live logs.
type LogStreamInput struct {
	Level string `input:"payload.level"`
}
