package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// LogQueryInput is the input for system:collections:audit_log:query and :export. The
// payload is an opaque QDSL document passed through to the collection query
// handler verbatim.
type LogQueryInput struct {
	Payload map[string]any `input:"payload"`
}

func LogQueryInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[LogQueryInput]("input", true)
}

// LogStreamInput is the input for system:audit:log:stream. The stream handler
// does not read any input fields.
type LogStreamInput struct{}

func LogStreamInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[LogStreamInput]("input", true)
}