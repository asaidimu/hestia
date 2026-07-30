package runtime

// Stored in the _operation_policy_ collection alongside each operation.

type RateLimitPolicy struct {
	Enabled  bool   `json:"enabled"`
	Identity string `json:"identity"`
	Capacity int64  `json:"capacity"`
	Refill   int64  `json:"refill"`
	Period   int64  `json:"period"` // seconds
}

type ThrottleActionPolicy struct {
	Message string         `json:"message"`
	Input   map[string]any `json:"input"`
}

type ThrottlePolicy struct {
	Limit  int64                `json:"limit"`
	Window int64                `json:"window"` // seconds
	Action *ThrottleActionPolicy `json:"action"`
}
