package model

import "github.com/asaidimu/go-anansi/v8/core/document"

// LogEntryView is the wire shape of a single log entry.
type LogEntryView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Level                  string         `anansi:"level"`
	TS                     float64        `anansi:"ts"`
	Caller                 string         `anansi:"caller"`
	Msg                    string         `anansi:"msg"`
	Fields                 map[string]any `anansi:"fields,omitempty"`
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
