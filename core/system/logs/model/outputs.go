package model

import "github.com/asaidimu/go-anansi/v8/core/document"

// LogEntryView is the wire shape of a single log entry.
type LogEntryView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Level                  string         `json:"level" anansi:"level"`
	TS                     float64        `json:"ts" anansi:"ts"`
	Caller                 string         `json:"caller" anansi:"caller"`
	Msg                    string         `json:"msg" anansi:"msg"`
	Fields                 map[string]any `json:"fields,omitempty" anansi:"fields,omitempty"`
	// Extra holds every top-level JSON key that isn't level/ts/caller/msg/fields.
	// This captures zap structured fields like operation, duration, request_id,
	// client_ip, user_id, email, error, stacktrace, etc.
	Extra map[string]any `json:"extra,omitempty"`
}

// LogListDocument is the body of a log list response.
type LogListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Entries                []LogEntryView `anansi:"entries"`
	Total                  int            `anansi:"total"`
	HasMore                bool           `anansi:"has_more"`
}

// LogListOutput is the envelope declaring the log list schema.
type LogListOutput struct {
	Document LogListDocument `anansi:"document"`
}

// LogStreamOutput is the envelope for a single streamed log entry.
type LogStreamOutput struct {
	Document LogEntryView `anansi:"document"`
}
